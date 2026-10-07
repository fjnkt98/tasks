package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/fjnkt98/tasks/settings"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var ErrDuplicatedUsername = errors.New("username was already used")
var ErrInvalidCredentials = errors.New("invalid credentials")

type contextKey int

const (
	contextKeyUser contextKey = iota
)

func NewSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func SetUserIDIntoContext(ctx context.Context, userID int) context.Context {
	return context.WithValue(ctx, contextKeyUser, userID)
}

func GetUserIDFromContext(ctx context.Context) int {
	value, ok := ctx.Value(contextKeyUser).(int)
	if !ok {
		return 0
	}
	return value
}

func IsAuthorized(ctx context.Context) bool {
	return GetUserIDFromContext(ctx) != 0
}

const AuthCookieName = "session_token"

func NewAuthCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     AuthCookieName,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   settings.UseSecureCookie,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func CreateUser(ctx context.Context, db *sql.DB, username string, password string) error {
	digest, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}

	q := "INSERT INTO users (name, digest) VALUES (?, ?)"
	if _, err := db.ExecContext(ctx, q, username, string(digest)); err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return ErrDuplicatedUsername
		}
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func Login(ctx context.Context, db *sql.DB, username string, password string) (string, error) {
	q := "SELECT id, digest FROM users WHERE name = ? LIMIT 1"
	row := db.QueryRowContext(ctx, q, username)
	var userID int
	var digest string
	if err := row.Scan(&userID, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(digest), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token := NewSessionToken()
	q = "INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)"
	if _, err := db.ExecContext(ctx, q, userID, token, time.Now().Add(24*time.Hour).Unix()); err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}

	return token, nil
}

type GetSignupHandler struct {
	t *template.Template
}

func NewGetSignupHandler() *GetSignupHandler {
	return &GetSignupHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signup.html")),
	}
}

func (h *GetSignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
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

type PostSignupHandler struct {
	db *sql.DB
	t  *template.Template
}

func NewPostSignupHandler(db *sql.DB) *PostSignupHandler {
	return &PostSignupHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signup.html")),
	}
}

func (h *PostSignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	username := strings.TrimSpace(r.PostForm.Get("username"))
	if username == "" {
		errs = append(errs, "username is required")
	}

	password := r.PostForm.Get("password")
	if password == "" {
		errs = append(errs, "password is required")
	}
	if len(password) > 72 {
		errs = append(errs, "password too long")
	}

	confirmPassword := r.PostForm.Get("confirm-password")
	if confirmPassword == "" {
		errs = append(errs, "confirm-password is required")
	}

	if password != "" && confirmPassword != "" && password != confirmPassword {
		errs = append(errs, "password not match")
	}

	if len(errs) > 0 {
		data := struct {
			LayoutData
			ErrorMessage string
		}{
			LayoutData: LayoutData{
				Authorized: IsAuthorized(r.Context()),
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

	if err := CreateUser(r.Context(), h.db, username, password); err != nil {
		if errors.Is(err, ErrDuplicatedUsername) {
			data := struct {
				LayoutData
				ErrorMessage string
			}{
				LayoutData: LayoutData{
					Authorized: IsAuthorized(r.Context()),
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

		slog.ErrorContext(r.Context(), "create user", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	http.Redirect(w, r, "/signup/success", http.StatusSeeOther)
}

type GetSignupSuccessHandler struct {
	t *template.Template
}

func NewGetSignupSuccessHandler() *GetSignupSuccessHandler {
	return &GetSignupSuccessHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signup_success.html")),
	}
}

func (h *GetSignupSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

type GetSigninHandler struct {
	t *template.Template
}

func NewGetSigninHandler() *GetSigninHandler {
	return &GetSigninHandler{
		t: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signin.html")),
	}
}

func (h *GetSigninHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
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

type PostSigninHandler struct {
	db *sql.DB
	t  *template.Template
}

func NewPostSigninHandler(db *sql.DB) *PostSigninHandler {
	return &PostSigninHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signin.html")),
	}
}

func (h *PostSigninHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	username := strings.TrimSpace(r.PostForm.Get("username"))
	if username == "" {
		errs = append(errs, "username is required")
	}

	password := r.PostForm.Get("password")
	if password == "" {
		errs = append(errs, "password is required")
	}
	if len(password) > 72 {
		errs = append(errs, "password too long")
	}

	if len(errs) > 0 {
		data := struct {
			LayoutData
			ErrorMessage string
		}{
			LayoutData: LayoutData{
				Authorized: IsAuthorized(r.Context()),
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

	token, err := Login(r.Context(), h.db, username, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			data := struct {
				LayoutData
				ErrorMessage string
			}{
				LayoutData: LayoutData{
					Authorized: IsAuthorized(r.Context()),
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

		slog.ErrorContext(r.Context(), "signin", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	http.SetCookie(w, NewAuthCookie(token, 86400))

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

// ---------- Signout ----------
type PostSignoutHandler struct {
	db *sql.DB
}

func NewPostSignoutHandler(db *sql.DB) *PostSignoutHandler {
	return &PostSignoutHandler{
		db: db,
	}
}

func (h *PostSignoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	q := "DELETE FROM sessions WHERE token = ?"
	if _, err := h.db.ExecContext(r.Context(), q, cookie.Value); err != nil {
		slog.ErrorContext(r.Context(), "delete session by token", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
