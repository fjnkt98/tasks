package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/fjnkt98/tasks/repository"
	"github.com/fjnkt98/tasks/settings"
)

func TestNewSessionToken(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Errorf("expected token length is 64, but got %d", len(token))
	}
}

func TestSetUserIDIntoContext(t *testing.T) {
	ctx := SetUserIDIntoContext(t.Context(), 1)

	v := ctx.Value(contextKeyUser)
	if ty := reflect.TypeOf(v).String(); ty != "int64" {
		t.Errorf("expected type `int64`, but got `%s`", ty)
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		v := GetUserIDFromContext(t.Context())
		if v != 0 {
			t.Errorf("expected 0, but got %d", v)
		}
	})

	t.Run("found", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), contextKeyUser, int64(1))
		v := GetUserIDFromContext(ctx)
		if v != 1 {
			t.Errorf("expected 1, but got %d", v)
		}
	})
}

func TestGetSignupHandler(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()

	h := NewGetSignupHandler()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status ok, but got %d", rec.Code)
	}
}

func TestPostSignupHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "username is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "password is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
		t.Run("confirm-password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "confirm-password is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
		t.Run("password don't match confirm-password", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "bar")
			values.Set("confirm-password", "baz")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSignupHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "password not match"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
			var count int
			if err := row.Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Errorf("users count should be 1, but got %d", count)
			}
		})
		t.Run("validation failed", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			values := url.Values{}
			values.Set("username", "\t")
			values.Set("password", "\n")
			values.Set("confirm-password", " ")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<div role="alert"`) {
				t.Error("alert component should exists, but not found")
			}
			if !strings.Contains(body, "username is required; password is required; confirm-password is required") {
				t.Error("error message should exists, but not found")
			}
		})
		t.Run("user already exists", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO users (id, name, password) VALUES (?, ?, '');", 1, "user1"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<div role="alert"`) {
				t.Error("alert component should exists, but not found")
			}
			if !strings.Contains(body, "username was already used") {
				t.Error("error message should exists, but not found")
			}
		})
	})
}

func TestGetSignupSuccessHandler(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()

	h := NewGetSignupSuccessHandler()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status ok, but got %d", rec.Code)
	}
}

func TestGetSigninHandler(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
	rec := httptest.NewRecorder()

	h := NewGetSigninHandler()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status ok, but got %d", rec.Code)
	}
}

func TestPostSigninHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSigninHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "username is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			h := NewPostSigninHandler(db)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("expected err is not nil, but got nil")
			}
			if msg := "password is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
	})

	t.Run("Signin", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				db, err := repository.NewTestDB()
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close() // nolint:errcheck

				if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
					t.Fatal(err)
				}

				h := NewPostSigninHandler(db)
				token, err := h.Signin(t.Context(), SigninParams{Username: "user1", Password: "password"})
				if err != nil {
					t.Fatal(err)
				}

				session, err := repository.New(db).GetSession(t.Context(), 1)
				if err != nil {
					t.Fatal(err)
				}

				if session.Token != token {
					t.Error("generated token should match saved session's one")
				}

				if session.ExpiresAt != 946771200 { // 2000-01-01T00:00:00Z + 24 hours
					t.Errorf("expires timestamp should be 946771200, but got %d", session.ExpiresAt)
				}
			})
		})
		t.Run("user not found", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
				t.Fatal(err)
			}

			h := NewPostSigninHandler(db)
			_, err = h.Signin(t.Context(), SigninParams{Username: "user2", Password: "password"})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("expected ErrInvalidCredentials, but got %+v", err)
			}
		})
		t.Run("password not match", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
				t.Fatal(err)
			}

			h := NewPostSigninHandler(db)
			_, err = h.Signin(t.Context(), SigninParams{Username: "user1", Password: "foo"})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("expected ErrInvalidCredentials, but got %+v", err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			session, err := repository.New(db).GetSession(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			cookies := make(map[string]*http.Cookie)
			for _, setCookieHeader := range rec.Header().Values("Set-Cookie") {
				cookie, err := http.ParseSetCookie(setCookieHeader)
				if err != nil {
					t.Fatal(err)
				}
				cookies[cookie.Name] = cookie
			}

			tokenCookie, ok := cookies["session_token"]
			if !ok {
				t.Fatal("session_token cookie should be set")
			}
			if tokenCookie.MaxAge != 86400 {
				t.Errorf("cookie MaxAge should be set")
			}
			if tokenCookie.Value != session.Token {
				t.Errorf("cookie Value should be set")
			}
		})
		t.Run("bad request", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<h1 class="text-4xl font-bold">Sign in</h1>`) {
				t.Errorf("expected sign in page")
			}

			if msg := "password is required"; !strings.Contains(body, msg) {
				t.Errorf("error message '%s' should exists, but not found", msg)
			}
		})
		t.Run("invalid credentials", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if err := NewPostSignupHandler(db).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(db)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<h1 class="text-4xl font-bold">Sign in</h1>`) {
				t.Errorf("expected sign in page")
			}

			if msg := ErrInvalidCredentials.Error(); !strings.Contains(body, msg) {
				t.Errorf("error message '%s' should exists, but not found", msg)
			}
		})
	})
}

func TestGetSignoutHandler(t *testing.T) {
	t.Run("authorized", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO users (id, name, password) VALUES (?, ?, '');", 1, "user1"); err != nil {
			t.Fatal(err)
		}

		q := repository.New(db)
		if _, err := q.CreateSession(t.Context(), repository.CreateSessionParams{
			UserID:    1,
			Token:     "token",
			ExpiresAt: 3000000000,
		}); err != nil {
			t.Fatal(err)
		}

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/signout", nil)
		req.AddCookie(&http.Cookie{
			Name:     "session_token",
			Value:    "token",
			MaxAge:   86400,
			Path:     "/",
			Secure:   settings.UseSecureCookie,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		rec := httptest.NewRecorder()

		h := NewGetSignoutHandler(db)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status see otehr, but got %d", rec.Code)
		}

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sessions")
		var count int
		if err := row.Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("sessions should be cleared, but still exists %d sessions", count)
		}
	})
	t.Run("unauthorized", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signout", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignoutHandler(db)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status see otehr, but got %d", rec.Code)
		}
	})
}
