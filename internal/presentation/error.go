package presentation

import (
	"fmt"
	"log/slog"
	"net/http"
)

type ErrorCode int

const (
	ErrorCode400 ErrorCode = iota
	ErrorCode404
	ErrorCode500
)

var mapping = map[ErrorCode]struct {
	message  string
	template string
	status   int
}{
	ErrorCode400: {
		message:  "bad request",
		template: "page/400",
		status:   http.StatusBadRequest,
	},
	ErrorCode404: {
		message:  "not found",
		template: "page/404",
		status:   http.StatusNotFound,
	},
	ErrorCode500: {
		message:  "server error",
		template: "page/500",
		status:   http.StatusInternalServerError,
	},
}

func Error(w http.ResponseWriter, r *http.Request, code ErrorCode) {
	metadata, ok := mapping[code]
	if !ok {
		panic(fmt.Sprintf("unknown error code %d", code))
	}

	if IsHTMX(r) {
		http.Error(w, metadata.message, metadata.status)
		return
	}

	data := LayoutData{
		Authorized:  IsAuthorized(r.Context()),
		AccountName: GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate(metadata.template, &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(metadata.status)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
