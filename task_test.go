package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTasksHandler(t *testing.T) {
	db := NewTestDB(t)

	_, err := db.ExecContext(t.Context(), `INSERT INTO users (name, digest) VALUES
		('user1', ''),
		('user2', '')`,
	)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `INSERT INTO tasks (user_id, title, status) VALUES
		(1, 'test1', 'created'),
		(1, 'test2', 'done'),
		(1, 'test3', 'done'),
		(1, 'test4', 'created'),
		(2, 'test5', 'created')`,
	)
	require.NoError(t, err)

	for _, test := range []struct {
		Name   string
		UserID int
		Query  string
		Wants  []string
	}{
		{Name: "no params", UserID: 1, Wants: []string{`id="task-4"`, `id="task-3"`, `id="task-2"`, `id="task-1"`}},
		{Name: "page", UserID: 1, Query: "?page=2"},
		{Name: "limit", UserID: 1, Query: "?limit=1", Wants: []string{`id="task-4"`}},
		{Name: "page and limit", UserID: 1, Query: "?page=2&limit=1", Wants: []string{`id="task-1"`}},
		{Name: "invalid page and limit", UserID: 1, Query: "?page=foo&limit=bar", Wants: []string{`id="task-4"`, `id="task-3"`, `id="task-2"`, `id="task-1"`}},
		{Name: "too large page and limit", UserID: 1, Query: "?page=123456789&limit=123456789"},
		{Name: "negative page and limit", UserID: 1, Query: "?page=-1&limit=-1", Wants: []string{`id="task-4"`, `id="task-3"`, `id="task-2"`, `id="task-1"`}},
		{Name: "created", UserID: 1, Query: "?status=created", Wants: []string{`id="task-4"`, `id="task-1"`}},
		{Name: "done", UserID: 1, Query: "?status=done", Wants: []string{`id="task-3"`, `id="task-2"`}},
		{Name: "other user", UserID: 2, Wants: []string{`id="task-5"`}},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), test.UserID)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("/tasks%s", test.Query), nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, "<head>")
			assert.Contains(t, body, "<body")
			assert.Contains(t, body, "<footer")
			for _, want := range test.Wants {
				assert.Contains(t, body, want)
			}
		})
	}

	t.Run("htmx", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewListTasksHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "<head>")
	})

	for _, test := range []struct {
		Name      string
		HXRequest string
	}{
		{Name: "render failed http", HXRequest: ""},
		{Name: "render failed htmx", HXRequest: "true"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			req.Header.Set("HX-Request", test.HXRequest)

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.templateHTTP = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))
			h.templateHTMX = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestGetTaskHandler(t *testing.T) {
	db := NewTestDB(t)

	_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
	require.NoError(t, err)

	t.Run("not found", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2", nil)
		req.SetPathValue("id", "2")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid path value", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo", nil)
		req.SetPathValue("id", "foo")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("render failed", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1", nil)
		req.SetPathValue("id", "1")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1", nil)
		req.SetPathValue("id", "1")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

		body := rec.Body.String()
		assert.NotContains(t, body, "<head>")
		assert.NotContains(t, body, "<body>")
		assert.NotContains(t, body, "<footer>")
	})
}

func TestGetTaskEditHandler(t *testing.T) {
	db := NewTestDB(t)

	_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
	require.NoError(t, err)

	t.Run("not found", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2/edit", nil)
		req.SetPathValue("id", "2")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid path value", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo/edit", nil)
		req.SetPathValue("id", "foo")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("render failed", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1/edit", nil)
		req.SetPathValue("id", "1")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.t = template.Must(template.New("task_edit").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1/edit", nil)
		req.SetPathValue("id", "1")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "<head>")
		assert.NotContains(t, body, "<body>")
		assert.NotContains(t, body, "<footer>")
	})
}

func TestPostTaskHandler(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
	}

	for _, test := range []struct {
		Name string
		Body string
	}{
		{Name: "empty", Body: "title="},
		{Name: "space only", Body: "title=%20%20"},
		{Name: "invalid format", Body: "title=test&foo=%zz"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)
			fixture(t, db)

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(test.Body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM tasks WHERE user_id = 1")
			var count int
			require.NoError(t, row.Scan(&count))

			assert.Equal(t, 0, count)
		})
	}

	t.Run("render failed", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader("title=test"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPostTaskHandler(db)
		h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		values := url.Values{}
		values.Set("title", "test")

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPostTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "<head>")
		assert.NotContains(t, body, "<body")
		assert.NotContains(t, body, "<footer")

		rows, err := db.QueryContext(t.Context(), "SELECT title, status FROM tasks WHERE user_id = 1")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck

		tasks := make([]Task, 0)
		for rows.Next() {
			var task Task
			require.NoError(t, rows.Scan(&task.Title, &task.Status))
			tasks = append(tasks, task)
		}
		require.NoError(t, rows.Err())

		require.Len(t, tasks, 1)
		assert.Equal(t, "test", tasks[0].Title)
		assert.Equal(t, "created", tasks[0].Status)
	})
}

func TestPutTaskHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPutTaskHandler(db)
			params := h.GetParams(req)

			assert.Equal(t, "test", params.Title)
			assert.Equal(t, "done", params.Status)
		})

		t.Run("empty", func(t *testing.T) {
			values := url.Values{}

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1?title=foo&status=bar", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPutTaskHandler(db)
			params := h.GetParams(req)

			assert.Equal(t, "", params.Title)
			assert.Equal(t, "", params.Status)
		})

		t.Run("title contains space", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "  ")
			values.Set("status", "")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPutTaskHandler(db)
			params := h.GetParams(req)

			assert.Equal(t, "", params.Title)
			assert.Equal(t, "", params.Status)
		})
	})

	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)
	}

	t.Run("UpdateTask", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			params := UpdateTaskParams{
				Title:  "new test",
				Status: "done",
			}

			h := NewPutTaskHandler(db)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			require.NoError(t, err)

			assert.Equal(t, "new test", updated.Title)
			assert.Equal(t, "done", updated.Status)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, updated.ID, task.ID)
			assert.Equal(t, updated.Title, task.Title)
			assert.Equal(t, updated.Status, task.Status)
			assert.Equal(t, updated.UserID, task.UserID)
		})

		t.Run("update non-existing task", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			h := NewPutTaskHandler(db)
			_, err := h.UpdateTask(t.Context(), 2, 1, UpdateTaskParams{})
			assert.ErrorIs(t, err, sql.ErrNoRows)
		})

		t.Run("update title only", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			params := UpdateTaskParams{
				Title: "new test",
			}

			h := NewPutTaskHandler(db)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			require.NoError(t, err)

			assert.Equal(t, "new test", updated.Title)
			assert.Equal(t, "created", updated.Status)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, updated.ID, task.ID)
			assert.Equal(t, updated.Title, task.Title)
			assert.Equal(t, updated.Status, task.Status)
			assert.Equal(t, updated.UserID, task.UserID)
		})

		t.Run("update status only", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			params := UpdateTaskParams{
				Status: "done",
			}

			h := NewPutTaskHandler(db)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			require.NoError(t, err)

			assert.Equal(t, "test", updated.Title)
			assert.Equal(t, "done", updated.Status)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, updated.ID, task.ID)
			assert.Equal(t, updated.Title, task.Title)
			assert.Equal(t, updated.Status, task.Status)
			assert.Equal(t, updated.UserID, task.UserID)
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("title", "new test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.NotContains(t, body, "<head>")
			assert.NotContains(t, body, "<body")
			assert.NotContains(t, body, "<footer")

			assert.Contains(t, body, `id="task-1"`)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "new test", task.Title)
			assert.Equal(t, "done", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("update title only", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("title", "new test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.NotContains(t, body, "<head>")
			assert.NotContains(t, body, "<body")
			assert.NotContains(t, body, "<footer")

			assert.Contains(t, body, `id="task-1"`)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "new test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("update status only", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.NotContains(t, body, "<head>")
			assert.NotContains(t, body, "<body")
			assert.NotContains(t, body, "<footer")

			assert.Contains(t, body, `id="task-1"`)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "done", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("update non-existing task", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "2")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)

			assert.Equal(t, "not found\n", rec.Body.String())

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("invalid path value", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "foo")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)

			assert.Equal(t, "not found\n", rec.Body.String())

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("invalid request body", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo", strings.NewReader("title=foo&bar=%zz"))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})

		t.Run("render failed", func(t *testing.T) {
			db := NewTestDB(t)

			fixture(db, t)

			values := url.Values{}
			values.Set("title", "new test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(db)
			h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	})
}

func TestDeleteTaskHandler(t *testing.T) {
	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)
	}

	t.Run("delete non-existing task", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/2", nil)
		req.SetPathValue("id", "2")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
		var task Task
		require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

		assert.Equal(t, "test", task.Title)
		assert.Equal(t, "created", task.Status)
	})

	t.Run("invalid path value", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/foo", nil)
		req.SetPathValue("id", "foo")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
		var task Task
		require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

		assert.Equal(t, "test", task.Title)
		assert.Equal(t, "created", task.Status)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/1", nil)
		req.SetPathValue("id", "1")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM tasks")
		var count int
		require.NoError(t, row.Scan(&count))

		assert.Equal(t, 0, count)
	})
}
