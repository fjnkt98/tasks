package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fjnkt98/tasks/repository"
)

// ---------- List Tasks ----------
type ListTasksHandler struct {
	db *sql.DB
}

func NewListTasksHandler(db *sql.DB) *ListTasksHandler {
	return &ListTasksHandler{
		db: db,
	}
}

type ListTasksParams struct {
	Page   int64
	Limit  int64
	Status string
}

type TaskData struct {
	LayoutData
	Tasks     []repository.Task
	LastIndex int
	NextPage  int64
	Limit     int64
	Status    string
}

func (h *ListTasksHandler) GetParams(values url.Values) ListTasksParams {
	var err error

	var page int64
	if p := values.Get("page"); p == "" {
		page = 1
	} else {
		page, err = strconv.ParseInt(p, 10, 64)
		if err != nil {
			page = 1
		}
		if page <= 0 {
			page = 1
		}
	}

	var limit int64
	if l := values.Get("limit"); l == "" {
		limit = 10
	} else {
		limit, err = strconv.ParseInt(l, 10, 64)
		if err != nil {
			limit = 10
		}
		if limit <= 0 {
			limit = 10
		}
	}

	status := values.Get("status")

	return ListTasksParams{
		Page:   page,
		Limit:  limit,
		Status: status,
	}
}

func (h *ListTasksHandler) GetTasks(ctx context.Context, userID int64, params ListTasksParams) ([]repository.Task, error) {
	q := repository.New(h.db)

	var tasks []repository.Task
	var err error
	if params.Status == "" {
		tasks, err = q.ListTasks(ctx, repository.ListTasksParams{
			UserID: userID,
			Offset: params.Limit * (params.Page - 1),
			Limit:  params.Limit,
		})
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
	} else {
		tasks, err = q.ListTasksByStatus(ctx, repository.ListTasksByStatusParams{
			UserID: userID,
			Offset: params.Limit * (params.Page - 1),
			Limit:  params.Limit,
			Status: params.Status,
		})
		if err != nil {
			return nil, fmt.Errorf("list tasks by status: %w", err)
		}
	}

	return tasks, nil
}

func (h *ListTasksHandler) ResponseHTTP(w http.ResponseWriter, data TaskData) error {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/tasks.html", "templates/partials/tasks.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *ListTasksHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *ListTasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	params := h.GetParams(r.URL.Query())
	tasks, err := h.GetTasks(r.Context(), userID, params)
	if err != nil {
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
		},
		Tasks:     tasks,
		LastIndex: len(tasks) - 1,
		NextPage:  params.Page + 1,
		Limit:     params.Limit,
		Status:    params.Status,
	}

	if IsHTMX(r) {
		if err := h.ResponseHTMX(w, data); err != nil {
			slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
			Handle500(w, r)
			return
		}
	} else {
		if err := h.ResponseHTTP(w, data); err != nil {
			slog.ErrorContext(r.Context(), "response for http", slog.Any("error", err))
			Handle500(w, r)
			return
		}
	}
}

// ---------- Get Task ----------
type GetTaskHandler struct {
	db *sql.DB
}

func NewGetTaskHandler(db *sql.DB) *GetTaskHandler {
	return &GetTaskHandler{
		db: db,
	}
}

func (h *GetTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *GetTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := repository.New(h.db)
	task, err := q.GetTaskByID(r.Context(), repository.GetTaskByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		LayoutData: LayoutData{
			Authorized: IsAuthorized(r.Context()),
		},
		Tasks:     []repository.Task{task},
		LastIndex: -1,
	}

	if err := h.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Get Task Edit ----------
type GetTaskEditHandler struct {
	q *repository.Queries
}

func NewGetTaskEditHandler(db *sql.DB) *GetTaskEditHandler {
	return &GetTaskEditHandler{
		q: repository.New(db),
	}
}

func (h *GetTaskEditHandler) ResponseHTMX(w http.ResponseWriter, task repository.Task) error {
	t, err := template.ParseFS(templates, "templates/partials/task_edit.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := t.ExecuteTemplate(w, "task_edit", &task); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *GetTaskEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	task, err := h.q.GetTaskByID(r.Context(), repository.GetTaskByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	if err := h.ResponseHTMX(w, task); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Post Task ----------
type PostTaskHandler struct {
	db *sql.DB
}

func NewPostTaskHandler(db *sql.DB) *PostTaskHandler {
	return &PostTaskHandler{
		db: db,
	}
}

type CreateTaskParams struct {
	Title string
}

func (h *PostTaskHandler) GetParams(r *http.Request) (CreateTaskParams, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if utf8.RuneCountInString(title) == 0 {
		return CreateTaskParams{}, fmt.Errorf("title required: %w", ErrBadRequest)
	}

	return CreateTaskParams{
		Title: title,
	}, nil
}

func (h *PostTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusCreated)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *PostTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := h.GetParams(r)
	if err != nil {
		Handle400(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := repository.New(h.db)
	task, err := q.CreateTask(r.Context(), repository.CreateTaskParams{
		Title:  params.Title,
		UserID: userID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []repository.Task{task},
		LastIndex: -1,
	}

	if err := h.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Put Task ----------
type PutTaskHandler struct {
	db *sql.DB
}

func NewPutTaskHandler(db *sql.DB) *PutTaskHandler {
	return &PutTaskHandler{
		db: db,
	}
}

type UpdateTaskParams struct {
	Title  string
	Status string
}

func (h *PutTaskHandler) GetParams(r *http.Request) UpdateTaskParams {
	return UpdateTaskParams{
		Title:  strings.TrimSpace(r.FormValue("title")),
		Status: r.FormValue("status"),
	}
}

func (h *PutTaskHandler) UpdateTask(ctx context.Context, id int64, userID int64, params UpdateTaskParams) (repository.Task, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Task{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := repository.New(h.db).WithTx(tx)
	task, err := q.GetTaskByID(ctx, repository.GetTaskByIDParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Task{}, err
		}
		return repository.Task{}, fmt.Errorf("get task by id: %w", err)
	}

	data := repository.UpdateTaskParams{
		ID:     id,
		UserID: userID,
	}
	if params.Title != "" {
		data.Title = params.Title
	} else {
		data.Title = task.Title
	}
	if params.Status != "" {
		data.Status = params.Status
	} else {
		data.Status = task.Status
	}

	task, err = q.UpdateTask(ctx, data)
	if err != nil {
		return repository.Task{}, fmt.Errorf("update task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return repository.Task{}, fmt.Errorf("commit transaction: %w", err)
	}

	return task, nil
}

func (h *PutTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *PutTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	params := h.GetParams(r)
	task, err := h.UpdateTask(r.Context(), id, userID, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}
		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []repository.Task{task},
		LastIndex: -1,
	}

	if err := h.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Delete Task ----------
type DeleteTaskHandler struct {
	db *sql.DB
}

func NewDeleteTaskHandler(db *sql.DB) *DeleteTaskHandler {
	return &DeleteTaskHandler{
		db: db,
	}
}

func (h *DeleteTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := repository.New(h.db)
	if err := q.DeleteTask(r.Context(), repository.DeleteTaskParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
