package presentation

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/fjnkt98/tasks/internal/usecases"
)

type Middleware func(http.Handler) http.Handler

func NewChainedMiddleware(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for _, m := range slices.Backward(middlewares) {
			h = m(h)
		}
		return h
	}
}

func NewLoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			slog.InfoContext(
				r.Context(), "ok",
				slog.String("addr", r.RemoteAddr),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("referer", r.Referer()),
				slog.String("host", r.Host),
				slog.Int64("delta", time.Since(start).Milliseconds()),
			)
		})
	}
}

func NewRecoveryMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if err == http.ErrAbortHandler {
						panic(err)
					}
					slog.ErrorContext(r.Context(), "panic recovered", slog.Any("error", err))
					Handle500(w, r)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func NewSessionMiddleware(db *sql.DB) Middleware {
	sessions := usecases.NewSessionUsecase(db)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AuthCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			session, err := sessions.Continue(r.Context(), cookie.Value)
			if err != nil {
				slog.ErrorContext(r.Context(), "continue session", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			if session == nil {
				http.SetCookie(w, NewAuthCookie("", -1))
				next.ServeHTTP(w, r)
				return
			}

			http.SetCookie(w, NewAuthCookie(cookie.Value, 86400))

			ctx := SetUserIntoContext(r.Context(), User{
				ID:    session.UserID,
				Name:  session.UserName,
				Roles: session.Roles,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewSigninRequiredMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsAuthorized(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			http.Redirect(w, r, "/account/signin", http.StatusSeeOther)
		})
	}
}

func NewAdminRequiredMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsAdmin(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
		})
	}
}

func NewByteLimitMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.MaxBytesHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
					if err := r.ParseForm(); err != nil {
						var ErrMaxBytesExceeded *http.MaxBytesError
						if errors.As(err, &ErrMaxBytesExceeded) {
							http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
							return
						}
						http.Error(w, "bad request", http.StatusBadRequest)
						return
					}
				}
				next.ServeHTTP(w, r)
			}),
			1<<20,
		)
	}
}

func NewCrossOriginProtectionMiddleware() Middleware {
	protection := http.NewCrossOriginProtection()
	return protection.Handler
}
