package presentation

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/fjnkt98/tasks/internal/usecases"
)

type contextKey int

const (
	contextKeyUser contextKey = iota
)

type User struct {
	ID    int
	Name  string
	Roles []string
}

func SetUserIntoContext(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, contextKeyUser, user)
}

func GetUserFromContext(ctx context.Context) User {
	value, ok := ctx.Value(contextKeyUser).(User)
	if !ok {
		return User{}
	}
	return value
}

func IsAuthorized(ctx context.Context) bool {
	return GetUserFromContext(ctx).ID != 0
}

func IsAdmin(ctx context.Context) bool {
	user := GetUserFromContext(ctx)
	return slices.Contains(user.Roles, "admin")
}

const AuthCookieName = "session_token"

func NewAuthCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     AuthCookieName,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   os.Getenv("USE_SECURE_COOKIE") == "true",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

type SignupHandler struct {
	accounts usecases.AccountUsecase
}

func NewSignupHandler(db *sql.DB) *SignupHandler {
	return &SignupHandler{
		accounts: usecases.NewAccountUsecase(db),
	}
}

func (h *SignupHandler) render(w http.ResponseWriter, r *http.Request, msg string) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		ErrorMessage: msg,
	}

	body, err := RenderTemplate("page/signup", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if msg == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *SignupHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "")
}

func (h *SignupHandler) Post(w http.ResponseWriter, r *http.Request) {
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

	if password != confirmPassword {
		errs = append(errs, "password not match")
	}

	if len(errs) > 0 {
		h.render(w, r, strings.Join(errs, "; "))
		return
	}

	if err := h.accounts.Signup(r.Context(), username, password); err != nil {
		if errors.Is(err, usecases.ErrUserAlreadyExists) {
			h.render(w, r, "username was already used")
			return
		}
		slog.ErrorContext(r.Context(), "request failed", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	http.Redirect(w, r, "/account/signup/success", http.StatusSeeOther)
}

func (h *SignupHandler) Success(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized:  IsAuthorized(r.Context()),
		AccountName: GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/signup_success", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type SigninHandler struct {
	accounts usecases.AccountUsecase
}

func NewSigninHandler(db *sql.DB) *SigninHandler {
	return &SigninHandler{
		accounts: usecases.NewAccountUsecase(db),
	}
}

func (h *SigninHandler) render(w http.ResponseWriter, r *http.Request, msg string) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		ErrorMessage: msg,
	}

	body, err := RenderTemplate("page/signin", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if msg == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *SigninHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "")
}

func (h *SigninHandler) Post(w http.ResponseWriter, r *http.Request) {
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
		h.render(w, r, strings.Join(errs, "; "))
		return
	}

	token, err := h.accounts.Signin(r.Context(), username, password)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			h.render(w, r, "invalid credentials")
			return
		}
		slog.ErrorContext(r.Context(), "signin", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	http.SetCookie(w, NewAuthCookie(token, 86400))
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

type ChangePasswordHandler struct {
	accounts usecases.AccountUsecase
}

func NewChangePasswordHandler(db *sql.DB) *ChangePasswordHandler {
	return &ChangePasswordHandler{
		accounts: usecases.NewAccountUsecase(db),
	}
}

func (h *ChangePasswordHandler) render(w http.ResponseWriter, r *http.Request, msg string) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		ErrorMessage: msg,
	}

	body, err := RenderTemplate("page/change_password", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if msg == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *ChangePasswordHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "")
}

func (h *ChangePasswordHandler) Post(w http.ResponseWriter, r *http.Request) {
	errs := make([]string, 0)
	currentPassword := r.PostForm.Get("current-password")
	if currentPassword == "" {
		errs = append(errs, "current password is required")
	}
	if len(currentPassword) > 72 {
		errs = append(errs, "current password too long")
	}

	newPassword := r.PostForm.Get("new-password")
	if newPassword == "" {
		errs = append(errs, "new password is required")
	}
	if len(newPassword) > 72 {
		errs = append(errs, "new password too long")
	}

	confirmNewPassword := r.PostForm.Get("confirm-new-password")
	if confirmNewPassword == "" {
		errs = append(errs, "confirm new password is required")
	}

	if newPassword != confirmNewPassword {
		errs = append(errs, "password not match")
	}

	if len(errs) > 0 {
		h.render(w, r, strings.Join(errs, "; "))
		return
	}

	user := GetUserFromContext(r.Context())

	if err := h.accounts.ChangePassword(r.Context(), user.ID, currentPassword, newPassword); err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			h.render(w, r, "invalid credentials")
			return
		}
		slog.ErrorContext(r.Context(), "change password", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))
	http.Redirect(w, r, "/account/password_change/success", http.StatusSeeOther)
}

func (h *ChangePasswordHandler) Success(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized:  IsAuthorized(r.Context()),
		AccountName: GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/change_password_success", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type SignoutHandler struct {
	accounts usecases.AccountUsecase
}

func NewSignoutHandler(db *sql.DB) *SignoutHandler {
	return &SignoutHandler{
		accounts: usecases.NewAccountUsecase(db),
	}
}

func (h *SignoutHandler) Post(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if err := h.accounts.Signout(r.Context(), cookie.Value); err != nil {
		slog.ErrorContext(r.Context(), "signout", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type AccountHandler struct {
	accounts usecases.AccountUsecase
	users    usecases.UserUsecase
	tasks    usecases.TaskUsecase
}

func NewAccountHandler(db *sql.DB) *AccountHandler {
	return &AccountHandler{
		accounts: usecases.NewAccountUsecase(db),
		users:    usecases.NewUserUsecase(db),
		tasks:    usecases.NewTaskUsecase(db),
	}
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	detail, err := h.users.Get(r.Context(), user.ID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get user detail", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	data := struct {
		LayoutData
		IsAdmin    bool
		TaskCount  int
		DoneCount  int
		Activities []usecases.Activity
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: user.Name,
		},
		IsAdmin:    IsAdmin(r.Context()),
		TaskCount:  detail.TaskCount,
		DoneCount:  detail.DoneCount,
		Activities: detail.Activities,
	}

	body, err := RenderTemplate("page/account", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type DeleteAccountHandler struct {
	accounts usecases.AccountUsecase
}

func NewDeleteAccountHandler(db *sql.DB) *DeleteAccountHandler {
	return &DeleteAccountHandler{
		accounts: usecases.NewAccountUsecase(db),
	}
}

func (h *DeleteAccountHandler) render(w http.ResponseWriter, r *http.Request, msg string) {
	data := struct {
		LayoutData
		ErrorMessage string
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		ErrorMessage: msg,
	}

	body, err := RenderTemplate("page/delete_account", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if msg == "" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *DeleteAccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "")
}

func (h *DeleteAccountHandler) Post(w http.ResponseWriter, r *http.Request) {
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
		h.render(w, r, strings.Join(errs, "; "))
		return
	}

	user := GetUserFromContext(r.Context())
	if err := h.accounts.Delete(r.Context(), user.ID, password); err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			h.render(w, r, "invalid credentials")
			return
		}

		slog.ErrorContext(r.Context(), "delete account", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	http.SetCookie(w, NewAuthCookie("", -1))
	http.Redirect(w, r, "/account/delete/success", http.StatusSeeOther)
}

func (h *DeleteAccountHandler) Success(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	body, err := RenderTemplate("page/delete_account_success", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
