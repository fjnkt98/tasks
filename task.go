package main

import (
	"bytes"
	"database/sql"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

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

func (h *ListTasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())

	var err error
	values := r.URL.Query()
	var page int
	if p := values.Get("page"); p == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(p)
		if err != nil {
			page = 1
		}
		page = min(1000, max(1, page))
	}

	var limit int
	if l := values.Get("limit"); l == "" {
		limit = 10
	} else {
		limit, err = strconv.Atoi(l)
		if err != nil {
			limit = 10
		}
		limit = min(1000, max(1, limit))
	}

	offset := limit * (page - 1)

	status := values.Get("status")
	if !slices.Contains([]string{"created", "done"}, status) {
		status = ""
	}

	var rows *sql.Rows
	if status == "" {
		q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
		rows, err = h.db.QueryContext(r.Context(), q, user.ID, limit, offset)
	} else {
		q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND status = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
		rows, err = h.db.QueryContext(r.Context(), q, user.ID, status, limit, offset)
	}

	if err != nil {
		slog.ErrorContext(r.Context(), "select tasks", slog.Any("error", err))
		Handle500(w, r)
		return
	}
	defer rows.Close() // nolint:errcheck

	tasks := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
			slog.ErrorContext(r.Context(), "scan row", slog.Any("error", err))
			Handle500(w, r)
			return
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(r.Context(), "rows error", slog.Any("error", err))
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

	var buf bytes.Buffer
	if IsHTMX(r) {
		err = h.templateHTMX.ExecuteTemplate(&buf, "tasks", &data)
	} else {
		err = h.templateHTTP.Execute(&buf, &data)
	}
	if err != nil {
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

func (h *GetTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	q := "SELECT id, user_id, title, status FROM tasks WHERE id = ? AND user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, id, user.ID)
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
			UserName:   GetUserFromContext(r.Context()).Name,
		},
		Tasks:     []Task{task},
		LastIndex: -1,
	}

	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
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

func (h *GetTaskEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	q := "SELECT id, user_id, title, status FROM tasks WHERE id = ? AND user_id = ?"
	row := h.db.QueryRowContext(r.Context(), q, id, user.ID)
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

	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "task_edit", &task); err != nil {
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

	title := strings.TrimSpace(r.PostForm.Get("title"))
	if utf8.RuneCountInString(title) == 0 {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	q := "INSERT INTO tasks (user_id, title) VALUES (?, ?) RETURNING id, user_id, title, status"
	row := h.db.QueryRowContext(r.Context(), q, user.ID, title)
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

	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusCreated)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

type PutTaskTitleHandler struct {
	db *sql.DB
	t  *template.Template
}

func NewPutTaskTitleHandler(db *sql.DB) *PutTaskTitleHandler {
	return &PutTaskTitleHandler{
		db: db,
		t:  template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

func (h *PutTaskTitleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	title := strings.TrimSpace(r.PostForm.Get("title"))
	if utf8.RuneCountInString(title) == 0 {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	q := "UPDATE tasks SET title = ?, updated_at = UNIXEPOCH() WHERE id = ? AND user_id = ? RETURNING id, user_id, title, status"
	row := h.db.QueryRowContext(r.Context(), q, title, id, user.ID)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []Task{task},
		LastIndex: -1,
	}

	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "tasks", &data); err != nil {
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

type PutTaskStatusHandler struct {
	db *sql.DB
}

func NewPutTaskStatusHandler(db *sql.DB) *PutTaskStatusHandler {
	return &PutTaskStatusHandler{
		db: db,
	}
}

func (h *PutTaskStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	status := r.PostForm.Get("status")
	if utf8.RuneCountInString(status) == 0 || !slices.Contains([]string{"created", "done"}, status) {
		Handle400(w, r)
		return
	}

	user := GetUserFromContext(r.Context())

	q := "UPDATE tasks SET status = ?, updated_at = UNIXEPOCH() WHERE id = ? AND user_id = ? RETURNING id, user_id, title, status"
	row := h.db.QueryRowContext(r.Context(), q, status, id, user.ID)
	var task Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

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

	user := GetUserFromContext(r.Context())

	q := "DELETE FROM tasks WHERE id = ? AND user_id = ?"
	if _, err := h.db.ExecContext(r.Context(), q, id, user.ID); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
