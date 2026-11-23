package controllers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
	"github.com/fjnkt98/tasks/internal/views"
)

type TaskData struct {
	LayoutData
	Tasks     []domain.Task
	LastIndex int
	NextPage  int
	Limit     int
	Status    string
}

type TaskHandler struct {
	tasks usecases.TaskUsecase
}

func NewTaskHandler(tasks usecases.TaskUsecase) *TaskHandler {
	return &TaskHandler{
		tasks: tasks,
	}
}

func (h *TaskHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	values := r.URL.Query()
	var err error

	page := 1
	if p := values.Get("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			page = 1
		}
		page = min(1000, max(1, page))
	}

	limit := 10
	if l := values.Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			limit = 10
		}
		limit = min(1000, max(1, limit))
	}

	status := values.Get("status")
	if status != "" {
		if _, err := domain.ParseTaskStatus(status); err != nil {
			status = ""
		}
	}

	tasks, err := h.tasks.List(r.Context(), user.ID, status, limit, page)
	if err != nil {
		slog.ErrorContext(r.Context(), "list tasks", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		Tasks:     tasks,
		LastIndex: len(tasks) - 1,
		NextPage:  page + 1,
		Limit:     limit,
		Status:    status,
	}
	if page >= 1000 {
		data.LastIndex = -1
	}

	var body []byte
	if IsHTMX(r) {
		body, err = views.Render("partial/tasks", &data)
	} else {
		body, err = views.Render("page/tasks", &data)
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

func (h *TaskHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Get(r.Context(), user.ID, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "get task", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized:  IsAuthorized(r.Context()),
			AccountName: GetUserFromContext(r.Context()).Name,
		},
		Tasks:     []domain.Task{task},
		LastIndex: -1,
		NextPage:  0,
		Limit:     0,
		Status:    "",
	}

	body, err := views.Render("partial/tasks", &data)
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

func (h *TaskHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Get(r.Context(), user.ID, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "get task", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	body, err := views.Render("partial/task_edit", task)
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

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.PostForm.Get("title"))
	if title == "" {
		Error(w, r, ErrorCode400)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Add(r.Context(), user.ID, title)
	if err != nil {
		MapUsecaseError(w, r, err, "add task")
		return
	}

	data := TaskData{
		LayoutData: LayoutData{Authorized: false, AccountName: ""},
		Tasks:      []domain.Task{task},
		LastIndex:  -1,
		NextPage:   0,
		Limit:      0,
		Status:     "",
	}

	body, err := views.Render("partial/tasks", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TaskHandler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	title := strings.TrimSpace(r.PostForm.Get("title"))
	if title == "" {
		Error(w, r, ErrorCode400)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.UpdateTitle(r.Context(), user.ID, id, title)
	if err != nil {
		MapUsecaseError(w, r, err, "update task title")
		return
	}

	data := TaskData{
		LayoutData: LayoutData{Authorized: false, AccountName: ""},
		Tasks:      []domain.Task{task},
		LastIndex:  -1,
		NextPage:   0,
		Limit:      0,
		Status:     "",
	}

	body, err := views.Render("partial/tasks", &data)
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
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	status := r.PostForm.Get("status")
	if _, err := domain.ParseTaskStatus(status); err != nil {
		Error(w, r, ErrorCode400)
		return
	}

	user := GetUserFromContext(r.Context())

	if _, err := h.tasks.UpdateStatus(r.Context(), user.ID, id, status); err != nil {
		MapUsecaseError(w, r, err, "update task status")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Error(w, r, ErrorCode404)
		return
	}

	user := GetUserFromContext(r.Context())

	if err := h.tasks.Delete(r.Context(), user.ID, id); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Error(w, r, ErrorCode500)
		return
	}

	w.WriteHeader(http.StatusOK)
}
