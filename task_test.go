package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
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
		Wants  []int
	}{
		{Name: "no params", UserID: 1, Wants: []int{4, 3, 2, 1}},
		{Name: "page", UserID: 1, Query: "?page=2"},
		{Name: "limit", UserID: 1, Query: "?limit=1", Wants: []int{4}},
		{Name: "page and limit", UserID: 1, Query: "?page=2&limit=1", Wants: []int{1}},
		{Name: "invalid page and limit", UserID: 1, Query: "?page=foo&limit=bar", Wants: []int{4, 3, 2, 1}},
		{Name: "too large page and limit", UserID: 1, Query: "?page=123456789&limit=123456789"},
		{Name: "negative page and limit", UserID: 1, Query: "?page=-1&limit=-1", Wants: []int{4}},
		{Name: "created", UserID: 1, Query: "?status=created", Wants: []int{4, 1}},
		{Name: "done", UserID: 1, Query: "?status=done", Wants: []int{3, 2}},
		{Name: "other user", UserID: 2, Wants: []int{5}},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ctx := SetUserIntoContext(t.Context(), User{ID: test.UserID})
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
			for i := range 5 {
				want := fmt.Sprintf(`id="task-%d"`, i+1)
				if slices.Contains(test.Wants, i+1) {
					assert.Contains(t, body, want)
				} else {
					assert.NotContains(t, body, want)
				}
			}
		})

		t.Run(fmt.Sprintf("%s htmx", test.Name), func(t *testing.T) {
			ctx := SetUserIntoContext(t.Context(), User{ID: test.UserID})
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("/tasks%s", test.Query), nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.NotContains(t, rec.Body.String(), "<head>")
			for i := range 5 {
				want := fmt.Sprintf(`id="task-%d"`, i+1)
				if slices.Contains(test.Wants, i+1) {
					assert.Contains(t, body, want)
				} else {
					assert.NotContains(t, body, want)
				}
			}
		})
	}

	for _, test := range []struct {
		Name      string
		HXRequest string
	}{
		{Name: "render failed http", HXRequest: ""},
		{Name: "render failed htmx", HXRequest: "true"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2", nil)
		req.SetPathValue("id", "2")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid path value", func(t *testing.T) {
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo", nil)
		req.SetPathValue("id", "foo")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("render failed", func(t *testing.T) {
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2/edit", nil)
		req.SetPathValue("id", "2")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid path value", func(t *testing.T) {
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo/edit", nil)
		req.SetPathValue("id", "foo")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewGetTaskEditHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("render failed", func(t *testing.T) {
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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
		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

			ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

func TestPutTaskTitleHandler(t *testing.T) {
	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)
	}

	for _, test := range []struct {
		Name string
		Body url.Values
	}{
		{Name: "empty", Body: url.Values{"title": []string{""}}},
		{Name: "space", Body: url.Values{"title": []string{"   "}}},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)
			fixture(db, t)

			ctx := SetUserIntoContext(t.Context(), User{ID: 1})
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2/title", strings.NewReader(test.Body.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskTitleHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "bad request\n", rec.Body.String())

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})
	}

	t.Run("task not found", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("title", "new test")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2/title", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "2")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
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

	t.Run("other user", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("title", "new test")

		ctx := SetUserIntoContext(t.Context(), User{ID: 2})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2/title", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
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
		values.Set("title", "new test")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo/title", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "foo")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo/title", strings.NewReader("title=foo&bar=%zz"))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
		h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("title", "new test")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1/title", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskTitleHandler(db)
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
}

func TestPutTaskStatusHandler(t *testing.T) {
	var fixture = func(db *sql.DB, t *testing.T) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)
	}

	for _, test := range []struct {
		Name string
		Body url.Values
	}{
		{Name: "empty", Body: url.Values{"status": []string{""}}},
		{Name: "space", Body: url.Values{"status": []string{"   "}}},
		{Name: "invalid", Body: url.Values{"status": []string{"invalid"}}},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)
			fixture(db, t)

			ctx := SetUserIntoContext(t.Context(), User{ID: 1})
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1/status", strings.NewReader(test.Body.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskStatusHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "bad request\n", rec.Body.String())

			row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
			var task Task
			require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

			assert.Equal(t, "test", task.Title)
			assert.Equal(t, "created", task.Status)
			assert.Equal(t, 1, task.UserID)
		})
	}

	t.Run("task not found", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("status", "done")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2/status", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "2")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskStatusHandler(db)
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

	t.Run("other user", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("status", "done")

		ctx := SetUserIntoContext(t.Context(), User{ID: 2})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1/status", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskStatusHandler(db)
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo/status", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "foo")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskStatusHandler(db)
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1/status", strings.NewReader("status=done&bar=%zz"))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskStatusHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "bad request\n", rec.Body.String())

		row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
		var task Task
		require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

		assert.Equal(t, "test", task.Title)
		assert.Equal(t, "created", task.Status)
		assert.Equal(t, 1, task.UserID)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(db, t)

		values := url.Values{}
		values.Set("status", "done")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
		req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1/status", strings.NewReader(values.Encode()))
		req.SetPathValue("id", "1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")

		rec := httptest.NewRecorder()

		h := NewPutTaskStatusHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "", rec.Body.String())

		row := db.QueryRowContext(t.Context(), "SELECT id, user_id, title, status FROM tasks WHERE id = 1")
		var task Task
		require.NoError(t, row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status))

		assert.Equal(t, "test", task.Title)
		assert.Equal(t, "done", task.Status)
		assert.Equal(t, 1, task.UserID)
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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

		ctx := SetUserIntoContext(t.Context(), User{ID: 1})
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
