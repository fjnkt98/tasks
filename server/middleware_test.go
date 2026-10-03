package server

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fjnkt98/tasks/repository"
	"github.com/fjnkt98/tasks/settings"
)

var mu sync.Mutex

func TestChainedMiddleware(t *testing.T) {
	mux := http.NewServeMux()

	data := make([]string, 0)

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		data = append(data, "main")
		w.WriteHeader(http.StatusOK)
	})

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data = append(data, "m1 before")
			next.ServeHTTP(w, r)
			data = append(data, "m1 after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data = append(data, "m2 before")
			next.ServeHTTP(w, r)
			data = append(data, "m2 after")
		})
	}

	m := NewChainedMiddleware(m1, m2)
	h := m(mux)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	want := []string{"m1 before", "m2 before", "main", "m2 after", "m1 after"}
	if !reflect.DeepEqual(data, want) {
		t.Errorf("expected '%+v', but got '%+v'", want, data)
	}
}

func TestRecoveryMidelleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	h := NewRecoveryMiddleware()(mux)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status internal server error, but got %d", rec.Code)
	}
}

func CaptureLog(t *testing.T, f func()) []byte {
	t.Helper()

	mu.Lock()
	defer mu.Unlock()

	original := slog.Default()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	slog.SetDefault(logger)
	defer slog.SetDefault(original)

	f()

	return buf.Bytes()
}

func TestLoggingMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1234 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	h := NewLoggingMiddleware()(mux)

	msg := CaptureLog(t, func() {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.RemoteAddr = "127.0.0.1"
		req.Host = "localhost:80"
		req.Header.Set("Referer", "localhost:80")

		rec := httptest.NewRecorder()

		synctest.Test(t, func(t *testing.T) {
			h.ServeHTTP(rec, req)
		})
	})

	var data struct {
		Level   string `json:"level"`
		Msg     string `json:"msg"`
		Addr    string `json:"addr"`
		Method  string `json:"method"`
		Path    string `json:"path"`
		Referer string `json:"referer"`
		Host    string `json:"host"`
		Delta   int    `json:"delta"`
	}
	if err := json.Unmarshal(msg, &data); err != nil {
		t.Fatal(err)
	}

	if data.Level != "INFO" {
		t.Errorf("expected level is 'INFO', but got '%s'", data.Level)
	}
	if data.Msg != "ok" {
		t.Errorf("expected msg is 'ok', but got '%s'", data.Msg)
	}
	if data.Addr != "127.0.0.1" {
		t.Errorf("expected addr is '127.0.0.1', but got '%s'", data.Addr)
	}
	if data.Method != "GET" {
		t.Errorf("expected method is 'GET', but got '%s'", data.Method)
	}
	if data.Path != "/test" {
		t.Errorf("expected path is '/test', but got '%s'", data.Path)
	}
	if data.Referer != "localhost:80" {
		t.Errorf("expected referer is 'localhost:80', but got '%s'", data.Referer)
	}
	if data.Host != "localhost:80" {
		t.Errorf("expected host is 'localhost:80', but got '%s'", data.Host)
	}
	if data.Delta != 1234 {
		t.Errorf("expected delta is 1234, but got '%d'", data.Delta)
	}
}

func TestCORSMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	h := NewCORSMiddleware("192.0.2.1:1234")(mux)

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected body 'test', but got '%s'", body)
		}

		for _, c := range []struct {
			Key  string
			Want string
		}{
			{Key: "Access-Control-Allow-Origin", Want: "192.0.2.1:1234"},
			{Key: "Access-Control-Allow-Methods", Want: "GET, POST, PUT, DELETE, OPTIONS"},
			{Key: "Access-Control-Allow-Headers", Want: "Content-Type, Authorization"},
			{Key: "Access-Control-Allow-Credentials", Want: "true"},
			{Key: "Access-Control-Max-Age", Want: "86400"},
		} {
			got := rec.Header().Get(c.Key)
			if got != c.Want {
				t.Errorf("expected %s, but got %s", c.Want, got)
			}
		}
	})

	t.Run("post", func(t *testing.T) {
		values := url.Values{}
		values.Set("foo", "bar")
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", strings.NewReader(values.Encode()))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected body 'test', but got '%s'", body)
		}

		for _, c := range []struct {
			Key  string
			Want string
		}{
			{Key: "Access-Control-Allow-Origin", Want: "192.0.2.1:1234"},
			{Key: "Access-Control-Allow-Methods", Want: "GET, POST, PUT, DELETE, OPTIONS"},
			{Key: "Access-Control-Allow-Headers", Want: "Content-Type, Authorization"},
			{Key: "Access-Control-Allow-Credentials", Want: "true"},
			{Key: "Access-Control-Max-Age", Want: "86400"},
		} {
			got := rec.Header().Get(c.Key)
			if got != c.Want {
				t.Errorf("expected %s, but got %s", c.Want, got)
			}
		}
	})

	t.Run("preflight", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/test", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status no content, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "" {
			t.Errorf("expected body is empty, but got '%s'", body)
		}

		for _, c := range []struct {
			Key  string
			Want string
		}{
			{Key: "Access-Control-Allow-Origin", Want: "192.0.2.1:1234"},
			{Key: "Access-Control-Allow-Methods", Want: "GET, POST, PUT, DELETE, OPTIONS"},
			{Key: "Access-Control-Allow-Headers", Want: "Content-Type, Authorization"},
			{Key: "Access-Control-Allow-Credentials", Want: "true"},
			{Key: "Access-Control-Max-Age", Want: "86400"},
		} {
			got := rec.Header().Get(c.Key)
			if got != c.Want {
				t.Errorf("expected %s, but got %s", c.Want, got)
			}
		}
	})
}

func TestSessionMiddleware(t *testing.T) {
	db, err := repository.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close() // nolint:errcheck

	if _, err := db.ExecContext(t.Context(), "INSERT INTO users (id, name, password) VALUES (1, 'foo', 'foo'), (2, 'bar', 'bar'), (3, 'baz', 'baz')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(t.Context(), "INSERT INTO sessions (user_id, token, expires_at) VALUES (1, 'token1', 3000000000), (2, 'token2', 1000000000)"); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	h := NewSessionMiddleware(db)(mux)

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected response body is 'test', but got '%s'", body)
		}

		if c := rec.Header().Values("Set-Cookie"); len(c) > 0 {
			t.Errorf("Set-Cookie header shouldn't be set, but got %+v", c)
		}
	})
	t.Run("expired", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:     "session_token",
			Value:    "token2",
			MaxAge:   86400,
			Path:     "/",
			Secure:   settings.UseSecureCookie,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected response body is 'test', but got '%s'", body)
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
		if tokenCookie.MaxAge != -1 {
			t.Errorf("cookie MaxAge should be reset, but got %v", tokenCookie.MaxAge)
		}
		if tokenCookie.Value != "" {
			t.Errorf("cookie Value should be reset, but got %v", tokenCookie.Value)
		}
	})
	t.Run("session not found", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:     "session_token",
			Value:    "token3",
			MaxAge:   86400,
			Path:     "/",
			Secure:   settings.UseSecureCookie,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected response body is 'test', but got '%s'", body)
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
		if tokenCookie.MaxAge != -1 {
			t.Errorf("cookie MaxAge should be reset")
		}
		if tokenCookie.Value != "" {
			t.Errorf("cookie Value should be reset")
		}
	})
	t.Run("authorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:     "session_token",
			Value:    "token1",
			MaxAge:   86400,
			Path:     "/",
			Secure:   settings.UseSecureCookie,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected response body is 'test', but got '%s'", body)
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
		if tokenCookie.Value != "token1" {
			t.Errorf("cookie Value should be set")
		}
	})
}

func TestLoginRequiredMiddleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	h := NewLoginRequiredMiddleware()(mux)

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("expected response body is 'test', but got '%s'", body)
		}
	})
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("expected status see other, but got %d", rec.Code)
		}
	})
}
