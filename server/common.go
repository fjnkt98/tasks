package server

import (
	"html/template"
	"log/slog"
	"net/http"
)

func Handle400(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/400.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	if err := t.Execute(w, nil); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/404.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusNotFound)
	if err := t.Execute(w, nil); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func Handle500(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/500.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	if err := t.Execute(w, nil); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
