package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

type Activity struct {
	UserID    int
	Kind      string
	Message   string
	CreatedAt int64
}

func SaveActivity(ctx context.Context, db DBTX, userID int, kind string, message string) error {
	q := "INSERT INTO activities (user_id, kind, message) VALUES (?, ?, ?)"
	_, err := db.ExecContext(ctx, q, userID, kind, message)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}
	return nil
}

func GetActivities(ctx context.Context, db *sql.DB, userID int) (activities []Activity, err error) {
	q := "SELECT kind, message, created_at FROM activities WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT 100"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("select activities: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	activities = make([]Activity, 0)
	for rows.Next() {
		activity := Activity{UserID: userID}
		if err := rows.Scan(&activity.Kind, &activity.Message, &activity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return
}

type GetMeHandler struct {
	db *sql.DB
}

func NewGetMeHandler(db *sql.DB) *GetMeHandler {
	return &GetMeHandler{
		db: db,
	}
}

func (h GetMeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	rawActivities, err := GetActivities(r.Context(), h.db, user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get activities", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	type ActivityWithDate struct {
		UserID    int
		Kind      string
		Message   string
		CreatedAt time.Time
	}

	activities := make([]ActivityWithDate, len(rawActivities))
	for i, a := range rawActivities {
		activities[i] = ActivityWithDate{
			UserID:    a.UserID,
			Kind:      a.Kind,
			Message:   a.Message,
			CreatedAt: time.Unix(a.CreatedAt, 0).UTC(),
		}
	}

	data := struct {
		LayoutData
		IsAdmin    bool
		TaskCount  int
		DoneCount  int
		Activities []ActivityWithDate
	}{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
			UserName:   user.Name,
		},
		IsAdmin:    IsAdmin(r.Context()),
		TaskCount:  0,
		DoneCount:  0,
		Activities: activities,
	}

	q := "SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0) FROM tasks WHERE user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, user.ID)
	if err := row.Scan(&data.TaskCount, &data.DoneCount); err != nil {
		slog.ErrorContext(r.Context(), "get counts", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	body, err := RenderTemplate("page/account", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type GetAccountDeleteHandler struct{}

func NewGetAccountDeleteHandler() *GetAccountDeleteHandler {
	return &GetAccountDeleteHandler{}
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

	body, err := RenderTemplate("page/delete_account", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type PostAccountDeleteHandler struct {
	db *sql.DB
}

func NewPostAccountDeleteHandler(db *sql.DB) *PostAccountDeleteHandler {
	return &PostAccountDeleteHandler{
		db: db,
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

		body, err := RenderTemplate("page/delete_account", &data)
		if err != nil {
			slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
			Handle500(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write(body); err != nil {
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

			body, err := RenderTemplate("page/delete_account", &data)
			if err != nil {
				slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write(body); err != nil {
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

type GetAccountDeleteSuccessHandler struct{}

func NewGetAccountDeleteSuccessHandler() *GetAccountDeleteSuccessHandler {
	return &GetAccountDeleteSuccessHandler{}
}

func (h *GetAccountDeleteSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	body, err := RenderTemplate("page/delete_account_success", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
