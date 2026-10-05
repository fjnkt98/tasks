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
)

var mu sync.Mutex

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

	type Log struct {
		Level   string `json:"level"`
		Msg     string `json:"msg"`
		Addr    string `json:"addr"`
		Method  string `json:"method"`
		Path    string `json:"path"`
		Referer string `json:"referer"`
		Host    string `json:"host"`
		Delta   int    `json:"delta"`
	}

	var data Log
	if err := json.Unmarshal(msg, &data); err != nil {
		t.Fatal(err)
	}

	var want = Log{
		Level:   "INFO",
		Msg:     "ok",
		Addr:    "127.0.0.1",
		Method:  "GET",
		Path:    "/test",
		Referer: "localhost:80",
		Host:    "localhost:80",
		Delta:   1234,
	}
	if data != want {
		t.Errorf("expected %+v, but got %+v", want, data)
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
			t.Errorf("body should be 'test', but got '%s'", body)
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
			t.Errorf("body should be 'test', but got '%s'", body)
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
			t.Errorf("body should be empty, but got '%s'", body)
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
	client := NewTestDB(t)

	if _, err := client.User.CreateBulk(
		client.User.Create().SetName("foo").SetPassword("foo"),
		client.User.Create().SetName("bar").SetPassword("bar"),
		client.User.Create().SetName("baz").SetPassword("baz"),
	).Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Session.CreateBulk(
		client.Session.Create().SetUserID(1).SetToken("token1").SetExpiresAt(time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)),
		client.Session.Create().SetUserID(2).SetToken("token2").SetExpiresAt(time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)),
	).Save(t.Context()); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	h := NewSessionMiddleware(client)(mux)

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("body should be 'test', but got '%s'", body)
		}

		if c := rec.Header().Values("Set-Cookie"); len(c) > 0 {
			t.Errorf("Set-Cookie header shouldn't be set, but got %+v", c)
		}
	})

	t.Run("expired", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(NewAuthCookie("token2", 86400))

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("body should be 'test', but got '%s'", body)
		}

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		if !ok {
			t.Fatal("session_token cookie should be set")
		}
		if tokenCookie.MaxAge != -1 {
			t.Errorf("cookie MaxAge should be reset, but got %v", tokenCookie.MaxAge)
		}
		if tokenCookie.Value != "" {
			t.Errorf("cookie Value should be reset, but got %v", tokenCookie.Value)
		}
		if tokenCookie.Path != "/" {
			t.Errorf("cookie Path should be '/', but got '%v'", tokenCookie.Path)
		}
	})

	t.Run("session not found", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(NewAuthCookie("token3", 86400))

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("body should be 'test', but got '%s'", body)
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

	t.Run("authorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(NewAuthCookie("token1", 86400))

		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "test\n" {
			t.Errorf("body should be 'test', but got '%s'", body)
		}

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		if !ok {
			t.Fatal("session_token cookie should be set")
		}
		if tokenCookie.MaxAge != 86400 {
			t.Errorf("cookie MaxAge should be set")
		}
		if tokenCookie.Value != "token1" {
			t.Errorf("cookie Value should be set")
		}
		if tokenCookie.Path != "/" {
			t.Errorf("cookie Path should be '/', but got '%v'", tokenCookie.Path)
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
			t.Errorf("body should be 'test', but got '%s'", body)
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
