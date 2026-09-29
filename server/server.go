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

	mux.Handle("GET /", &IndexHandler{})
	mux.Handle("GET /tasks/", NewListTaskService(db))
	mux.Handle("GET /tasks/{id}", NewGetTaskService(db))
	mux.Handle("GET /tasks/{id}/edit", NewGetTaskEditService(db))
	mux.Handle("POST /tasks/", NewPostTaskService(db))
	mux.Handle("PUT /tasks/{id}", NewPutTaskService(db))
	mux.Handle("DELETE /tasks/{id}", NewDeleteTaskService(db))
	mux.Handle("GET /static/", http.FileServer(http.FS(statics)))

	handler := ChainMiddleware(
		mux,
		RecoveryMiddleware,
		CORSMiddleware,
	)
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
