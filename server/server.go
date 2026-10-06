// Package server produces http server
package server

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/fjnkt98/tasks/ent"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:embed templates
var templates embed.FS

//go:embed static
var statics embed.FS

func NewServer(port int, client *ent.Client) (*http.Server, error) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      newHandler(client),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return server, nil
}

func newHandler(client *ent.Client) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServer(http.FS(statics)))

	{
		m := NewChainedMiddleware(
			NewCrossOriginProtectionMiddleware(),
			NewByteLimitMiddleware(),
			NewSessionMiddleware(client),
			NewRecoveryMiddleware(),
		)
		mux.Handle("GET /", m(NewIndexHandler()))
	}
	{
		m := NewChainedMiddleware(
			NewCrossOriginProtectionMiddleware(),
			NewByteLimitMiddleware(),
			NewSessionMiddleware(client),
			NewRecoveryMiddleware(),
		)
		mux.Handle("GET /signup", m(NewGetSignupHandler()))
		mux.Handle("POST /signup", m(NewPostSignupHandler(client)))
		mux.Handle("GET /signup/success", m(NewGetSignupSuccessHandler()))
		mux.Handle("GET /signin", m((NewGetSigninHandler())))
		mux.Handle("POST /signin", m((NewPostSigninHandler(client))))
	}
	{
		m := NewChainedMiddleware(
			NewCrossOriginProtectionMiddleware(),
			NewByteLimitMiddleware(),
			NewSessionMiddleware(client),
			NewLoginRequiredMiddleware(),
			NewRecoveryMiddleware(),
		)
		mux.Handle("POST /signout", m(NewPostSignoutHandler(client)))

		mux.Handle("GET /tasks", m(NewListTasksHandler(client)))
		mux.Handle("GET /tasks/{id}", m(NewGetTaskHandler(client)))
		mux.Handle("GET /tasks/{id}/edit", m(NewGetTaskEditHandler(client)))
		mux.Handle("POST /tasks", m(NewPostTaskHandler(client)))
		mux.Handle("PUT /tasks/{id}", m(NewPutTaskHandler(client)))
		mux.Handle("DELETE /tasks/{id}", m(NewDeleteTaskHandler(client)))
	}

	return otelhttp.NewHandler(mux, "http-request")
}
