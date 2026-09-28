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
type ListTasksService struct {
	db *sql.DB
}

func NewListTaskService(db *sql.DB) *ListTasksService {
	return &ListTasksService{
		db: db,
	}
}

type ListTasksParams struct {
	Page   int64
	Limit  int64
	Status string
}

type TaskData struct {
	Tasks     []repository.Task
	LastIndex int
	NextPage  int64
	Limit     int64
	Status    string
}

func (s *ListTasksService) GetParams(values url.Values) ListTasksParams {
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

func (s *ListTasksService) GetTasks(ctx context.Context, params ListTasksParams) ([]repository.Task, error) {
	q := repository.New(s.db)

	var tasks []repository.Task
	var err error
	if params.Status == "" {
		tasks, err = q.ListTasks(ctx, repository.ListTasksParams{
			Offset: params.Limit * (params.Page - 1),
			Limit:  params.Limit,
		})
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
	} else {
		tasks, err = q.ListTasksByStatus(ctx, repository.ListTasksByStatusParams{
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

func (s *ListTasksService) ResponseHTTP(w http.ResponseWriter, data TaskData) error {
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

func (s *ListTasksService) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
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

func (s *ListTasksService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := s.GetParams(r.URL.Query())
	tasks, err := s.GetTasks(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     tasks,
		LastIndex: len(tasks) - 1,
		NextPage:  params.Page + 1,
		Limit:     params.Limit,
		Status:    params.Status,
	}

	if IsHTMX(r) {
		if err := s.ResponseHTMX(w, data); err != nil {
			slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
			Handle500(w, r)
			return
		}
	} else {
		if err := s.ResponseHTTP(w, data); err != nil {
			slog.ErrorContext(r.Context(), "response for http", slog.Any("error", err))
			Handle500(w, r)
			return
		}
	}
}

// ---------- Get Task ----------
type GetTaskService struct {
	db *sql.DB
}

func NewGetTaskService(db *sql.DB) *GetTaskService {
	return &GetTaskService{
		db: db,
	}
}

func (s *GetTaskService) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
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

func (s *GetTaskService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	q := repository.New(s.db)
	task, err := q.GetTaskByID(r.Context(), id)
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
		Tasks:     []repository.Task{task},
		LastIndex: -1,
	}

	if err := s.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Get Task Edit ----------
type GetTaskEditService struct {
	q *repository.Queries
}

func NewGetTaskEditService(db *sql.DB) *GetTaskEditService {
	return &GetTaskEditService{
		q: repository.New(db),
	}
}

func (s *GetTaskEditService) ResponseHTMX(w http.ResponseWriter, task repository.Task) error {
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

func (s *GetTaskEditService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	task, err := s.q.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	if err := s.ResponseHTMX(w, task); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Post Task ----------
type PostTaskService struct {
	db *sql.DB
}

func NewPostTaskService(db *sql.DB) *PostTaskService {
	return &PostTaskService{
		db: db,
	}
}

type CreateTaskParams struct {
	Title string
}

func (s *PostTaskService) GetParams(r *http.Request) (CreateTaskParams, error) {
	title := strings.TrimSpace(r.FormValue("title"))
	if utf8.RuneCountInString(title) == 0 {
		return CreateTaskParams{}, fmt.Errorf("title required: %w", ErrBadRequest)
	}

	return CreateTaskParams{
		Title: title,
	}, nil
}

func (s *PostTaskService) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
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

func (s *PostTaskService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, err := s.GetParams(r)
	if err != nil {
		Handle400(w, r)
		return
	}

	q := repository.New(s.db)
	task, err := q.CreateTask(r.Context(), params.Title)
	if err != nil {
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	data := TaskData{
		Tasks:     []repository.Task{task},
		LastIndex: -1,
	}

	if err := s.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Put Task ----------
type PutTaskService struct {
	db *sql.DB
}

func NewPutTaskService(db *sql.DB) *PutTaskService {
	return &PutTaskService{
		db: db,
	}
}

type UpdateTaskParams struct {
	Title  string
	Status string
}

func (s *PutTaskService) GetParams(r *http.Request) UpdateTaskParams {
	return UpdateTaskParams{
		Title:  strings.TrimSpace(r.FormValue("title")),
		Status: r.FormValue("status"),
	}
}

func (s *PutTaskService) UpdateTask(ctx context.Context, id int64, params UpdateTaskParams) (repository.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.Task{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	q := repository.New(s.db).WithTx(tx)
	task, err := q.GetTaskByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Task{}, err
		}
		return repository.Task{}, fmt.Errorf("get task by id: %w", err)
	}

	data := repository.UpdateTaskParams{
		ID: id,
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

func (s *PutTaskService) ResponseHTMX(w http.ResponseWriter, data TaskData) error {
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

func (s *PutTaskService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	params := s.GetParams(r)
	task, err := s.UpdateTask(r.Context(), id, params)
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

	if err := s.ResponseHTMX(w, data); err != nil {
		slog.ErrorContext(r.Context(), "response for htmx", slog.Any("error", err))
		Handle500(w, r)
		return
	}
}

// ---------- Delete Task ----------
type DeleteTaskService struct {
	db *sql.DB
}

func NewDeleteTaskService(db *sql.DB) *DeleteTaskService {
	return &DeleteTaskService{
		db: db,
	}
}

func (s *DeleteTaskService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle404(w, r)
		return
	}

	q := repository.New(s.db)
	if err := q.DeleteTask(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
