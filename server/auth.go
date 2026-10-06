package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fjnkt98/tasks/ent"
	"github.com/fjnkt98/tasks/ent/session"
	"github.com/fjnkt98/tasks/ent/user"
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

// ---------- Signup ----------
type SignupData struct {
	LayoutData
	ErrorMessage string
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
	data := SignupData{
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
	client *ent.Client
	t      *template.Template
}

func NewPostSignupHandler(client *ent.Client) *PostSignupHandler {
	return &PostSignupHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signup.html")),
	}
}

type SignupParams struct {
	Username string
	Password string
}

func (h *PostSignupHandler) GetParams(r *http.Request) (SignupParams, error) {
	errs := make([]string, 0)

	username := strings.TrimSpace(r.PostForm.Get("username"))
	if utf8.RuneCountInString(username) == 0 {
		errs = append(errs, "username is required")
	}

	password := r.PostForm.Get("password")
	if utf8.RuneCountInString(password) == 0 {
		errs = append(errs, "password is required")
	}
	if len(password) > 72 {
		errs = append(errs, "password too long")
	}

	confirmPassword := r.PostForm.Get("confirm-password")
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
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	_, err = tx.User.Query().Where(user.Name(params.Username)).Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return fmt.Errorf("get user by name: %w", err)
		}
	} else {
		return ErrDuplicatedUsername
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}

	if _, err := tx.User.Create().SetName(params.Username).SetPassword(string(digest)).Save(ctx); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
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

	params, err := h.GetParams(r)
	if err != nil {
		data := SignupData{
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

	if err := h.Signup(r.Context(), params); err != nil {
		if errors.Is(err, ErrDuplicatedUsername) {
			data := SignupData{
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
	data := SignupData{
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

// ---------- Signin ----------
type SigninData struct {
	LayoutData
	ErrorMessage string
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
	data := SigninData{
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
	client *ent.Client
	t      *template.Template
}

func NewPostSigninHandler(client *ent.Client) *PostSigninHandler {
	return &PostSigninHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/layout.html", "templates/signin.html")),
	}
}

type SigninParams struct {
	Username string
	Password string
}

func (h *PostSigninHandler) GetParams(r *http.Request) (SigninParams, error) {
	errs := make([]string, 0)

	username := strings.TrimSpace(r.PostForm.Get("username"))
	if utf8.RuneCountInString(username) == 0 {
		errs = append(errs, "username is required")
	}

	password := r.PostForm.Get("password")
	if utf8.RuneCountInString(password) == 0 {
		errs = append(errs, "password is required")
	}
	if len(password) > 72 {
		errs = append(errs, "password too long")
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
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	u, err := tx.User.Query().Where(user.Name(params.Username)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(params.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := NewSessionToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	_, err = tx.Session.Create().SetUserID(u.ID).SetToken(token).SetExpiresAt(time.Now().Add(24 * time.Hour)).Save(ctx)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return token, nil
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

	params, err := h.GetParams(r)
	if err != nil {

		data := SigninData{
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

	token, err := h.Signin(r.Context(), params)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			data := SigninData{
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
	client *ent.Client
}

func NewPostSignoutHandler(client *ent.Client) *PostSignoutHandler {
	return &PostSignoutHandler{
		client: client,
	}
}

func (h *PostSignoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if _, err := h.client.Session.Delete().Where(session.Token(cookie.Value)).Exec(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "delete session by token", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
