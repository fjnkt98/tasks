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

func (h *TasksHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
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

	type Data struct {
		Tasks []repository.Task
		LastIndex int
		NextPage int64
		Limit int64
		Status string
	}

	data := Data{
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

func (h *TasksHandler) GetNewTask(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/layout.html", "templates/task_new.html")
	if err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, nil); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) PostNewTask(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	description := r.FormValue("description")

	if _, err := h.q.CreateTask(r.Context(), repository.CreateTaskParams{
		Title: title,
		Description: description,
	}); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "create task", slog.Any("error", err))
		return
	}

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *TasksHandler) GetTaskEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle400(w, r)
		return
	}

	task, err := h.q.GetTaskByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			Handle404(w, r)
			return
		}

		Handle500(w, r)
		slog.ErrorContext(r.Context(), "get task by id", slog.Any("error", err))
		return
	}

	t, err := template.ParseFS(templates, "templates/layout.html", "templates/task_edit.html")
	if err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	type Data struct {
		ID int64
		Title string
		Description string
		Status string
	}

	data := Data{
		ID: task.ID,
		Title: task.Title,
		Description: task.Description,
		Status: task.Status,
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) PostTaskEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle400(w, r)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	status := "created"
	if r.FormValue("status") == "on" {
		status = "done"
	}

	if err := h.q.UpdateTask(r.Context(), repository.UpdateTaskParams{
		Title: title,
		Description: description,
		Status: status,
		ID: id,
	}); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "update task", slog.Any("error", err))
		return
	}

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *TasksHandler) PutTaskStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle400(w, r)
		return
	}
	status := r.FormValue("status")

	if err := h.q.UpdateTaskStatus(r.Context(), repository.UpdateTaskStatusParams{
		Status: status,
		ID: id,
	}); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "update task status", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TasksHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		Handle400(w, r)
		return
	}

	if err := h.q.DeleteTask(r.Context(), id); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "delete task", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
