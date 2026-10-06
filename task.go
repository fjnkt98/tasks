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
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrBadRequest = errors.New("bad request")

// ---------- List Tasks ----------
type ListTasksHandler struct {
	db           *sql.DB
	templateHTTP *template.Template
	templateHTMX *template.Template
}

func NewListTasksHandler(db *sql.DB) *ListTasksHandler {
	return &ListTasksHandler{
		db:           db,
		templateHTTP: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/tasks.html", "templates/partials/tasks.html")),
		templateHTMX: template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

type ListTasksParams struct {
	Page   int
	Limit  int
	Status string
}

type Task struct {
	ID     int
	UserID int
	Title  string
	Status string
}

type TaskData struct {
	LayoutData
	Tasks     []Task
	LastIndex int
	NextPage  int
	Limit     int
	Status    string
}

func (h *ListTasksHandler) GetParams(values url.Values) ListTasksParams {
	var err error

	var page int
	if p := values.Get("page"); p == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(p)
		if err != nil {
			page = 1
		}
		if page <= 0 {
			page = 1
		}
		if page > 1000 {
			page = 1000
		}
	}

	var limit int
	if l := values.Get("limit"); l == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(l)
		if err != nil {
			limit = 10
		}
		if limit <= 0 {
			limit = 10
		}
		if limit > 1000 {
			limit = 1000
		}
	}

	status := values.Get("status")

	return ListTasksParams{
		Page:   page,
		Limit:  limit,
		Status: status,
	}
}

func (h *ListTasksHandler) GetTasks(ctx context.Context, userID int, params ListTasksParams) ([]Task, error) {
	var rows *sql.Rows
	var err error

	limit := params.Limit
	offset := params.Limit * (params.Page - 1)

	if status := params.Status; status == "" {
		q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
		rows, err = h.db.QueryContext(ctx, q, userID, limit, offset)
	} else {
		q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND status = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
		rows, err = h.db.QueryContext(ctx, q, userID, status, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close() // nolint:errcheck

	tasks := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}

func (h *ListTasksHandler) ResponseHTTP(w http.ResponseWriter, data TaskData) error {
	var buf bytes.Buffer
	if err := h.templateHTTP.Execute(&buf, &data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *ListTasksHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	var buf bytes.Buffer
	if err := h.templateHTMX.ExecuteTemplate(&buf, "tasks", &data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
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
	if params.Page >= 1000 {
		data.LastIndex = -1
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
	t  *template.Template
}

func NewGetTaskHandler(db *sql.DB) *GetTaskHandler {
	return &GetTaskHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

func (h *GetTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *GetTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := "SELECT id, user_id, title, status FROM tasks WHERE id = ? AND user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, id, userID)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
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
		Tasks:     []Task{task},
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
	db *sql.DB
	t  *template.Template
}

func NewGetTaskEditHandler(db *sql.DB) *GetTaskEditHandler {
	return &GetTaskEditHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/partials/task_edit.html")),
	}
}

func (h *GetTaskEditHandler) ResponseHTMX(w http.ResponseWriter, task Task) error {
	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "task_edit", &task); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *GetTaskEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := "SELECT id, user_id, title, status FROM tasks WHERE id = ? AND user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, id, userID)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
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
	t  *template.Template
}

func NewPostTaskHandler(db *sql.DB) *PostTaskHandler {
	return &PostTaskHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

type CreateTaskParams struct {
	Title string
}

func (h *PostTaskHandler) GetParams(r *http.Request) (CreateTaskParams, error) {
	title := strings.TrimSpace(r.PostForm.Get("title"))
	if utf8.RuneCountInString(title) == 0 {
		return CreateTaskParams{}, fmt.Errorf("title required: %w", ErrBadRequest)
	}

	return CreateTaskParams{
		Title: title,
	}, nil
}

func (h *PostTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusCreated)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *PostTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		Handle400(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := "INSERT INTO tasks (user_id, title) VALUES (?, ?) RETURNING id, user_id, title, status"
	row := h.db.QueryRowContext(r.Context(), q, userID, params.Title)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []Task{task},
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
	t  *template.Template
}

func NewPutTaskHandler(db *sql.DB) *PutTaskHandler {
	return &PutTaskHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

type UpdateTaskParams struct {
	Title  string
	Status string
}

func (h *PutTaskHandler) GetParams(r *http.Request) UpdateTaskParams {
	return UpdateTaskParams{
		Title:  strings.TrimSpace(r.PostForm.Get("title")),
		Status: r.PostForm.Get("status"),
	}
}

func (h *PutTaskHandler) UpdateTask(ctx context.Context, id int, userID int, params UpdateTaskParams) (*Task, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := "SELECT id, user_id, title, status FROM tasks WHERE id = ? AND user_id = ?"
	row := tx.QueryRowContext(ctx, q, id, userID)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	if params.Title == "" && params.Status == "" {
		return &task, nil
	} else if params.Title == "" {
		q = "UPDATE tasks SET status = ?, updated_at = UNIXEPOCH() WHERE id = ? AND user_id = ? RETURNING id, user_id, title, status"
		row = tx.QueryRowContext(ctx, q, params.Status, id, userID)
	} else if params.Status == "" {
		q = "UPDATE tasks SET title = ?, updated_at = UNIXEPOCH() WHERE id = ? AND user_id = ? RETURNING id, user_id, title, status"
		row = tx.QueryRowContext(ctx, q, params.Title, id, userID)
	} else {
		q = "UPDATE tasks SET title = ?, status = ?, updated_at = UNIXEPOCH() WHERE id = ? AND user_id = ? RETURNING id, user_id, title, status"
		row = tx.QueryRowContext(ctx, q, params.Title, params.Status, id, userID)
	}

	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &task, nil
}

func (h *PutTaskHandler) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

func (h *PutTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		var ErrMaxBytesExceeded *http.MaxBytesError
		if errors.As(err, &ErrMaxBytesExceeded) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		Handle400(w, r)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	params := h.GetParams(r)
	t, err := h.UpdateTask(r.Context(), id, userID, params)
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
		Tasks:     []Task{*t},
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
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	q := "DELETE FROM tasks WHERE id = ? AND user_id = ?"
	if _, err := h.db.ExecContext(r.Context(), q, id, userID); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
