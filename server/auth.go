package server

import (
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
	"unicode/utf8"

	"github.com/fjnkt98/tasks/repository"
	"github.com/fjnkt98/tasks/settings"
	"golang.org/x/crypto/bcrypt"
)

var ErrDuplicatedUsername = errors.New("username was already used")
var ErrInvalidCredentials = errors.New("invalid credentials")

func NewSessionToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func SetUserIDIntoContext(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, contextKeyUser, userID)
}

func GetUserIDFromContext(ctx context.Context) int64 {
	value, ok := ctx.Value(contextKeyUser).(int64)
	if !ok {
		return 0
	}
	return value
}

func IsAuthorized(ctx context.Context) bool {
	return GetUserIDFromContext(ctx) != 0
}

// ---------- Signup ----------
type SignupData struct {
	LayoutData
	ErrorMessage string
}

type GetSignupHandler struct{}

func NewGetSignupHandler() *GetSignupHandler {
	return &GetSignupHandler{}
}

func (h *GetSignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/signup.html")
	if err != nil {
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := SignupData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
		},
		ErrorMessage: "",
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

type PostSignupHandler struct {
	db *sql.DB
}

func NewPostSignupHandler(db *sql.DB) *PostSignupHandler {
	return &PostSignupHandler{
		db: db,
	}
}

type SignupParams struct {
	Username string
	Password string
}

func (h *PostSignupHandler) GetParams(r *http.Request) (SignupParams, error) {
	errs := make([]string, 0)

	username := strings.TrimSpace(r.FormValue("username"))
	if utf8.RuneCountInString(username) == 0 {
		errs = append(errs, "username is required")
	}

	password := strings.TrimSpace(r.FormValue("password"))
	if utf8.RuneCountInString(password) == 0 {
		errs = append(errs, "password is required")
	}

	confirmPassword := strings.TrimSpace(r.FormValue("confirm-password"))
	if utf8.RuneCountInString(confirmPassword) == 0 {
		errs = append(errs, "confirm-password is required")
	}

	if password != "" && confirmPassword != "" && password != confirmPassword {
		errs = append(errs, "password not match")
	}

	if len(errs) > 0 {
		return SignupParams{}, errors.New(strings.Join(errs, "; "))
	}

	return SignupParams{
		Username: username,
		Password: password,
	}, nil
}

func (h *PostSignupHandler) Signup(ctx context.Context, params SignupParams) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	q := repository.New(h.db).WithTx(tx)

	if _, err := q.GetUserByName(ctx, params.Username); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("get user by name: %w", err)
		}
	} else {
		return ErrDuplicatedUsername
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}

	if err := q.CreateUser(ctx, repository.CreateUserParams{
		Name:     params.Username,
		Password: string(digest),
	}); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (h *PostSignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := h.GetParams(r)
	if err != nil {
		msg := err.Error()

		t, err := template.ParseFS(templates, "templates/layout.html", "templates/signup.html")
		if err != nil {
			slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
			Handle500(w, r)
			return
		}

		data := SignupData{
			LayoutData: LayoutData{
				Authorized: IsAuthorized(r.Context()),
			},
			ErrorMessage: msg,
		}

		w.WriteHeader(http.StatusBadRequest)
		if err := t.Execute(w, &data); err != nil {
			slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
			Handle500(w, r)
			return
		}
		return
	}

	if err := h.Signup(r.Context(), params); err != nil {
		if errors.Is(err, ErrDuplicatedUsername) {
			msg := err.Error()

			t, err := template.ParseFS(templates, "templates/layout.html", "templates/signup.html")
			if err != nil {
				slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			data := SignupData{
				LayoutData: LayoutData{
					Authorized: IsAuthorized(r.Context()),
				},
				ErrorMessage: msg,
			}

			w.WriteHeader(http.StatusBadRequest)
			if err := t.Execute(w, &data); err != nil {
				slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
				Handle500(w, r)
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

type GetSignupSuccessHandler struct{}

func NewGetSignupSuccessHandler() *GetSignupSuccessHandler {
	return &GetSignupSuccessHandler{}
}

func (h *GetSignupSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/signup_success.html")
	if err != nil {
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := SignupData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
		},
		ErrorMessage: "",
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Signin ----------
type SigninData struct {
	LayoutData
	ErrorMessage string
}

type GetSigninHandler struct{}

func NewGetSigninHandler() *GetSigninHandler {
	return &GetSigninHandler{}
}

func (h *GetSigninHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/signin.html")
	if err != nil {
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := SigninData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
		},
		ErrorMessage: "",
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

type PostSigninHandler struct {
	db *sql.DB
}

func NewPostSigninHandler(db *sql.DB) *PostSigninHandler {
	return &PostSigninHandler{
		db: db,
	}
}

type SigninParams struct {
	Username string
	Password string
}

func (h *PostSigninHandler) GetParams(r *http.Request) (SigninParams, error) {
	errs := make([]string, 0)

	username := strings.TrimSpace(r.FormValue("username"))
	if utf8.RuneCountInString(username) == 0 {
		errs = append(errs, "username is required")
	}

	password := strings.TrimSpace(r.FormValue("password"))
	if utf8.RuneCountInString(password) == 0 {
		errs = append(errs, "password is required")
	}

	if len(errs) > 0 {
		return SigninParams{}, errors.New(strings.Join(errs, "; "))
	}

	return SigninParams{
		Username: username,
		Password: password,
	}, nil
}

func (h *PostSigninHandler) Signin(ctx context.Context, params SigninParams) (string, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := repository.New(h.db).WithTx(tx)

	user, err := q.GetUserByName(ctx, params.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := NewSessionToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	_, err = q.CreateSession(ctx, repository.CreateSessionParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return token, nil
}

func (h *PostSigninHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := h.GetParams(r)
	if err != nil {
		msg := err.Error()

		t, err := template.ParseFS(templates, "templates/layout.html", "templates/signin.html")
		if err != nil {
			slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
			Handle500(w, r)
			return
		}

		data := SigninData{
			LayoutData: LayoutData{
				Authorized: IsAuthorized(r.Context()),
			},
			ErrorMessage: msg,
		}

		w.WriteHeader(http.StatusBadRequest)
		if err := t.Execute(w, &data); err != nil {
			slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
			Handle500(w, r)
			return
		}
		return
	}

	token, err := h.Signin(r.Context(), params)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			msg := err.Error()

			t, err := template.ParseFS(templates, "templates/layout.html", "templates/signin.html")
			if err != nil {
				slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			data := SigninData{
				LayoutData: LayoutData{
					Authorized: IsAuthorized(r.Context()),
				},
				ErrorMessage: msg,
			}

			w.WriteHeader(http.StatusBadRequest)
			if err := t.Execute(w, &data); err != nil {
				slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
				Handle500(w, r)
				return
			}
			return
		}

		slog.ErrorContext(r.Context(), "signin", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	cookie := http.Cookie{
		Name:     "session_token",
		Value:    token,
		MaxAge:   86400,
		Path:     "/",
		Secure:   settings.UseSecureCookie,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, &cookie)

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

// ---------- Signout ----------
type GetSignoutHandler struct {
	db *sql.DB
}

func NewGetSignoutHandler(db *sql.DB) *GetSignoutHandler {
	return &GetSignoutHandler{
		db: db,
	}
}

func (h *GetSignoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := repository.New(h.db)

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := q.DeleteSessionByToken(r.Context(), cookie.Value); err != nil {
		slog.ErrorContext(r.Context(), "delete session by token", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	cookie.Value = ""
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
