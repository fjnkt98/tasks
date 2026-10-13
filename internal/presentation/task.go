package presentation

import (
	"database/sql"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/fjnkt98/tasks/internal/usecases"
)

type TaskData struct {
	LayoutData
	Tasks     []usecases.Task
	LastIndex int
	NextPage  int
	Limit     int
	Status    string
}

type TaskHandler struct {
	tasks usecases.TaskUsecase
}

func NewTaskHandler(db *sql.DB) *TaskHandler {
	return &TaskHandler{
		tasks: usecases.NewTaskUsecase(db),
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
	if !slices.Contains([]string{"created", "done"}, status) {
		status = ""
	}

	tasks, err := h.tasks.List(r.Context(), user.ID, status, limit, page)
	if err != nil {
		slog.ErrorContext(r.Context(), "list tasks", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
			UserName:   GetUserFromContext(r.Context()).Name,
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
		body, err = RenderTemplate("partial/tasks", &data)
	} else {
		body, err = RenderTemplate("page/tasks", &data)
	}

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

func (h *TaskHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Get(r.Context(), user.ID, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "get task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
			UserName:   GetUserFromContext(r.Context()).Name,
		},
		Tasks:     []usecases.Task{*task},
		LastIndex: -1,
	}

	body, err := RenderTemplate("partial/tasks", &data)
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

func (h *TaskHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Get(r.Context(), user.ID, id)
	if err != nil {
		slog.ErrorContext(r.Context(), "get task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	body, err := RenderTemplate("partial/task_edit", task)
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

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.PostForm.Get("title"))
	if title == "" {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.Add(r.Context(), user.ID, title)
	if err != nil {
		slog.ErrorContext(r.Context(), "add task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []usecases.Task{*task},
		LastIndex: -1,
	}

	body, err := RenderTemplate("partial/tasks", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
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
		Handle404(w, r)
		return
	}

	title := strings.TrimSpace(r.PostForm.Get("title"))
	if title == "" {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	task, err := h.tasks.UpdateTitle(r.Context(), user.ID, id, title)
	if err != nil {
		slog.ErrorContext(r.Context(), "update task title", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []usecases.Task{*task},
		LastIndex: -1,
	}

	body, err := RenderTemplate("partial/tasks", &data)
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
func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	status := r.PostForm.Get("status")
	if !slices.Contains([]string{"created", "done"}, status) {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	if _, err := h.tasks.UpdateStatus(r.Context(), user.ID, id, status); err != nil {
		slog.ErrorContext(r.Context(), "update task status", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	if err := h.tasks.Delete(r.Context(), user.ID, id); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
