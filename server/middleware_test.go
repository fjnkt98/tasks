package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func TestRecoveryMidelleware(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	h := RecoveryMiddleware(mux)
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

	h := LoggingMiddleware(mux)

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
	t.Setenv("CORS_ALLOW_ORIGIN", "192.0.2.1:1234")

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	h := CORSMiddleware(mux)

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
			{Key: "Access-Control-Max-Age", Want: "86400"},
		} {
			got := rec.Header().Get(c.Key)
			if got != c.Want {
				t.Errorf("expected %s, but got %s", c.Want, got)
			}
		}
	})
}
