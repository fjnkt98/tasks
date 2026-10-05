package server

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"
)

type IndexHandler struct {
	t *template.Template
}

func NewIndexHandler() *IndexHandler {
	return &IndexHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/index.html")),
	}
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := h.t.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
