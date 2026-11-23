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

	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/adapter/web"
	"github.com/fjnkt98/tasks/internal/app"
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
	server := httptest.NewServer(app.NewHandler(db))
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

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	return string(body)
}

func get(t *testing.T, e *env, path string) (*http.Response, string) {
	t.Helper()

	res, err := e.client.Get(e.server.URL + path)
	require.NoError(t, err)
	return res, readBody(t, res)
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

func signInAs(t *testing.T, e *env, username string) {
	t.Helper()

	const password = "password"
	signup(t, e, username, password)
	res := signin(t, e, username, password)
	discard(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "/tasks", res.Request.URL.Path)
}

func setupAdmin(t *testing.T, e *env, username, password string) {
	t.Helper()

	require.NoError(t, app.NewAdminUsecase(e.db).Setup(t.Context(), username, password))
}

func authCookie(t *testing.T, e *env) *http.Cookie {
	t.Helper()

	u, err := url.Parse(e.server.URL)
	require.NoError(t, err)
	cookies := e.client.Jar.Cookies(u)
	i := slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == web.AuthCookieName
	})
	if i < 0 {
		return nil
	}
	return cookies[i]
}

func userID(t *testing.T, db *sql.DB, name string) int {
	t.Helper()

	var id int
	require.NoError(t, db.QueryRow(`SELECT id FROM users WHERE name = ?`, name).Scan(&id))
	return id
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

func activityCount(t *testing.T, db *sql.DB, userID int, kind, message string) int {
	t.Helper()

	var n int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM activities WHERE user_id = ? AND kind = ? AND message = ?`,
		userID, kind, message,
	).Scan(&n))
	return n
}

func taskPath(id int, suffix string) string {
	if suffix == "" {
		return fmt.Sprintf("/tasks/%d", id)
	}
	return fmt.Sprintf("/tasks/%d/%s", id, suffix)
}
