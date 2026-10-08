package main

import (
	"database/sql"
	"html/template"
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
)

func TestNewServer(t *testing.T) {
	client := NewTestDB(t)

	_, err := NewServer(8000, client)
	assert.NoError(t, err)
}

func TestNewHandler(t *testing.T) {
	client := NewTestDB(t)
	h := NewHandler(client)

	t.Run("index", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestCrossOriginProtection(t *testing.T) {
	client := NewTestDB(t)
	h := NewHandler(client)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/signout", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestIsHTMX(t *testing.T) {
	t.Run("normal request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		assert.False(t, IsHTMX(r))
	})

	t.Run("htmx request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "true")

		assert.True(t, IsHTMX(r))
	})
}

func TestHandle400(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Bad Request")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Bad Request")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "bad request\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template400
		t.Cleanup(func() {
			template400 = original
		})
		template400 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}

func TestHandle404(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Not Found")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Not Found")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "not found\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template404
		t.Cleanup(func() {
			template404 = original
		})
		template404 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}

func TestHandle500(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "An Error Occurred!")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "An Error Occurred!")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template500
		t.Cleanup(func() {
			template500 = original
		})
		template500 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}

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

	db = NewTestDB(t)
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

func TestUnauthorizedRedirection(t *testing.T) {
	_, server, client := integrationTestFixture(t)

	// top page
	res, err := client.Get(server.URL + "/")
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/", res.Request.URL.Path)

	// login required page
	for _, test := range []struct {
		Name string
		Path string
	}{
		{Name: "get tasks", Path: "/tasks"},
		{Name: "get task", Path: "/tasks/1"},
		{Name: "get task edit", Path: "/tasks/1/edit"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			res, err = client.Get(server.URL + test.Path)
			require.NoError(t, err)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/signin", res.Request.URL.Path)
		})
	}

	for _, test := range []struct {
		Name string
		Path string
		Body url.Values
	}{
		{Name: "post tasks", Path: "/tasks", Body: url.Values{"title": []string{"test"}}},
		{Name: "signout", Path: "/signout"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			res, err = client.PostForm(server.URL+test.Path, test.Body)
			require.NoError(t, err)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/signin", res.Request.URL.Path)
		})
	}

	for _, test := range []struct {
		Name string
		Path string
		Body url.Values
	}{
		{Name: "put task title", Path: "/tasks/1/title", Body: url.Values{"title": []string{"test"}}},
		{Name: "put task status", Path: "/tasks/1/status", Body: url.Values{"status": []string{"done"}}},
	} {
		t.Run(test.Name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, server.URL+test.Path, strings.NewReader(test.Body.Encode()))
			require.NoError(t, err)
			res, err = client.Do(req)
			require.NoError(t, err)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/signin", res.Request.URL.Path)
		})
	}

	for _, test := range []struct {
		Name string
		Path string
	}{
		{Name: "delete task", Path: "/tasks/1"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, server.URL+test.Path, nil)
			require.NoError(t, err)
			res, err = client.Do(req)
			require.NoError(t, err)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/signin", res.Request.URL.Path)
		})
	}
}

func TestIntegrationSignupAndSignin(t *testing.T) {
	_, server, client := integrationTestFixture(t)

	res, err := client.PostForm(server.URL+"/signup", url.Values{
		"username":         []string{"test"},
		"password":         []string{"test"},
		"confirm-password": []string{"test"},
	})
	require.NoError(t, err)
	discard(t, res.Body)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "/signup/success", res.Request.URL.Path)

	res, err = client.PostForm(server.URL+"/signin", url.Values{
		"username": []string{"test"},
		"password": []string{"test"},
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
