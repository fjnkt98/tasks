package presentation

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

//go:embed static
var statics embed.FS

//go:embed templates
var templates embed.FS

var parsedTemplates = template.Must(template.ParseFS(
	templates,
	"templates/*.html",
	"templates/partials/*.html",
))

func RenderTemplate(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := parsedTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

type LayoutData struct {
	Authorized  bool
	AccountName string
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

	index := NewIndexHandler()
	signup := NewSignupHandler(db)
	signin := NewSigninHandler(db)
	signout := NewSignoutHandler(db)
	changePassword := NewChangePasswordHandler(db)
	deleteAccount := NewDeleteAccountHandler(db)
	tasks := NewTaskHandler(db)
	account := NewAccountHandler(db)
	users := NewUserHandler(db)

	mux.Handle("GET /", m(index))
	mux.Handle("GET /account/signup", m(http.HandlerFunc(signup.Get)))
	mux.Handle("POST /account/signup", m(http.HandlerFunc(signup.Post)))
	mux.Handle("GET /account/signup/success", m(http.HandlerFunc(signup.Success)))
	mux.Handle("GET /account/signin", m(http.HandlerFunc(signin.Get)))
	mux.Handle("POST /account/signin", m(http.HandlerFunc(signin.Post)))
	mux.Handle("GET /account/password_change/success", m(http.HandlerFunc(changePassword.Success)))
	mux.Handle("GET /account/delete/success", m(http.HandlerFunc(deleteAccount.Success)))

	{
		m := NewChainedMiddleware(
			m,
			NewSigninRequiredMiddleware(),
		)
		mux.Handle("POST /account/signout", m(http.HandlerFunc(signout.Post)))

		mux.Handle("GET /tasks", m(http.HandlerFunc(tasks.Index)))
		mux.Handle("GET /tasks/{id}", m(http.HandlerFunc(tasks.Show)))
		mux.Handle("GET /tasks/{id}/edit", m(http.HandlerFunc(tasks.Edit)))
		mux.Handle("POST /tasks", m(http.HandlerFunc(tasks.Create)))
		mux.Handle("PUT /tasks/{id}/title", m(http.HandlerFunc(tasks.UpdateTitle)))
		mux.Handle("PUT /tasks/{id}/status", m(http.HandlerFunc(tasks.UpdateStatus)))
		mux.Handle("DELETE /tasks/{id}", m(http.HandlerFunc(tasks.Destroy)))

		mux.Handle("GET /account/me", m(http.HandlerFunc(account.Get)))
		mux.Handle("GET /account/password_change", m(http.HandlerFunc(changePassword.Get)))
		mux.Handle("POST /account/password_change", m(http.HandlerFunc(changePassword.Post)))
		mux.Handle("GET /account/delete", m(http.HandlerFunc(deleteAccount.Get)))
		mux.Handle("POST /account/delete", m(http.HandlerFunc(deleteAccount.Post)))
	}

	{
		m := NewChainedMiddleware(
			m,
			NewSigninRequiredMiddleware(),
			NewAdminRequiredMiddleware(),
		)

		mux.Handle("GET /users", m(http.HandlerFunc(users.Index)))
		mux.Handle("GET /users/{id}", m(http.HandlerFunc(users.Show)))
	}

	return otelhttp.NewHandler(mux, "http-request")
}

type IndexHandler struct{}

func NewIndexHandler() *IndexHandler {
	return &IndexHandler{}
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := LayoutData{
		Authorized:  IsAuthorized(r.Context()),
		AccountName: GetUserFromContext(r.Context()).Name,
	}

	body, err := RenderTemplate("page/index", &data)
	if err != nil {
		slog.ErrorContext(r.Context(), "render template", slog.Any("error", err))
		Handle500(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		slog.ErrorContext(r.Context(), "write response", slog.Any("error", err))
		return
	}
}
