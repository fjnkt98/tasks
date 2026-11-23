package e2e_test

import (
	"database/sql"
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

func TestAccountSignupAndSignin(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ok: 正しいパスワードを入力すると、サインインに成功する"},
		{name: "ng: トリム後のパスワードでサインインすると、認証に失敗する"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{"test"},
				"password":         []string{" contains whitespace "},
				"confirm-password": []string{" contains whitespace "},
			})
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "/account/signup/success", res.Request.URL.Path)
			require.Equal(t, 1, userCount(t, e.db, "test"))

			switch tt.name {
			case "ok: 正しいパスワードを入力すると、サインインに成功する":
				res = postForm(t, e, "/account/signin", url.Values{
					"username": []string{"test"},
					"password": []string{" contains whitespace "},
				})
				discard(t, res.Body)
				assert.Equal(t, http.StatusOK, res.StatusCode)
				assert.Equal(t, "/tasks", res.Request.URL.Path)

				cookie := authCookie(t, e)
				require.NotNil(t, cookie)
				assert.NotEmpty(t, cookie.Value)
				assert.Equal(t, 1, sessionCount(t, e.db))

			case "ng: トリム後のパスワードでサインインすると、認証に失敗する":
				res = postForm(t, e, "/account/signin", url.Values{
					"username": []string{"test"},
					"password": []string{"contains whitespace"},
				})
				discard(t, res.Body)
				assert.Equal(t, http.StatusBadRequest, res.StatusCode)
				assert.Equal(t, "/account/signin", res.Request.URL.Path)
				assert.Nil(t, authCookie(t, e))
				assert.Equal(t, 0, sessionCount(t, e.db))
			}
		})
	}
}

func TestAccountSignupLimits(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ng: 過大なリクエストボディを送ると、413 になる"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{strings.Repeat("a", 1<<20)},
				"password":         []string{"test"},
				"confirm-password": []string{"test"},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
			assert.Equal(t, 0, userCount(t, e.db, strings.Repeat("a", 1<<20)))
		})
	}
}

func TestAccountPasswordChange(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ok: パスワードを変更すると、新しいパスワードでサインインできる"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{"test"},
				"password":         []string{"password"},
				"confirm-password": []string{"password"},
			})
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)

			res = postForm(t, e, "/account/signin", url.Values{
				"username": []string{"test"},
				"password": []string{"password"},
			})
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "/tasks", res.Request.URL.Path)

			res = postForm(t, e, "/account/password_change", url.Values{
				"current-password":     []string{"password"},
				"new-password":         []string{"new-password"},
				"confirm-new-password": []string{"new-password"},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/account/password_change/success", res.Request.URL.Path)
			assert.Nil(t, authCookie(t, e))
			assert.Equal(t, 0, sessionCount(t, e.db))

			res = postForm(t, e, "/account/signin", url.Values{
				"username": []string{"test"},
				"password": []string{"new-password"},
			})
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
		name string
	}{
		{name: "ok: アカウントを削除すると、ユーザー行が消える"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{"test"},
				"password":         []string{"password"},
				"confirm-password": []string{"password"},
			})
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, 1, userCount(t, e.db, "test"))

			res = postForm(t, e, "/account/signin", url.Values{
				"username": []string{"test"},
				"password": []string{"password"},
			})
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)

			res = postForm(t, e, "/account/delete", url.Values{
				"password":     []string{"password"},
				"confirmation": []string{"delete account"},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/account/delete/success", res.Request.URL.Path)
			assert.Equal(t, 0, userCount(t, e.db, "test"))
			assert.Equal(t, 0, sessionCount(t, e.db))
		})
	}
}

func TestTasks(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ok: タスクを作成すると、DB に保存される"},
		{name: "ng: 未サインインでタスクを作ろうとすると、サインインへリダイレクトされる"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			switch tt.name {
			case "ok: タスクを作成すると、DB に保存される":
				res := postForm(t, e, "/account/signup", url.Values{
					"username":         []string{"test"},
					"password":         []string{"password"},
					"confirm-password": []string{"password"},
				})
				discard(t, res.Body)
				require.Equal(t, http.StatusOK, res.StatusCode)

				res = postForm(t, e, "/account/signin", url.Values{
					"username": []string{"test"},
					"password": []string{"password"},
				})
				discard(t, res.Body)
				require.Equal(t, http.StatusOK, res.StatusCode)

				res = postForm(t, e, "/tasks", url.Values{
					"title": []string{"buy milk"},
				})
				discard(t, res.Body)
				assert.Equal(t, http.StatusCreated, res.StatusCode)
				assert.Equal(t, 1, taskCount(t, e.db, "buy milk"))

			case "ng: 未サインインでタスクを作ろうとすると、サインインへリダイレクトされる":
				res := postForm(t, e, "/tasks", url.Values{
					"title": []string{"buy milk"},
				})
				discard(t, res.Body)
				assert.Equal(t, http.StatusOK, res.StatusCode)
				assert.Equal(t, "/account/signin", res.Request.URL.Path)
				assert.Equal(t, 0, taskCount(t, e.db, "buy milk"))
			}
		})
	}
}
