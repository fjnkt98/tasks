package presentation

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

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/fjnkt98/tasks/internal/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discard(t *testing.T, body io.ReadCloser) {
	t.Helper()

	_, err := io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
}

func integrationTestFixture(t *testing.T) (
	db *sql.DB,
	server *httptest.Server,
	client *http.Client,
) {
	t.Setenv("USE_SECURE_COOKIE", "false")

	db = testutil.NewDB(t)
	handler := NewHandler(db)
	server = httptest.NewServer(handler)
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	client = server.Client()
	client.Jar = jar
	client.Timeout = 5 * time.Second

	return
}

func TestIntegrationSignupAndSignin(t *testing.T) {
	_, server, client := integrationTestFixture(t)

	res, err := client.PostForm(server.URL+"/account/signup", url.Values{
		"username":         []string{"test"},
		"password":         []string{" contains whitespace "},
		"confirm-password": []string{" contains whitespace "},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/account/signup/success", res.Request.URL.Path)

	// fail
	res, err = client.PostForm(server.URL+"/account/signin", url.Values{
		"username": []string{"test"},
		"password": []string{"contains whitespace"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "/account/signin", res.Request.URL.Path)

	// success
	res, err = client.PostForm(server.URL+"/account/signin", url.Values{
		"username": []string{"test"},
		"password": []string{" contains whitespace "},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/tasks", res.Request.URL.Path)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	cookies := client.Jar.Cookies(u)
	i := slices.IndexFunc(cookies, func(cookie *http.Cookie) bool { return cookie.Name == AuthCookieName })
	require.NotEqual(t, -1, i)
	assert.NotEmpty(t, cookies[i].Value)
}

func TestIntegrationContentTooLong(t *testing.T) {
	_, server, client := integrationTestFixture(t)

	res, err := client.PostForm(server.URL+"/account/signup", url.Values{
		"username":         []string{strings.Repeat("a", 1<<20)},
		"password":         []string{"test"},
		"confirm-password": []string{"test"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusRequestEntityTooLarge, res.StatusCode)
}

func TestIntegrationPasswordChangeAndSignin(t *testing.T) {
	db, server, client := integrationTestFixture(t)

	err := usecases.NewAccountUsecase(db).Signup(t.Context(), "test", "password")
	require.NoError(t, err)

	// signin
	res, err := client.PostForm(server.URL+"/account/signin", url.Values{
		"username": []string{"test"},
		"password": []string{"password"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/tasks", res.Request.URL.Path)

	// password change
	res, err = client.PostForm(server.URL+"/account/password_change", url.Values{
		"current-password":     []string{"password"},
		"new-password":         []string{"new-password"},
		"confirm-new-password": []string{"new-password"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/account/password_change/success", res.Request.URL.Path)

	// check logged out
	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	cookies := client.Jar.Cookies(u)
	i := slices.IndexFunc(cookies, func(cookie *http.Cookie) bool { return cookie.Name == AuthCookieName })
	require.Equal(t, -1, i)

	// signin again with new password
	res, err = client.PostForm(server.URL+"/account/signin", url.Values{
		"username": []string{"test"},
		"password": []string{"new-password"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/tasks", res.Request.URL.Path)

	u, err = url.Parse(server.URL)
	require.NoError(t, err)
	cookies = client.Jar.Cookies(u)
	i = slices.IndexFunc(cookies, func(cookie *http.Cookie) bool { return cookie.Name == AuthCookieName })
	require.NotEqual(t, -1, i)
	assert.NotEmpty(t, cookies[i].Value)
}

func TestIntegrationDeleteAccount(t *testing.T) {
	_, server, client := integrationTestFixture(t)

	// signup
	res, err := client.PostForm(server.URL+"/account/signup", url.Values{
		"username":         []string{"test"},
		"password":         []string{"password"},
		"confirm-password": []string{"password"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/account/signup/success", res.Request.URL.Path)

	// signin
	res, err = client.PostForm(server.URL+"/account/signin", url.Values{
		"username": []string{"test"},
		"password": []string{"password"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/tasks", res.Request.URL.Path)

	// delete account
	res, err = client.PostForm(server.URL+"/account/delete", url.Values{
		"password":     []string{"password"},
		"confirmation": []string{"delete account"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/account/delete/success", res.Request.URL.Path)
}
