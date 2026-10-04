package server

import (
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

	w.WriteHeader(http.StatusOK)
	if err := h.t.Execute(w, &data); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}
