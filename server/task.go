package server

import (
	"database/sql"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fjnkt98/tasks/repository"
)

type TasksHandler struct {
	q *repository.Queries
}

func NewTasksHandler(db *sql.DB) *TasksHandler {
	return &TasksHandler{
		q: repository.New(db),
	}
}

type TaskData struct {
		Tasks []repository.Task
		LastIndex int
		NextPage int64
		Limit int64
		Status string
}

func (h *TasksHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()

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

	var tasks []repository.Task
	if status == "" || status == "all" {
		tasks, err = h.q.ListTasks(r.Context(), repository.ListTasksParams{
			Offset: limit * (page - 1),
			Limit: limit,
		})
	} else {
		tasks, err = h.q.ListTasksByStatus(r.Context(), repository.ListTasksByStatusParams{
			Status: status,
			Offset: limit * (page - 1),
			Limit: limit,
		})
	}

	if err != nil {
		if r.Header.Get("HX-Request") == "true" {
			http.Error(w, "server error", http.StatusInternalServerError)
		} else {
			Handle500(w, r)
		}
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		return
	}

	data := TaskData{
		Tasks: tasks,
		LastIndex: len(tasks) - 1,
		NextPage: page + 1,
		Limit: limit,
		Status: status,
	}

	var t *template.Template
	if r.Header.Get("HX-Request") == "true" {
		t, err = template.ParseFS(templates, "templates/partials/tasks.html")
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
			return
		}
	} else {
		t, err = template.ParseFS(templates, "templates/layout.html", "templates/tasks.html", "templates/partials/tasks.html")
		if err != nil {
			Handle500(w, r)
			slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := t.Execute(w, &data); err != nil {
			Handle500(w, r)
			slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
			return
		}
	}
}

func (h *TasksHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		return
	}

	data := TaskData{
		Tasks: []repository.Task{task},
		LastIndex: -1,
	}

	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) GetTaskEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		return
	}

	t, err := template.ParseFS(templates, "templates/partials/task_edit.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := t.ExecuteTemplate(w, "task_edit", &task); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) PostTask(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")

	task, err := h.q.CreateTask(r.Context(), title)
	if  err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		return
	}

	data := TaskData{
		Tasks: []repository.Task{task},
		LastIndex: -1,
	}

	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) PutTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		return
	}

	params := repository.UpdateTaskParams{ID: id}
	if title := r.FormValue("title"); title != "" {
		params.Title = title
	} else {
		params.Title = task.Title
	}
	if status := r.FormValue("status"); status != "" {
		params.Status = status
	} else {
		params.Status = task.Status
	}

	task, err = h.q.UpdateTask(r.Context(), params)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		return
	}

	data := TaskData{
		Tasks: []repository.Task{task},
		LastIndex: -1,
	}

	t, err := template.ParseFS(templates, "templates/partials/tasks.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := t.ExecuteTemplate(w, "tasks", &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.q.DeleteTask(r.Context(), id); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
