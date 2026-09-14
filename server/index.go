package server

import (
	"net/http"
)

type IndexHandler struct{}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}
