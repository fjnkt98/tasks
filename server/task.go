package server

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/fjnkt98/tasks/ent"
	"github.com/fjnkt98/tasks/ent/task"
)

// ---------- List Tasks ----------
type ListTasksHandler struct {
	client       *ent.Client
	templateHTTP *template.Template
	templateHTMX *template.Template
}

func NewListTasksHandler(client *ent.Client) *ListTasksHandler {
	return &ListTasksHandler{
		client:       client,
		templateHTTP: template.Must(template.ParseFS(templates, "templates/layout.html", "templates/tasks.html", "templates/partials/tasks.html")),
		templateHTMX: template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
	}
}

type ListTasksParams struct {
	Page   int
	Limit  int
	Status string
}

type TaskData struct {
	LayoutData
	Tasks     []*ent.Task
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
	}

	status := values.Get("status")

	return ListTasksParams{
		Page:   page,
		Limit:  limit,
		Status: status,
	}
}

func (h *ListTasksHandler) GetTasks(ctx context.Context, userID int, params ListTasksParams) ([]*ent.Task, error) {
	q := h.client.Task.Query().
		Where(task.UserID(userID)).
		Limit(params.Limit).
		Offset(params.Limit*(params.Page-1)).
		Order(
			task.ByStatus(entsql.OrderAsc()),
			task.ByID(entsql.OrderDesc()),
		)
	if params.Status != "" {
		status := task.Status(params.Status)
		if err := task.StatusValidator(status); err == nil {
			q = q.Where(task.StatusEQ(status))
		}
	}

	tasks, err := q.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
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
	client *ent.Client
	t      *template.Template
}

func NewGetTaskHandler(client *ent.Client) *GetTaskHandler {
	return &GetTaskHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
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

	t, err := h.client.Task.Query().Where(task.UserID(userID)).Where(task.ID(id)).Only(r.Context())
	if err != nil {
		if ent.IsNotFound(err) {
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
		Tasks:     []*ent.Task{t},
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
	client *ent.Client
	t      *template.Template
}

func NewGetTaskEditHandler(client *ent.Client) *GetTaskEditHandler {
	return &GetTaskEditHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/partials/task_edit.html")),
	}
}

func (h *GetTaskEditHandler) ResponseHTMX(w http.ResponseWriter, t *ent.Task) error {
	var buf bytes.Buffer
	if err := h.t.ExecuteTemplate(&buf, "task_edit", &t); err != nil {
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

	t, err := h.client.Task.Query().Where(task.UserID(userID)).Where(task.ID(id)).Only(r.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	if err := h.ResponseHTMX(w, t); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Post Task ----------
type PostTaskHandler struct {
	client *ent.Client
	t      *template.Template
}

func NewPostTaskHandler(client *ent.Client) *PostTaskHandler {
	return &PostTaskHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
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
	params, err := h.GetParams(r)
	if err != nil {
		Handle400(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	t, err := h.client.Task.Create().SetTitle(params.Title).SetStatus(task.StatusCreated).SetUserID(userID).Save(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []*ent.Task{t},
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
	client *ent.Client
	t      *template.Template
}

func NewPutTaskHandler(client *ent.Client) *PutTaskHandler {
	return &PutTaskHandler{
		client: client,
		t:      template.Must(template.ParseFS(templates, "templates/partials/tasks.html")),
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

func (h *PutTaskHandler) UpdateTask(ctx context.Context, id int, userID int, params UpdateTaskParams) (*ent.Task, error) {
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	t, err := tx.Task.Query().Where(task.UserID(userID)).Where(task.ID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, err
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	update := t.Update()

	if params.Title != "" {
		update = update.SetTitle(params.Title)
	}
	if params.Status != "" {
		status := task.Status(params.Status)
		if err := task.StatusValidator(status); err == nil {
			update = update.SetStatus(status)
		}
	}

	t, err = update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return t, nil
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
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	params := h.GetParams(r)
	t, err := h.UpdateTask(r.Context(), id, userID, params)
	if err != nil {
		if ent.IsNotFound(err) {
			Handle404(w, r)
			return
		}
		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []*ent.Task{t},
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
	client *ent.Client
}

func NewDeleteTaskHandler(client *ent.Client) *DeleteTaskHandler {
	return &DeleteTaskHandler{
		client: client,
	}
}

func (h *DeleteTaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		Handle404(w, r)
		return
	}

	userID := GetUserIDFromContext(r.Context())

	if _, err := h.client.Task.Delete().Where(task.UserID(userID)).Where(task.ID(id)).Exec(r.Context()); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
