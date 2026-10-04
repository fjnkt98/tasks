package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetIndex(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := NewIndexHandler()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `href="/signin"`) {
			t.Error("body should contain sign in button")
		}
	})
	t.Run("authorized", func(t *testing.T) {
		db := setupDB(t)
		defer db.Close() // nolint:errcheck

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := NewIndexHandler()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `href="/signout"`) {
			t.Error("body should contain sign out button")
		}
	})
}
