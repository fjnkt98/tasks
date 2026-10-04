package server

import (
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

	w.WriteHeader(http.StatusBadRequest)
	if err := template400.Execute(w, &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
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

	w.WriteHeader(http.StatusNotFound)
	if err := template404.Execute(w, &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
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

	w.WriteHeader(http.StatusInternalServerError)
	if err := template500.Execute(w, &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
