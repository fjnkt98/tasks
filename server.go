package main

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

	mux.Handle("GET /static/", http.FileServer(http.FS(statics)))

	m := NewChainedMiddleware(
		NewRecoveryMiddleware(),
		NewCrossOriginProtectionMiddleware(),
		NewByteLimitMiddleware(),
		NewSessionMiddleware(db),
	)

	mux.Handle("GET /", m(NewIndexHandler()))

	mux.Handle("GET /signup", m(NewGetSignupHandler()))
	mux.Handle("POST /signup", m(NewPostSignupHandler(db)))
	mux.Handle("GET /signup/success", m(NewGetSignupSuccessHandler()))
	mux.Handle("GET /signin", m((NewGetSigninHandler())))
	mux.Handle("POST /signin", m((NewPostSigninHandler(db))))

	{
		m := NewChainedMiddleware(
			m,
			NewLoginRequiredMiddleware(),
		)
		mux.Handle("POST /signout", m(NewPostSignoutHandler(db)))

		mux.Handle("GET /tasks", m(NewListTasksHandler(db)))
		mux.Handle("GET /tasks/{id}", m(NewGetTaskHandler(db)))
		mux.Handle("GET /tasks/{id}/edit", m(NewGetTaskEditHandler(db)))
		mux.Handle("POST /tasks", m(NewPostTaskHandler(db)))
		mux.Handle("PUT /tasks/{id}", m(NewPutTaskHandler(db)))
		mux.Handle("DELETE /tasks/{id}", m(NewDeleteTaskHandler(db)))
	}

	return otelhttp.NewHandler(mux, "http-request")
}
