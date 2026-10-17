package presentation

import (
	"log/slog"
	"net/http"
)

func Handle400(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
		AccountName:   GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/400", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusBadRequest)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func Handle404(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
		AccountName:   GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/404", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusNotFound)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func Handle500(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
		AccountName:   GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/500", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusInternalServerError)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
