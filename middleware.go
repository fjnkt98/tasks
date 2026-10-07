package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"
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
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AuthCookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			q := "SELECT user_id, expires_at FROM sessions WHERE token = ? LIMIT 1"
			row := db.QueryRowContext(r.Context(), q, cookie.Value)
			var userID int
			var expiresAt int64
			if err := row.Scan(&userID, &expiresAt); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.SetCookie(w, NewAuthCookie("", -1))

					next.ServeHTTP(w, r)
					return
				}

				slog.ErrorContext(r.Context(), "get session by token", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			if expiresAt <= time.Now().Unix() {
				http.SetCookie(w, NewAuthCookie("", -1))

				next.ServeHTTP(w, r)
				return
			}

			q = "SELECT id FROM users WHERE id = ? LIMIT 1"
			row = db.QueryRowContext(r.Context(), q, userID)
			if err := row.Scan(&userID); err != nil {
				slog.ErrorContext(r.Context(), "get user", slog.Any("error", err))
				Handle500(w, r)
				return
			}

			if _, err := db.ExecContext(r.Context(), "UPDATE sessions SET expires_at = ? WHERE token = ?", time.Now().Add(24*time.Hour).Unix(), cookie.Value); err != nil {
				slog.ErrorContext(r.Context(), "extend session", slog.Any("error", err))
				Handle500(w, r)
				return
			}
			http.SetCookie(w, NewAuthCookie(cookie.Value, 86400))

			ctx := SetUserIDIntoContext(r.Context(), userID)
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
