package main

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionToken(t *testing.T) {
	token := NewSessionToken()
	assert.Equal(t, 64, len(token))
}

func TestSetUserIntoContext(t *testing.T) {
	ctx := SetUserIntoContext(t.Context(), User{ID: 1})

	assert.Equal(t, User{ID: 1}, ctx.Value(contextKeyUser))
}

func TestGetUserFromContext(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		v := GetUserFromContext(t.Context())
		assert.Equal(t, User{}, v)
	})

	t.Run("found", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), contextKeyUser, User{ID: 1, Name: "test"})
		v := GetUserFromContext(ctx)
		assert.Equal(t, User{ID: 1, Name: "test"}, v)
	})
}

func extractCookies(t *testing.T, header http.Header) map[string]*http.Cookie {
	t.Helper()

	cookies := make(map[string]*http.Cookie)
	for _, setCookieHeader := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(setCookieHeader)
		require.NoError(t, err)

		cookies[cookie.Name] = cookie
	}
	return cookies
}

func TestCreateUser(t *testing.T) {
	t.Run("password too long", func(t *testing.T) {
		db := NewTestDB(t)

		err := CreateUser(t.Context(), db, "user1", strings.Repeat("p", 73))
		require.Error(t, err)
	})

	t.Run("user already exists", func(t *testing.T) {
		db := NewTestDB(t)

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', 'foo')")
		require.NoError(t, err)

		err = CreateUser(t.Context(), db, "user1", "password")
		require.ErrorIs(t, err, ErrDuplicatedUsername)

		row := db.QueryRowContext(t.Context(), "SELECT name, digest FROM users")
		var name string
		var digest string
		require.NoError(t, row.Scan(&name, &digest))
		assert.Equal(t, "user1", name)
		assert.Equal(t, "foo", digest)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)

		err := CreateUser(t.Context(), db, "user1", "password")
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 1, count)
	})
}

func TestLogin(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()
		require.NoError(t, CreateUser(t.Context(), db, "user1", "password"))
	}

	t.Run("user not found", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		_, err := Login(t.Context(), db, "user2", "password")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("password not match", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		_, err := Login(t.Context(), db, "user1", "invalid")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("password too long", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		_, err := Login(t.Context(), db, "user1", strings.Repeat("p", 73))
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		token, err := Login(t.Context(), db, "user1", "password")
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT token FROM sessions")
		var saved string
		require.NoError(t, row.Scan(&saved))
		assert.Equal(t, saved, token)
	})
}

func TestGetSignupHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})
}

func TestPostSignupHandler(t *testing.T) {
	for _, test := range []struct {
		Name       string
		Body       url.Values
		ErrMessage string
	}{
		{Name: "username empty", Body: url.Values{"password": []string{"bar"}, "confirm-password": []string{"bar"}}, ErrMessage: "username is required"},
		{Name: "password empty", Body: url.Values{"username": []string{"foo"}, "confirm-password": []string{"bar"}}, ErrMessage: "password is required"},
		{Name: "confirm-password empty", Body: url.Values{"username": []string{"foo"}, "password": []string{"bar"}}, ErrMessage: "confirm-password is required"},
		{Name: "all empty", Body: url.Values{}, ErrMessage: "username is required; password is required; confirm-password is required"},
		{Name: "password not match confirm-password", Body: url.Values{"username": []string{"foo"}, "password": []string{"bar"}, "confirm-password": []string{"baz"}}, ErrMessage: "password not match"},
		{Name: "password too long", Body: url.Values{"username": []string{"foo"}, "password": []string{strings.Repeat("p", 73)}, "confirm-password": []string{strings.Repeat("p", 73)}}, ErrMessage: "password too long"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signup?username=foo&password=bar&confirm-password=bar", strings.NewReader(test.Body.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<div role="alert"`)
			assert.Contains(t, body, test.ErrMessage)

			row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
			var count int
			require.NoError(t, row.Scan(&count))
			assert.Equal(t, 0, count)
		})
	}

	t.Run("invalid request body", func(t *testing.T) {
		db := NewTestDB(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signup", strings.NewReader("username=user1&password=password&confirm-password=password&foo=%zz"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSignupHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("user already exists", func(t *testing.T) {
		db := NewTestDB(t)

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)

		values := url.Values{}
		values.Set("username", "user1")
		values.Set("password", "password")
		values.Set("confirm-password", "password")

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signup", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSignupHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, `<div role="alert"`)
		assert.Contains(t, body, "username was already used")

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 1, count)
	})

	t.Run("render failed", func(t *testing.T) {
		db := NewTestDB(t)

		values := url.Values{}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signup", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSignupHandler(db)
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)

		values := url.Values{}
		values.Set("username", "user1")
		values.Set("password", "password")
		values.Set("confirm-password", "password")

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signup", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSignupHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 1, count)
	})
}

func TestGetSignupSuccessHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signup/success", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupSuccessHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signup/success", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupSuccessHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})
}

func TestGetSigninHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signin", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/account/signin", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})
}

func TestPostSigninHandler(t *testing.T) {
	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		require.NoError(t, CreateUser(t.Context(), db, "user1", "password"))
	}

	for _, test := range []struct {
		Name       string
		Body       url.Values
		ErrMessage string
	}{
		{Name: "username empty", Body: url.Values{"password": []string{"bar"}}, ErrMessage: "username is required"},
		{Name: "password empty", Body: url.Values{"username": []string{"foo"}}, ErrMessage: "password is required"},
		{Name: "all empty", Body: url.Values{}, ErrMessage: "username is required; password is required"},
		{Name: "password too long", Body: url.Values{"username": []string{"foo"}, "password": []string{strings.Repeat("p", 73)}}, ErrMessage: "password too long"},
		{Name: "user not found", Body: url.Values{"username": []string{"user2"}, "password": []string{"password"}}, ErrMessage: ErrInvalidCredentials.Error()},
		{Name: "password not match", Body: url.Values{"username": []string{"user1"}, "password": []string{"foo"}}, ErrMessage: ErrInvalidCredentials.Error()},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signin?username=user1&password=password", strings.NewReader(test.Body.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<h1 class="text-4xl font-bold">Sign in</h1>`)
			assert.Contains(t, body, test.ErrMessage)

			cookies := extractCookies(t, rec.Result().Header)
			assert.NotContains(t, cookies, AuthCookieName)

			row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sessions")
			var count int
			require.NoError(t, row.Scan(&count))
			assert.Equal(t, 0, count)
		})
	}

	t.Run("invalid request body", func(t *testing.T) {
		db := NewTestDB(t)

		fixture(db, t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signin", strings.NewReader("username=user1&password=password&foo=%zz"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSigninHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		db := NewTestDB(t)

		values := url.Values{}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signin", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostSigninHandler(db)
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("success", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusSeeOther, rec.Code)

			row := db.QueryRowContext(t.Context(), "SELECT token, expires_at FROM sessions")
			var token string
			var expiresAt int64
			require.NoError(t, row.Scan(&token, &expiresAt))
			assert.Equal(t, int64(946684800+86400), expiresAt) // 2000-01-02T00:00:00Z

			cookies := extractCookies(t, rec.Result().Header)
			tokenCookie, ok := cookies[AuthCookieName]
			require.True(t, ok, "session_token cookie should be set")
			assert.Equal(t, 86400, tokenCookie.MaxAge)
			assert.Equal(t, token, tokenCookie.Value)
			assert.Equal(t, "/", tokenCookie.Path)
		})
	})
}

func TestPostSignoutHandler(t *testing.T) {
	t.Run("authorized", func(t *testing.T) {
		db := NewTestDB(t)

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO sessions (user_id, token, expires_at) VALUES (1, 'token', 32503680000)")
		require.NoError(t, err)

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/account/signout", nil)
		req.AddCookie(NewAuthCookie("token", 86400))

		rec := httptest.NewRecorder()

		h := NewPostSignoutHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sessions")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 0, count)

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		require.True(t, ok, "session_token cookie should be set")
		assert.Equal(t, -1, tokenCookie.MaxAge)
		assert.Equal(t, "", tokenCookie.Value)
		assert.Equal(t, "/", tokenCookie.Path)
	})

	t.Run("unauthorized", func(t *testing.T) {
		db := NewTestDB(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/account/signout", nil)
		rec := httptest.NewRecorder()

		h := NewPostSignoutHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}
