package server

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fjnkt98/tasks/ent"
)

func TestNewSessionToken(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Errorf("token length should be 64, but got %d", len(token))
	}
}

func TestSetUserIDIntoContext(t *testing.T) {
	ctx := SetUserIDIntoContext(t.Context(), 1)

	v := ctx.Value(contextKeyUser)
	if ty := reflect.TypeOf(v).String(); ty != "int" {
		t.Errorf("expected type `int`, but got `%s`", ty)
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
		ctx := context.WithValue(t.Context(), contextKeyUser, int(1))
		v := GetUserIDFromContext(ctx)
		if v != 1 {
			t.Errorf("expected 1, but got %d", v)
		}
	})
}

func extractCookies(t *testing.T, header http.Header) map[string]*http.Cookie {
	t.Helper()

	cookies := make(map[string]*http.Cookie)
	for _, setCookieHeader := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(setCookieHeader)
		if err != nil {
			t.Fatal(err)
		}
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

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})
}

func TestPostSignupHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?username=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSignupHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shoudn't be nil, but got nil")
			}
			if msg := "username is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})

		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("confirm-password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSignupHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shoudn't be nil, but got nil")
			}
			if msg := "password is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})

		t.Run("confirm-password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup?confirm-password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSignupHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shoudn't be nil, but got nil")
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
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSignupHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shouldn't be nil, but got nil")
			}
			if msg := "password not match"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := NewTestDB(t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			count, err := client.User.Query().Count(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if count != 1 {
				t.Errorf("users count should be 1, but got %d", count)
			}
		})

		t.Run("validation failed", func(t *testing.T) {
			client := NewTestDB(t)

			values := url.Values{}
			values.Set("username", "\t")
			values.Set("password", "\n")
			values.Set("confirm-password", " ")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<div role="alert"`) {
				t.Error("body should contain alert component, but not found")
			}
			if !strings.Contains(body, "username is required; password is required; confirm-password is required") {
				t.Error("body should contain error message, but not found")
			}
		})

		t.Run("invalid request body", func(t *testing.T) {
			client := NewTestDB(t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader("username=user1&password=password&confirm-password=password&foo=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}
		})

		t.Run("user already exists", func(t *testing.T) {
			client := NewTestDB(t)

			if _, err := client.User.Create().SetName("user1").SetPassword("password").Save(t.Context()); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")
			values.Set("confirm-password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<div role="alert"`) {
				t.Error("body should contain alert component, but not found")
			}
			if !strings.Contains(body, "username was already used") {
				t.Error("body should contain error message, but not found")
			}
		})

		t.Run("render failed", func(t *testing.T) {
			client := NewTestDB(t)

			values := url.Values{}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signup", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSignupHandler(client)
			h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}
		})
	})
}

func TestGetSignupSuccessHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupSuccessHandler()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSignupSuccessHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})
}

func TestGetSigninHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/signup", nil)
		rec := httptest.NewRecorder()

		h := NewGetSigninHandler()
		h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})
}

func TestPostSigninHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("username empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("password", "bar")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?username=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSigninHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shouldn't be nil, but got nil")
			}
			if msg := "username is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})

		t.Run("password empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("username", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin?password=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostSigninHandler(client)
			_, err := h.GetParams(req)
			if err == nil {
				t.Fatal("err shouldn't be nil, but got nil")
			}
			if msg := "password is required"; err.Error() != msg {
				t.Errorf("error should be '%s', but got '%s'", msg, err)
			}
		})
	})

	var fixture = func(client *ent.Client, t *testing.T) {
		t.Helper()

		if err := NewPostSignupHandler(client).Signup(t.Context(), SignupParams{Username: "user1", Password: "password"}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("Signin", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := NewTestDB(t)

				fixture(client, t)

				h := NewPostSigninHandler(client)
				token, err := h.Signin(t.Context(), SigninParams{Username: "user1", Password: "password"})
				if err != nil {
					t.Fatal(err)
				}

				s, err := client.Session.Get(t.Context(), 1)
				if err != nil {
					t.Fatal(err)
				}

				if s.Token != token {
					t.Error("generated token should match saved session's one")
				}

				if !s.ExpiresAt.Equal(time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)) {
					t.Errorf("expires timestamp should be 2000-01-02T00:00:00Z, but got %v", s.ExpiresAt)
				}
			})
		})

		t.Run("user not found", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			h := NewPostSigninHandler(client)
			_, err := h.Signin(t.Context(), SigninParams{Username: "user2", Password: "password"})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("expected ErrInvalidCredentials, but got %+v", err)
			}
		})

		t.Run("password not match", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			h := NewPostSigninHandler(client)
			_, err := h.Signin(t.Context(), SigninParams{Username: "user1", Password: "foo"})
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("expected ErrInvalidCredentials, but got %+v", err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "password")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status see other, but got %d", rec.Code)
			}

			s, err := client.Session.Get(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			cookies := extractCookies(t, rec.Result().Header)
			tokenCookie, ok := cookies[AuthCookieName]
			if !ok {
				t.Fatal("session_token cookie should be set")
			}
			if tokenCookie.MaxAge != 86400 {
				t.Errorf("cookie MaxAge should be set")
			}
			if tokenCookie.Value != s.Token {
				t.Errorf("cookie Value should be set")
			}
			if tokenCookie.Path != "/" {
				t.Errorf("cookie Path should be '/', but got '%v'", tokenCookie.Path)
			}
		})

		t.Run("bad request", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<h1 class="text-4xl font-bold">Sign in</h1>`) {
				t.Errorf("body should be sign in page")
			}

			if msg := "password is required"; !strings.Contains(body, msg) {
				t.Errorf("body should contain error message '%s', but not found", msg)
			}
		})

		t.Run("invalid credentials", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("username", "user1")
			values.Set("password", "foo")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, `<h1 class="text-4xl font-bold">Sign in</h1>`) {
				t.Errorf("body should be sign in page")
			}

			if msg := ErrInvalidCredentials.Error(); !strings.Contains(body, msg) {
				t.Errorf("body should contain error message '%s', but not found", msg)
			}
		})

		t.Run("invalid request body", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader("username=user1&password=password&foo=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}
		})

		t.Run("render failed", func(t *testing.T) {
			client := NewTestDB(t)

			values := url.Values{}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostSigninHandler(client)
			h.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}
		})
	})
}

func TestPostSignoutHandler(t *testing.T) {
	t.Run("authorized", func(t *testing.T) {
		client := NewTestDB(t)

		if _, err := client.User.Create().SetName("user1").SetPassword("password").Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Session.Create().SetUserID(1).SetToken("token").SetExpiresAt(time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/signout", nil)
		req.AddCookie(NewAuthCookie("token", 86400))

		rec := httptest.NewRecorder()

		h := NewPostSignoutHandler(client)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status see otehr, but got %d", rec.Code)
		}

		count, err := client.Session.Query().Count(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("sessions should be cleared, but still exists %d sessions", count)
		}

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		if !ok {
			t.Fatal("session_token cookie should be set")
		}
		if tokenCookie.MaxAge != -1 {
			t.Errorf("cookie MaxAge should be reset")
		}
		if tokenCookie.Value != "" {
			t.Errorf("cookie Value should be reset")
		}
		if tokenCookie.Path != "/" {
			t.Errorf("cookie Path should be '/', but got '%v'", tokenCookie.Path)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		client := NewTestDB(t)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signout", nil)
		rec := httptest.NewRecorder()

		h := NewPostSignoutHandler(client)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status see otehr, but got %d", rec.Code)
		}
	})
}
