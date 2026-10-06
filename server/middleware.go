package server

import (
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/fjnkt98/tasks/ent"
	"github.com/fjnkt98/tasks/ent/session"
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
					slog.ErrorContext(r.Context(), "panic recovered", slog.Any("error", err))
					Handle500(w, r)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func NewSessionMiddleware(client *ent.Client) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AuthCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			s, err := client.Session.Query().Where(session.Token(cookie.Value)).Only(r.Context())
			if err != nil {
				if ent.IsNotFound(err) {
					http.SetCookie(w, NewAuthCookie("", -1))

					next.ServeHTTP(w, r)
					return
				}

				slog.ErrorContext(r.Context(), "get session by token", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			if s.ExpiresAt.Before(time.Now()) {
				http.SetCookie(w, NewAuthCookie("", -1))

				next.ServeHTTP(w, r)
				return
			}

			user, err := client.User.Get(r.Context(), s.UserID)
			if err != nil {
				slog.ErrorContext(r.Context(), "get user", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			http.SetCookie(w, NewAuthCookie(cookie.Value, 86400))

			ctx := SetUserIDIntoContext(r.Context(), user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func NewLoginRequiredMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if IsAuthorized(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
		})
	}
}

func NewByteLimitMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.MaxBytesHandler(next, 1<<20)
	}
}

func NewCrossOriginProtectionMiddleware() Middleware {
	protection := http.NewCrossOriginProtection()
	return protection.Handler
}
