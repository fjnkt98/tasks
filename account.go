package main

import (
	"bytes"
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
)

type GetMeHandler struct {
	db *sql.DB
	t  *template.Template
}

func NewGetMeHandler(db *sql.DB) *GetMeHandler {
	return &GetMeHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/layout.html", "templates/account.html")),
	}
}

func (h GetMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	data := struct {
		LayoutData
		IsAdmin   bool
		TaskCount int
		DoneCount int
	}{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
			UserName:   user.Name,
		},
		IsAdmin:   IsAdmin(r.Context()),
		TaskCount: 0,
		DoneCount: 0,
	}

	q := "SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0) FROM tasks WHERE user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, user.ID)
	if err := row.Scan(&data.TaskCount, &data.DoneCount); err != nil {
		slog.ErrorContext(r.Context(), "get counts", slog.Any("error", err))
		Handle500(w, r)
		return
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

type GetUserHandler struct {
	db *sql.DB
}

func NewGetUserHandler(db *sql.DB) *GetUserHandler {
	return &GetUserHandler{
		db: db,
	}
}

func (h GetUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}
