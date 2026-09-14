package server

import (
	"database/sql"
	"log/slog"
	"net/http"
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

func (h *TasksHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templates, "templates/tasks.html", "templates/base.html")
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "parse template", slog.Any("error", err))
		return
	}

	type Data struct {
		Title string
		Tasks []repository.Task
	}

	tasks, err := h.q.ListTasks(r.Context())
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "get tasks", slog.Any("error", err))
		return
	}

	data := Data{
		Title: "Tasks",
		Tasks: tasks,
	}

	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, &data); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
