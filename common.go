package main

import (
	"bytes"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
)

var ErrBadRequest = errors.New("bad request")

type contextKey int

const (
	contextKeyUser contextKey = iota
)

func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

type LayoutData struct {
	Authorized bool
}

var template400 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/400.html"))

func Handle400(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template400.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusBadRequest)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

var template404 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/404.html"))

func Handle404(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template404.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusNotFound)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

var template500 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/500.html"))

func Handle500(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template500.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusInternalServerError)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
