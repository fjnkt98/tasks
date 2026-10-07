package main

import (
	"bytes"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, want, data)
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

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
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
	require.NoError(t, json.Unmarshal(msg, &data))

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
	assert.Equal(t, want, data)
}

func TestSessionMiddleware(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (name, digest) VALUES
			('foo', ''),
			('bar', ''),
			('baz', '')`,
		)
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(1, 'token1', 32503680000),
			(2, 'token2', 0)`,
		)
		require.NoError(t, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "test") // nolint:errcheck
	})

	t.Run("unauthorized", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h := NewSessionMiddleware(db)(mux)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test\n", rec.Body.String())

		assert.Empty(t, rec.Header().Values("Set-Cookie"))
	})

	t.Run("expired", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(NewAuthCookie("token2", 86400))

		rec := httptest.NewRecorder()

		h := NewSessionMiddleware(db)(mux)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test\n", rec.Body.String())

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		require.True(t, ok, "session_token cookie should be set")
		assert.Equal(t, -1, tokenCookie.MaxAge)
		assert.Equal(t, "", tokenCookie.Value)
		assert.Equal(t, "/", tokenCookie.Path)
	})

	t.Run("session not found", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		req.AddCookie(NewAuthCookie("token3", 86400))

		rec := httptest.NewRecorder()

		h := NewSessionMiddleware(db)(mux)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test\n", rec.Body.String())

		cookies := extractCookies(t, rec.Result().Header)
		tokenCookie, ok := cookies[AuthCookieName]
		require.True(t, ok, "session_token cookie should be set")
		assert.Equal(t, -1, tokenCookie.MaxAge)
		assert.Equal(t, "", tokenCookie.Value)
		assert.Equal(t, "/", tokenCookie.Path)
	})

	t.Run("authorized", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			db := NewTestDB(t)
			fixture(t, db)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
			req.AddCookie(NewAuthCookie("token1", 86400))

			rec := httptest.NewRecorder()

			h := NewSessionMiddleware(db)(mux)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "test\n", rec.Body.String())

			cookies := extractCookies(t, rec.Result().Header)
			tokenCookie, ok := cookies[AuthCookieName]
			require.True(t, ok, "session_token cookie should be set")
			assert.Equal(t, 86400, tokenCookie.MaxAge)
			assert.Equal(t, "token1", tokenCookie.Value)
			assert.Equal(t, "/", tokenCookie.Path)

			row := db.QueryRowContext(t.Context(), "SELECT expires_at FROM sessions WHERE token = 'token1'")
			var expiresAt int64
			require.NoError(t, row.Scan(&expiresAt))
			assert.Equal(t, int64(946771200), expiresAt)
		})
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

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test\n", rec.Body.String())
	})

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}
