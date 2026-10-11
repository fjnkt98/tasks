package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func DeleteAccount(ctx context.Context, db *sql.DB, userID int, password string) error {
	q := "SELECT digest FROM users WHERE id = ? LIMIT 1"
	row := db.QueryRowContext(ctx, q, userID)
	var digest string
	if err := row.Scan(&digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("get user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(digest), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	q = "DELETE FROM users WHERE id = ? AND digest = ?"
	result, err := db.ExecContext(ctx, q, userID, digest)
	if err != nil {
		return fmt.Errorf("delete the user record: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if count == 0 {
		return ErrInvalidCredentials
	}

	return nil
}

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

type GetAccountDeleteHandler struct {
	t *template.Template
}

func NewGetAccountDeleteHandler() *GetAccountDeleteHandler {
	return &GetAccountDeleteHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/delete_account.html")),
	}
}

func (h *GetAccountDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
			UserName:   GetUserFromContext(r.Context()).Name,
		},
		ErrorMessage: "",
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

type PostAccountDeleteHandler struct {
	db *sql.DB
	t  *template.Template
}

func NewPostAccountDeleteHandler(db *sql.DB) *PostAccountDeleteHandler {
	return &PostAccountDeleteHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/layout.html", "templates/delete_account.html")),
	}
}

func (h *PostAccountDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		var ErrMaxBytesExceeded *http.MaxBytesError
		if errors.As(err, &ErrMaxBytesExceeded) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		Handle400(w, r)
		return
	}

	errs := make([]string, 0)

	password := r.PostForm.Get("password")
	if password == "" {
		errs = append(errs, "password is required")
	}
	if len(password) > 72 {
		errs = append(errs, "password too long")
	}

	confirmation := r.PostForm.Get("confirmation")
	if confirmation != "delete account" {
		errs = append(errs, "confirmation is required")
	}

	if len(errs) > 0 {
		data := struct {
			LayoutData
			ErrorMessage string
		}{
			LayoutData: LayoutData{
				Authorized: IsAuthorized(r.Context()),
				UserName:   GetUserFromContext(r.Context()).Name,
			},
			ErrorMessage: strings.Join(errs, "; "),
		}

		var buf bytes.Buffer
		if err := h.t.Execute(&buf, &data); err != nil {
			slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
			Handle500(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		w.WriteHeader(http.StatusBadRequest)
		if _, err := buf.WriteTo(w); err != nil {
			slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
			return
		}

		return
	}

	user := GetUserFromContext(r.Context())
	if err := DeleteAccount(r.Context(), h.db, user.ID, password); err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			data := struct {
				LayoutData
				ErrorMessage string
			}{
				LayoutData: LayoutData{
					Authorized: IsAuthorized(r.Context()),
					UserName:   GetUserFromContext(r.Context()).Name,
				},
				ErrorMessage: err.Error(),
			}

			var buf bytes.Buffer
			if err := h.t.Execute(&buf, &data); err != nil {
				slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			w.WriteHeader(http.StatusBadRequest)
			if _, err := buf.WriteTo(w); err != nil {
				slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
				return
			}

			return
		}

		slog.ErrorContext(r.Context(), "delete account", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))
	http.Redirect(w, r, "/account/delete/success", http.StatusSeeOther)
}

type GetAccountDeleteSuccessHandler struct {
	t *template.Template
}

func NewGetAccountDeleteSuccessHandler() *GetAccountDeleteSuccessHandler {
	return &GetAccountDeleteSuccessHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/delete_account_success.html")),
	}
}

func (h *GetAccountDeleteSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
