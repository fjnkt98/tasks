package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.Run()
}

func TestTraceHandler(t *testing.T) {
	var buf bytes.Buffer
	h := &traceHandler{
		Handler: slog.NewJSONHandler(&buf, nil),
	}

	results := func() []map[string]any {
		var ms []map[string]any
		for line := range bytes.SplitSeq(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal(line, &m))
			ms = append(ms, m)
		}
		return ms
	}

	assert.NoError(t, slogtest.TestHandler(h, results))
}

func TestTraceHandlerWithTrace(t *testing.T) {
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(t.Context(), spanContext)

	for _, test := range []struct {
		Name   string
		Logger func(*slog.Logger) *slog.Logger
		Want   map[string]any
	}{
		{
			Name:   "plain",
			Logger: func(l *slog.Logger) *slog.Logger { return l },
			Want:   map[string]any{"action": "query"},
		},
		{
			Name:   "attributes",
			Logger: func(l *slog.Logger) *slog.Logger { return l.With("component", "db") },
			Want:   map[string]any{"component": "db", "action": "query"},
		},
		{
			Name:   "group",
			Logger: func(l *slog.Logger) *slog.Logger { return l.WithGroup("request") },
			Want:   map[string]any{"request": map[string]any{"action": "query"}},
		},
		{
			Name: "nested groups and attributes",
			Logger: func(l *slog.Logger) *slog.Logger {
				return l.With("component", "db").
					WithGroup("request").With("user_id", 1).
					WithGroup("sql").With("table", "tasks")
			},
			Want: map[string]any{
				"component": "db",
				"request": map[string]any{
					"user_id": float64(1),
					"sql":     map[string]any{"table": "tasks", "action": "query"},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := test.Logger(slog.New(&traceHandler{
				Handler: slog.NewJSONHandler(&buf, nil),
			}))
			logger.InfoContext(ctx, "test", slog.String("action", "query"))

			var got map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
			assert.Contains(t, got, "time")
			delete(got, "time")

			test.Want["level"] = "INFO"
			test.Want["msg"] = "test"
			test.Want["logging.googleapis.com/trace"] = fmt.Sprintf(
				"projects/%s/traces/%s", os.Getenv("GOOGLE_CLOUD_PROJECT_NAME"), spanContext.TraceID(),
			)
			test.Want["logging.googleapis.com/spanId"] = spanContext.SpanID().String()
			test.Want["logging.googleapis.com/trace_sampled"] = true
			assert.Equal(t, test.Want, got)
		})
	}
}

func TestTraceHandlerDerivedLoggers(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(&traceHandler{Handler: slog.NewJSONHandler(&buf, nil)})
	group := base.WithGroup("request").With("component", "db")
	first := group.With("branch", "first")
	second := group.With("branch", "second")

	for _, test := range []struct {
		Logger *slog.Logger
		Want   map[string]any
	}{
		{Logger: first, Want: map[string]any{"request": map[string]any{"component": "db", "branch": "first"}}},
		{Logger: second, Want: map[string]any{"request": map[string]any{"component": "db", "branch": "second"}}},
		{Logger: group, Want: map[string]any{"request": map[string]any{"component": "db"}}},
		{Logger: base, Want: map[string]any{}},
	} {
		buf.Reset()
		test.Logger.InfoContext(t.Context(), "test")

		var got map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		delete(got, "time")
		test.Want["level"] = "INFO"
		test.Want["msg"] = "test"
		assert.Equal(t, test.Want, got)
	}
}

func TestTraceHandlerContextPerRecord(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(&traceHandler{Handler: slog.NewJSONHandler(&buf, nil)}).WithGroup("request").With("component", "db")

	for _, spanContext := range []trace.SpanContext{
		trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled}),
		trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{3}, SpanID: trace.SpanID{4}}),
		{},
	} {
		buf.Reset()
		ctx := trace.ContextWithSpanContext(t.Context(), spanContext)
		logger.InfoContext(ctx, "test")

		var got map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
		if spanContext.IsValid() {
			assert.Equal(t, fmt.Sprintf("projects/%s/traces/%s", os.Getenv("GOOGLE_CLOUD_PROJECT_NAME"), spanContext.TraceID()), got["logging.googleapis.com/trace"])
			assert.Equal(t, spanContext.SpanID().String(), got["logging.googleapis.com/spanId"])
			assert.Equal(t, spanContext.IsSampled(), got["logging.googleapis.com/trace_sampled"])
		} else {
			assert.NotContains(t, got, "logging.googleapis.com/trace")
			assert.NotContains(t, got, "logging.googleapis.com/spanId")
			assert.NotContains(t, got, "logging.googleapis.com/trace_sampled")
		}
		assert.Equal(t, map[string]any{"component": "db"}, got["request"])
	}
}

func TestSetup(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	file := NewTestDBFile(t)

	t.Setenv("DATABASE_URL", fmt.Sprintf("file:%s", file))
	t.Setenv("PORT", "8000")
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "admin")

	_, shutdown, err := setup(t.Context())
	require.NoError(t, err)
	defer shutdown() // nolint:errcheck
}

func TestServe(t *testing.T) {
	for _, test := range []struct {
		Name    string
		Force   bool
		Timeout time.Duration
	}{
		{Name: "graceful shutdown", Force: false, Timeout: 1 * time.Second},
		{Name: "force shutdown", Force: true, Timeout: 50 * time.Millisecond},
	} {
		t.Run(test.Name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			disconnected := make(chan struct{})

			unblock := sync.OnceFunc(func() { close(release) })

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			require.NoError(t, listener.Close())

			server := &http.Server{
				Addr: listener.Addr().String(),
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(started)

					select {
					case <-release:
						_, _ = io.WriteString(w, "ok")
					case <-r.Context().Done():
						close(disconnected)
					}
				}),
			}

			if !test.Force {
				server.RegisterOnShutdown(unblock)
			}

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(func() {
				cancel()
				unblock()
				assert.NoError(t, server.Close())
			})

			serveDone := make(chan error, 1)
			go func() {
				serveDone <- serve(ctx, server, test.Timeout)
			}()

			require.Eventually(t, func() bool {
				conn, err := net.DialTimeout("tcp", server.Addr, 100*time.Millisecond)
				if err != nil {
					return false
				}
				assert.NoError(t, conn.Close())
				return true
			}, 3*time.Second, 10*time.Millisecond)

			type result struct {
				status int
				body   string
				err    error
			}

			requestDone := make(chan result, 1)
			client := &http.Client{}
			t.Cleanup(client.CloseIdleConnections)
			go func() {
				res, err := client.Get("http://" + server.Addr)
				if err != nil {
					requestDone <- result{err: err}
					return
				}
				defer res.Body.Close() // nolint:errcheck

				body, err := io.ReadAll(res.Body)
				requestDone <- result{
					status: res.StatusCode,
					body:   string(body),
					err:    err,
				}
			}()

			<-started
			cancel()

			err = <-serveDone

			if test.Force {
				require.ErrorIs(t, err, context.DeadlineExceeded)

				<-disconnected
				got := <-requestDone
				require.Error(t, got.err)
			} else {
				require.NoError(t, err)

				got := <-requestDone
				require.NoError(t, got.err)
				assert.Equal(t, http.StatusOK, got.status)
				assert.Equal(t, "ok", got.body)
			}
		})
	}
}

func TestSeed(t *testing.T) {
	type User struct {
		name   string
		digest string
	}

	type Relation struct {
		userID int
		role   string
	}

	t.Run("expired sessions will be deleted", func(t *testing.T) {
		t.Setenv("ADMIN_USERNAME", "admin")
		t.Setenv("ADMIN_PASSWORD", "admin")

		db := NewTestDB(t)
		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', 'test')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(1, 'token1', 0),
			(1, 'token2', 32503680000)`,
		)
		require.NoError(t, err)

		err = seed(t.Context(), db)
		require.NoError(t, err)

		tokens := make([]string, 0)
		rows, err := db.QueryContext(t.Context(), "SELECT token FROM sessions")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var token string
			err = rows.Scan(&token)
			require.NoError(t, err)

			tokens = append(tokens, token)
		}
		require.NoError(t, rows.Err())

		assert.Equal(t, []string{"token2"}, tokens)
	})

	t.Run("admin user created", func(t *testing.T) {
		t.Setenv("ADMIN_USERNAME", "admin")
		t.Setenv("ADMIN_PASSWORD", "admin")

		db := NewTestDB(t)
		err := seed(t.Context(), db)
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT name, digest FROM users")
		var user User
		require.NoError(t, row.Scan(&user.name, &user.digest))
		assert.Equal(t, "admin", user.name)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.digest), []byte("admin")))

		row = db.QueryRowContext(t.Context(), "SELECT name FROM roles")
		var name string
		require.NoError(t, row.Scan(&name))
		assert.Equal(t, "admin", name)

		row = db.QueryRowContext(t.Context(), "SELECT user_id, role FROM user_role_relations")
		var relation Relation
		require.NoError(t, row.Scan(&relation.userID, &relation.role))
		assert.Equal(t, 1, relation.userID)
		assert.Equal(t, "admin", relation.role)
	})

	t.Run("empty admin username or password will be rejected", func(t *testing.T) {
		for _, test := range []struct {
			Name     string
			Username string
			Password string
		}{
			{Name: "empty username", Username: "", Password: "admin"},
			{Name: "empty password", Username: "admin", Password: ""},
		} {
			t.Run(test.Name, func(t *testing.T) {
				t.Setenv("ADMIN_USERNAME", test.Username)
				t.Setenv("ADMIN_PASSWORD", test.Password)

				db := NewTestDB(t)
				err := seed(t.Context(), db)
				require.Error(t, err)
			})
		}
	})

	t.Run("existing admin's password will be updated", func(t *testing.T) {
		db := NewTestDB(t)

		t.Setenv("ADMIN_USERNAME", "admin")
		t.Setenv("ADMIN_PASSWORD", "admin")
		err := seed(t.Context(), db)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), "INSERT INTO sessions (user_id, token, expires_at) VALUES (1, 'token1', 32503680000)")
		require.NoError(t, err)

		t.Setenv("ADMIN_USERNAME", "admin")
		t.Setenv("ADMIN_PASSWORD", "new-admin-password")
		err = seed(t.Context(), db)
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT name, digest FROM users")
		var user User
		require.NoError(t, row.Scan(&user.name, &user.digest))

		assert.Equal(t, "admin", user.name)
		assert.Error(t, bcrypt.CompareHashAndPassword([]byte(user.digest), []byte("admin")))
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.digest), []byte("new-admin-password")))

		row = db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sessions WHERE user_id = 1")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("old admin user and role will be deleted", func(t *testing.T) {
		db := NewTestDB(t)

		t.Setenv("ADMIN_USERNAME", "admin")
		t.Setenv("ADMIN_PASSWORD", "admin")
		err := seed(t.Context(), db)
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT name, digest FROM users")
		var user User
		require.NoError(t, row.Scan(&user.name, &user.digest))
		assert.Equal(t, "admin", user.name)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.digest), []byte("admin")))

		row = db.QueryRowContext(t.Context(), "SELECT user_id, role FROM user_role_relations")
		var relation Relation
		require.NoError(t, row.Scan(&relation.userID, &relation.role))
		assert.Equal(t, 1, relation.userID)
		assert.Equal(t, "admin", relation.role)

		t.Setenv("ADMIN_USERNAME", "new-admin")
		t.Setenv("ADMIN_PASSWORD", "new-admin")
		err = seed(t.Context(), db)
		require.NoError(t, err)

		users := make([]User, 0)
		rows, err := db.QueryContext(t.Context(), "SELECT name, digest FROM users")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var user User
			require.NoError(t, rows.Scan(&user.name, &user.digest))
			users = append(users, user)
		}
		require.NoError(t, rows.Err())

		require.Len(t, users, 1)
		assert.Equal(t, "new-admin", users[0].name)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(users[0].digest), []byte("new-admin")))

		relations := make([]Relation, 0)
		rows, err = db.QueryContext(t.Context(), "SELECT user_id, role FROM user_role_relations")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var relation Relation
			require.NoError(t, rows.Scan(&relation.userID, &relation.role))
			relations = append(relations, relation)
		}
		require.NoError(t, rows.Err())

		require.Len(t, relations, 1)
		assert.Equal(t, 2, relations[0].userID)
		assert.Equal(t, "admin", relations[0].role)
	})
}
