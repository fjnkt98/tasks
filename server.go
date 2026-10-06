package main

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:embed templates
var templates embed.FS

//go:embed static
var statics embed.FS

func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

type LayoutData struct {
	Authorized bool
}

func NewServer(port int, db *sql.DB) (*http.Server, error) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      NewHandler(db),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return server, nil
}

func NewHandler(db *sql.DB) http.Handler {
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

var template400 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/400.html"))

func Handle400(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template400.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusBadRequest)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

var template404 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/404.html"))

func Handle404(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template404.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusNotFound)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}

var template500 = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/500.html"))

func Handle500(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	data := LayoutData{
		Authorized: IsAuthorized(r.Context()),
	}

	var buf bytes.Buffer
	if err := template500.Execute(&buf, &data); err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusInternalServerError)
	if _, err := buf.WriteTo(w); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
