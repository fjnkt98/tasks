package e2e_test

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/presentation"
	"github.com/fjnkt98/tasks/internal/testutil"
)

type env struct {
	db     *sql.DB
	server *httptest.Server
	client *http.Client
}

func newEnv(t *testing.T) *env {
	t.Helper()

	t.Setenv("USE_SECURE_COOKIE", "false")

	db := testutil.NewDB(t)
	server := httptest.NewServer(presentation.NewHandler(db))
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client := server.Client()
	client.Jar = jar
	client.Timeout = 5 * time.Second

	return &env{db: db, server: server, client: client}
}

func discard(t *testing.T, body io.ReadCloser) {
	t.Helper()

	_, err := io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
}

func postForm(t *testing.T, e *env, path string, values url.Values) *http.Response {
	t.Helper()

	res, err := e.client.PostForm(e.server.URL+path, values)
	require.NoError(t, err)
	return res
}

func doForm(t *testing.T, e *env, method, path string, values url.Values) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(
		t.Context(),
		method,
		e.server.URL+path,
		strings.NewReader(values.Encode()),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := e.client.Do(req)
	require.NoError(t, err)
	return res
}

func signup(t *testing.T, e *env, username, password string) {
	t.Helper()

	res := postForm(t, e, "/account/signup", url.Values{
		"username":         []string{username},
		"password":         []string{password},
		"confirm-password": []string{password},
	})
	discard(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "/account/signup/success", res.Request.URL.Path)
}

func signin(t *testing.T, e *env, username, password string) *http.Response {
	t.Helper()

	return postForm(t, e, "/account/signin", url.Values{
		"username": []string{username},
		"password": []string{password},
	})
}

func authCookie(t *testing.T, e *env) *http.Cookie {
	t.Helper()

	u, err := url.Parse(e.server.URL)
	require.NoError(t, err)
	cookies := e.client.Jar.Cookies(u)
	i := slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == presentation.AuthCookieName
	})
	if i < 0 {
		return nil
	}
	return cookies[i]
}

func userCount(t *testing.T, db *sql.DB, name string) int {
	t.Helper()

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users WHERE name = ?`, name).Scan(&n))
	return n
}

func sessionCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n))
	return n
}

func taskCount(t *testing.T, db *sql.DB, title string) int {
	t.Helper()

	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE title = ?`, title).Scan(&n))
	return n
}

func createTask(t *testing.T, e *env, title string) int {
	t.Helper()

	res := postForm(t, e, "/tasks", url.Values{"title": []string{title}})
	discard(t, res.Body)
	require.Equal(t, http.StatusCreated, res.StatusCode)

	var id int
	require.NoError(t, e.db.QueryRow(`SELECT id FROM tasks WHERE title = ?`, title).Scan(&id))
	return id
}

func taskTitle(t *testing.T, db *sql.DB, id int) string {
	t.Helper()

	var title string
	require.NoError(t, db.QueryRow(`SELECT title FROM tasks WHERE id = ?`, id).Scan(&title))
	return title
}

func taskStatus(t *testing.T, db *sql.DB, id int) string {
	t.Helper()

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM tasks WHERE id = ?`, id).Scan(&status))
	return status
}

func signInAs(t *testing.T, e *env, username, password string) {
	t.Helper()

	signup(t, e, username, password)
	res := signin(t, e, username, password)
	discard(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "/tasks", res.Request.URL.Path)
}

func TestAccountSignupAndSignin(t *testing.T) {
	for _, tt := range []struct {
		name           string
		signupPassword string
		signinPassword string
		wantStatus     int
		wantPath       string
		wantCookie     bool
		wantSessions   int
	}{
		{
			name:           "ok: 正しいパスワードを入力すると、サインインに成功する",
			signupPassword: " contains whitespace ",
			signinPassword: " contains whitespace ",
			wantStatus:     http.StatusOK,
			wantPath:       "/tasks",
			wantCookie:     true,
			wantSessions:   1,
		},
		{
			name:           "ng: トリム後のパスワードでサインインすると、認証に失敗する",
			signupPassword: " contains whitespace ",
			signinPassword: "contains whitespace",
			wantStatus:     http.StatusBadRequest,
			wantPath:       "/account/signin",
			wantCookie:     false,
			wantSessions:   0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, "test", tt.signupPassword)
			assert.Equal(t, 1, userCount(t, e.db, "test"))

			res := signin(t, e, "test", tt.signinPassword)
			discard(t, res.Body)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)

			cookie := authCookie(t, e)
			if tt.wantCookie {
				require.NotNil(t, cookie)
				assert.NotEmpty(t, cookie.Value)
			} else {
				assert.Nil(t, cookie)
			}
			assert.Equal(t, tt.wantSessions, sessionCount(t, e.db))
		})
	}
}

func TestAccountSignupLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		username   string
		password   string
		wantStatus int
		wantUsers  int
	}{
		{
			name:       "ng: 過大なリクエストボディを送ると、413 になる",
			username:   strings.Repeat("a", 1<<20),
			password:   "test",
			wantStatus: http.StatusRequestEntityTooLarge,
			wantUsers:  0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{tt.username},
				"password":         []string{tt.password},
				"confirm-password": []string{tt.password},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantUsers, userCount(t, e.db, tt.username))
		})
	}
}

func TestAccountPasswordChange(t *testing.T) {
	for _, tt := range []struct {
		name        string
		username    string
		oldPassword string
		newPassword string
		wantPath    string
	}{
		{
			name:        "ok: パスワードを変更すると、新しいパスワードでサインインできる",
			username:    "test",
			oldPassword: "password",
			newPassword: "new-password",
			wantPath:    "/account/password_change/success",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, tt.username, tt.oldPassword)

			res := signin(t, e, tt.username, tt.oldPassword)
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "/tasks", res.Request.URL.Path)

			res = postForm(t, e, "/account/password_change", url.Values{
				"current-password":     []string{tt.oldPassword},
				"new-password":         []string{tt.newPassword},
				"confirm-new-password": []string{tt.newPassword},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			assert.Nil(t, authCookie(t, e))
			assert.Equal(t, 0, sessionCount(t, e.db))

			res = signin(t, e, tt.username, tt.newPassword)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/tasks", res.Request.URL.Path)

			cookie := authCookie(t, e)
			require.NotNil(t, cookie)
			assert.NotEmpty(t, cookie.Value)
			assert.Equal(t, 1, sessionCount(t, e.db))
		})
	}
}

func TestAccountDelete(t *testing.T) {
	for _, tt := range []struct {
		name         string
		username     string
		password     string
		confirmation string
		wantPath     string
	}{
		{
			name:         "ok: アカウントを削除すると、ユーザー行が消える",
			username:     "test",
			password:     "password",
			confirmation: "delete account",
			wantPath:     "/account/delete/success",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, tt.username, tt.password)
			require.Equal(t, 1, userCount(t, e.db, tt.username))

			res := signin(t, e, tt.username, tt.password)
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)

			res = postForm(t, e, "/account/delete", url.Values{
				"password":     []string{tt.password},
				"confirmation": []string{tt.confirmation},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			assert.Equal(t, 0, userCount(t, e.db, tt.username))
			assert.Equal(t, 0, sessionCount(t, e.db))
		})
	}
}

func TestTasksCreate(t *testing.T) {
	for _, tt := range []struct {
		name      string
		signedIn  bool
		title     string
		wantCode  int
		wantPath  string
		wantTasks int
	}{
		{
			name:      "ok: タスクを作成すると、DB に保存される",
			signedIn:  true,
			title:     "buy milk",
			wantCode:  http.StatusCreated,
			wantPath:  "",
			wantTasks: 1,
		},
		{
			name:      "ng: 未サインインでタスクを作ろうとすると、サインインへリダイレクトされる",
			signedIn:  false,
			title:     "buy milk",
			wantCode:  http.StatusOK,
			wantPath:  "/account/signin",
			wantTasks: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.signedIn {
				signInAs(t, e, "test", "password")
			}

			res := postForm(t, e, "/tasks", url.Values{
				"title": []string{tt.title},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			if tt.wantPath != "" {
				assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			}
			assert.Equal(t, tt.wantTasks, taskCount(t, e.db, tt.title))
		})
	}
}

func TestTasksUpdateTitle(t *testing.T) {
	for _, tt := range []struct {
		name      string
		initial   string
		newTitle  string
		wantCode  int
		wantTitle string
	}{
		{
			name:      "ok: タイトルを更新すると、DB のタイトルが変わる",
			initial:   "buy milk",
			newTitle:  "buy oat milk",
			wantCode:  http.StatusOK,
			wantTitle: "buy oat milk",
		},
		{
			name:      "ng: 空のタイトルで更新すると、400 になる",
			initial:   "buy milk",
			newTitle:  "   ",
			wantCode:  http.StatusBadRequest,
			wantTitle: "buy milk",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test", "password")
			id := createTask(t, e, tt.initial)

			res := doForm(t, e, http.MethodPut, fmt.Sprintf("/tasks/%d/title", id), url.Values{
				"title": []string{tt.newTitle},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantTitle, taskTitle(t, e.db, id))
		})
	}
}

func TestTasksUpdateStatus(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     string
		wantCode   int
		wantStatus string
	}{
		{
			name:       "ok: ステータスを done にすると、DB のステータスが変わる",
			status:     "done",
			wantCode:   http.StatusNoContent,
			wantStatus: "done",
		},
		{
			name:       "ng: 不正なステータスで更新すると、400 になる",
			status:     "archived",
			wantCode:   http.StatusBadRequest,
			wantStatus: "created",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test", "password")
			id := createTask(t, e, "buy milk")
			require.Equal(t, "created", taskStatus(t, e.db, id))

			res := doForm(t, e, http.MethodPut, fmt.Sprintf("/tasks/%d/status", id), url.Values{
				"status": []string{tt.status},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantStatus, taskStatus(t, e.db, id))
		})
	}
}
