package server

import (
	"database/sql"
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

func (h *TasksHandler) Index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/tasks.html", "templates/base.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	type Data struct {
		Title string
		Statuses []string
		Tasks []repository.Task
	}

	statuses, err := h.q.ListStatuses(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get statuses", slog.Any("error", err))
		return
	}

	tasks, err := h.q.ListTasks(r.Context(), repository.ListTasksParams{
		Offset: 0,
		Limit: 100,
	})
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		return
	}

	data := Data{
		Title: "Tasks",
		Tasks: tasks,
		Statuses: statuses,
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

func (h *TasksHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")

	if err := h.q.UpdateTaskStatus(r.Context(), repository.UpdateTaskStatusParams{
		Status: status,
		ID: id,
	}); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "update task status", slog.Any("error", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
