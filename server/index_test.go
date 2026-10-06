package server

import (
	"html/template"
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

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `href="/signin"`) {
			t.Error("body should contain sign in button")
		}
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := NewIndexHandler()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `"/signout"`) {
			t.Error("body should contain sign out button")
		}
	})

	t.Run("render failed", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler := NewIndexHandler()
		handler.t = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
			t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
		}
	})
}
