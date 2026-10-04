package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fjnkt98/tasks/repository"
)

func setupDB(t *testing.T) *sql.DB {
	db, err := repository.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestIsHTMX(t *testing.T) {
	t.Run("normal request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		if IsHTMX(r) {
			t.Errorf("expected false, but got true")
		}
	})

	t.Run("htmx request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "true")

		if !IsHTMX(r) {
			t.Errorf("expected true, but got false")
		}
	})
}

func TestHandle400(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status bad request, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "Bad Request") {
			t.Error("expected contains 'Bad Request', but not found")
		}

		if !strings.Contains(body, `href="/signin"`) {
			t.Errorf("Sign in button should exists")
		}
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status bad request, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "Bad Request") {
			t.Error("expected contains 'Bad Request', but not found")
		}

		if !strings.Contains(body, `href="/signout"`) {
			t.Errorf("Sign out button should exists")
		}
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle400(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status bad request, but got %d", rec.Code)
		}

		if body := rec.Body.String(); body != "bad request\n" {
			t.Errorf("expected body 'bad request', but got %s", body)
		}
	})
}

func TestHandle404(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status not found, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "Not Found") {
			t.Error("expected contains 'Not Found', but not found")
		}

		if !strings.Contains(body, `href="/signin"`) {
			t.Errorf("Sign in button should exists")
		}
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status not found, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "Not Found") {
			t.Error("expected contains 'Not Found', but not found")
		}

		if !strings.Contains(body, `href="/signout"`) {
			t.Errorf("Sign out button should exists")
		}
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle404(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status not found, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "not found\n" {
			t.Errorf("expected body 'not found', but got %s", body)
		}
	})
}

func TestHandle500(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "An Error Occurred!") {
			t.Error("expected contains 'An Error Occurred!', but not found")
		}

		if !strings.Contains(body, `href="/signin"`) {
			t.Errorf("Sign in button should exists")
		}
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "<head>") {
			t.Error("expected contains <head> element, but not found")
		}
		if !strings.Contains(body, "<body") {
			t.Error("expected contains <body> element, but not found")
		}
		if !strings.Contains(body, "<footer") {
			t.Error("expected contains <footer> element, but not found")
		}
		if !strings.Contains(body, "An Error Occurred!") {
			t.Error("expected contains 'An Error Occurred!', but not found")
		}

		if !strings.Contains(body, `href="/signout"`) {
			t.Errorf("Sign out button should exists")
		}
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle500(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status internal server error, but got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "server error\n" {
			t.Errorf("expected body 'server error', but got %s", body)
		}
	})
}
