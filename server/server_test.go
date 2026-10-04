package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServer(t *testing.T) {
	client := NewTestDB(t)

	_, err := NewServer(8000, client)
	if err != nil {
		t.Errorf("expected nil, but got %v", err)
	}

}

func TestNewHandler(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "http://localhost:8000")

	client := NewTestDB(t)
	h := newHandler(client)

	t.Run("index", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if allowOrigin := rec.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "" {
			t.Errorf("Access-Control-Allow-Origin header should be empty, but got %s", allowOrigin)
		}
	})
}
