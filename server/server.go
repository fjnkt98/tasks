// Package server produces http server
package server

import (
	"database/sql"
	"embed"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:embed templates
var templates embed.FS

//go:embed static
var statics embed.FS

func NewServer(port int, db *sql.DB) (*http.Server, error) {
	mux := http.NewServeMux()

	tasksHandler := NewTasksHandler(db)

	mux.Handle("GET /", &IndexHandler{})
	mux.HandleFunc("GET /tasks/", tasksHandler.GetTasks)
	mux.HandleFunc("POST /tasks/", tasksHandler.PostNewTask)
	mux.HandleFunc("GET /tasks/new", tasksHandler.GetNewTask)
	mux.HandleFunc("GET /tasks/{id}/edit", tasksHandler.GetTaskEdit)
	mux.HandleFunc("POST /tasks/{id}/edit", tasksHandler.PostTaskEdit)
	mux.HandleFunc("GET /api/tasks", tasksHandler.GetTaskParts)
	mux.HandleFunc("PUT /api/tasks/{id}/status", tasksHandler.PutTaskStatus)
	mux.HandleFunc("DELETE /api/tasks/{id}", tasksHandler.DeleteTask)
	mux.Handle("GET /static/", http.FileServer(http.FS(statics)))

	handler := RecoveryMiddleware(mux)
	handler = otelhttp.NewHandler(handler, "http-request")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return server, nil
}
