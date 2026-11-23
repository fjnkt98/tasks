package web

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecase"
)

type UserHandler struct {
	users usecase.UserUsecase
}

func NewUserHandler(users usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		users: users,
	}
}

func (h *UserHandler) Index(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	var err error

	page := 1
	if p := values.Get("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			page = 1
		}
		page = max(page, 1)
	}
	if page > 1000 {
		Error(w, r, ErrorCode400)
		return
	}

	users, err := h.users.List(r.Context(), page, 100)
	if err != nil {
		slog.ErrorContext(r.Context(), "select users", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	data := struct {
		LayoutData
		Users   []domain.User
		Prev    int
		Current int
		Next    int
		HasPrev bool
		HasNext bool
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		Users:   users.Users,
		Prev:    max(page-1, 1),
		Current: page,
		Next:    page + 1,
		HasPrev: page-1 > 0,
		HasNext: users.HasNext && page < 1000,
	}

	var body []byte
	if IsHTMX(r) {
		body, err = RenderTemplate("partial/users", &data)
	} else {
		body, err = RenderTemplate("page/users", &data)
	}
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

func (h *UserHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	detail, err := h.users.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			Error(w, r, ErrorCode404)
			return
		}
		slog.ErrorContext(r.Context(), "get user", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	roles := make([]string, len(detail.Roles))
	for i, role := range detail.Roles {
		roles[i] = string(role)
	}

	data := struct {
		LayoutData
		UserID     int
		UserName   string
		Roles      []string
		TaskCount  int
		DoneCount  int
		Activities []domain.Activity
	}{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		UserID:     detail.User.ID(),
		UserName:   detail.User.Name(),
		Roles:      roles,
		TaskCount:  detail.TaskCount,
		DoneCount:  detail.DoneCount,
		Activities: detail.Activities,
	}

	body, err := RenderTemplate("page/user", data)
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
