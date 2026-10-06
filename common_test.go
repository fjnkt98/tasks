package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHTMX(t *testing.T) {
	t.Run("normal request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		assert.False(t, IsHTMX(r))
	})

	t.Run("htmx request", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		r.Header.Set("HX-Request", "true")

		assert.True(t, IsHTMX(r))
	})
}

func TestHandle400(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Bad Request")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Bad Request")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "bad request\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template400
		t.Cleanup(func() {
			template400 = original
		})
		template400 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle400(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}

func TestHandle404(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Not Found")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "Not Found")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "not found\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template404
		t.Cleanup(func() {
			template404 = original
		})
		template404 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle404(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}

func TestHandle500(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "An Error Occurred!")
		assert.Contains(t, body, `href="/signin"`)
	})

	t.Run("authorized", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, "<head>")
		assert.Contains(t, body, "<body")
		assert.Contains(t, body, "<footer")
		assert.Contains(t, body, "An Error Occurred!")
		assert.Contains(t, body, `"/signout"`)
	})

	t.Run("htmx request", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})

	t.Run("render failed", func(t *testing.T) {
		original := template500
		t.Cleanup(func() {
			template500 = original
		})
		template500 = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		Handle500(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "server error\n", rec.Body.String())
	})
}
