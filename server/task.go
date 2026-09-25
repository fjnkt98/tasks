package server

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"text/template"

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
	t, err := template.ParseFS(templates, "templates/tasks.html", "templates/base.html")
	if err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	type Data struct {
		Tasks []repository.Task
	}

	tasks, err := h.q.ListTasks(r.Context(), repository.ListTasksParams{
		Offset: 0,
		Limit: 100,
	})
	if err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		return
	}

	data := Data{
		Tasks: tasks,
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		Handle500(w, r)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
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

	t, err := template.ParseFS(templates, "templates/task_edit.html", "templates/base.html")
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
