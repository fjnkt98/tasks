package server

import (
	"html/template"
	"log/slog"
	"net/http"
)

type IndexHandler struct{}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/index.html")
	if err != nil {
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, nil); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}
