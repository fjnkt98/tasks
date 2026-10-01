package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fjnkt98/tasks/repository"
)

func TestNewServer(t *testing.T) {
	db, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("failed to create test db: %s", err)
	}
	defer db.Close() // nolint:errcheck

	_, err = NewServer(8000, db)
	if err != nil {
		t.Errorf("expected nil, but got %v", err)
	}

}

func TestNewHandler(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "http://localhost:8000")

	db, err := repository.NewTestDB()
	if err != nil {
		t.Fatalf("failed to create test db: %s", err)
	}
	defer db.Close() // nolint:errcheck

	h := newHandler(db)

	t.Run("index", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if allowOrigin := rec.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "" {
			t.Errorf("expected Access-Control-Allow-Origin is empty, but got %s", allowOrigin)
		}
	})
}
