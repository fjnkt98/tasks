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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionToken(t *testing.T) {
	token, err := NewSessionToken()
	require.NoError(t, err)
	assert.Equal(t, 64, len(token))
}

func TestSetUserIDIntoContext(t *testing.T) {
	ctx := SetUserIDIntoContext(t.Context(), 1)

	assert.Equal(t, 1, ctx.Value(contextKeyUser))
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		v := GetUserIDFromContext(t.Context())
		assert.Equal(t, 0, v)
	})

	t.Run("found", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), contextKeyUser, int(1))
		v := GetUserIDFromContext(ctx)
		assert.Equal(t, 1, v)
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

func TestGetSignupHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})
}

func TestPostSignupHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?username=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)

			require.Error(t, err)
			assert.EqualError(t, err, "username is required")
		})

		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)

			require.Error(t, err)
			assert.EqualError(t, err, "password is required")
		})

		t.Run("confirm-password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?confirm-password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)

			require.Error(t, err)
			assert.EqualError(t, err, "confirm-password is required")
		})

		t.Run("password don't match confirm-password", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "bar")
			values.Set("confirm-password", "baz")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)

			require.Error(t, err)
			assert.EqualError(t, err, "password not match")
		})

		t.Run("password contains whitespace", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "password ")
			values.Set("confirm-password", "password ")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			params, err := h.GetParams(req)

			require.NoError(t, err)
			assert.Equal(t, "password ", params.Password)
		})

		t.Run("password too long", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", strings.Repeat("p", 73))
			values.Set("confirm-password", strings.Repeat("p", 73))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)

			require.Error(t, err)
			assert.EqualError(t, err, "password too long")
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db := NewTestDB(t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
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

		t.Run("validation failed", func(t *testing.T) {
			db := NewTestDB(t)

			values := url.Values{}
			values.Set("username", "\t")
			values.Set("password", "")
			values.Set("confirm-password", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<div role="alert"`)
			assert.Contains(t, body, "username is required; password is required; confirm-password is required")
		})

		t.Run("invalid request body", func(t *testing.T) {
			db := NewTestDB(t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader("username=user1&password=password&confirm-password=password&foo=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
		})

		t.Run("user already exists", func(t *testing.T) {
			db := NewTestDB(t)

			_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
			require.NoError(t, err)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<div role="alert"`)
			assert.Contains(t, body, "username was already used")
		})

		t.Run("render failed", func(t *testing.T) {
			db := NewTestDB(t)

			values := url.Values{}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
		})
	})
}

func TestGetSignupSuccessHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupSuccessHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
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
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
	})
}

func TestPostSigninHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?username=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSigninHandler(db)
			_, err := h.GetParams(req)
			require.Error(t, err)
			assert.EqualError(t, err, "username is required")
		})

		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSigninHandler(db)
			_, err := h.GetParams(req)
			require.Error(t, err)
			assert.EqualError(t, err, "password is required")
		})

		t.Run("password contains whitespace", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "password ")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSigninHandler(db)
			params, err := h.GetParams(req)
			require.NoError(t, err)

			assert.Equal(t, "password ", params.Password)
		})

		t.Run("password too long", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", strings.Repeat("p", 73))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			require.NoError(t, req.ParseForm())

			h := NewPostSigninHandler(db)
			_, err := h.GetParams(req)
			require.Error(t, err)
			assert.EqualError(t, err, "password too long")
		})
	})

	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		require.NoError(t, CreateUser(t.Context(), db, "user1", "password"))
	}

	t.Run("Signin", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				db := NewTestDB(t)

				fixture(db, t)

				h := NewPostSigninHandler(db)
				token, err := h.Signin(t.Context(), SigninParams{Username: "user1", Password: "password"})
				require.NoError(t, err)

				row := db.QueryRowContext(t.Context(), "SELECT token, expires_at FROM sessions WHERE id = 1")
				var savedToken string
				var expiresAt int64
				require.NoError(t, row.Scan(&savedToken, &expiresAt))

				assert.Equal(t, savedToken, token)
				assert.Equal(t, time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC).Unix(), expiresAt)
			})
		})

		t.Run("user not found", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			h := NewPostSigninHandler(db)
			_, err := h.Signin(t.Context(), SigninParams{Username: "user2", Password: "password"})
			assert.ErrorIs(t, err, ErrInvalidCredentials)
		})

		t.Run("password not match", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			h := NewPostSigninHandler(db)
			_, err := h.Signin(t.Context(), SigninParams{Username: "user1", Password: "foo"})
			assert.ErrorIs(t, err, ErrInvalidCredentials)
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusSeeOther, rec.Code)

			row := db.QueryRowContext(t.Context(), "SELECT token, expires_at FROM sessions WHERE id = 1")
			var token string
			var expiresAt int64
			require.NoError(t, row.Scan(&token, &expiresAt))

			cookies := extractCookies(t, rec.Result().Header)
			tokenCookie, ok := cookies[AuthCookieName]
			require.True(t, ok, "session_token cookie should be set")
			assert.Equal(t, 86400, tokenCookie.MaxAge)
			assert.Equal(t, token, tokenCookie.Value)
			assert.Equal(t, "/", tokenCookie.Path)
		})

		t.Run("bad request", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<h1 class="text-4xl font-bold">Sign in</h1>`)

			assert.Contains(t, body, "password is required")
		})

		t.Run("invalid credentials", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, `<h1 class="text-4xl font-bold">Sign in</h1>`)

			assert.Contains(t, body, ErrInvalidCredentials.Error())
		})

		t.Run("invalid request body", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader("username=user1&password=password&foo=%zz"))
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
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
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

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/signout", nil)
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

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signout", nil)
		rec := httptest.NewRecorder()

		h := NewPostSignoutHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}
