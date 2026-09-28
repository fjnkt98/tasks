package server

import (
	"errors"
	"html/template"
	"log/slog"
	"net/http"
)

var ErrBadRequest = errors.New("bad request")

func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func Handle400(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	} else {
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
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else {
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
}

func Handle500(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	} else {
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
}
