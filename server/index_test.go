package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fjnkt98/tasks/repository"
)

func TestGetIndex(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := &IndexHandler{}
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `href="/signin"`) {
			t.Errorf("Sign in button should exists")
		}
	})
	t.Run("authorized", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := &IndexHandler{}
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `href="/signout"`) {
			t.Errorf("Sign out button should exists")
		}
	})
}
