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

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      newHandler(db),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return server, nil
}

func newHandler(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /", &IndexHandler{})
	mux.Handle("GET /static/", http.FileServer(http.FS(statics)))

	{
		m := NewChainedMiddleware(
			CORSMiddleware,
		)
		mux.Handle("GET /tasks/", m(NewListTasksHandler(db)))
		mux.Handle("GET /tasks/{id}", m(NewGetTaskHandler(db)))
		mux.Handle("GET /tasks/{id}/edit", m(NewGetTaskEditHandler(db)))
		mux.Handle("POST /tasks/", m(NewPostTaskHandler(db)))
		mux.Handle("PUT /tasks/{id}", m(NewPutTaskHandler(db)))
		mux.Handle("DELETE /tasks/{id}", m(NewDeleteTaskHandler(db)))
	}

	return otelhttp.NewHandler(RecoveryMiddleware(mux), "http-request")
}
