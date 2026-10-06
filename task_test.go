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
	t.Run("GetParams", func(t *testing.T) {
		db := NewTestDB(t)
		h := NewListTasksHandler(db)

		t.Run("default values will be used if parameter is empty", func(t *testing.T) {
			values := url.Values{}
			params := h.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			assert.Equal(t, want, params)
		})
		t.Run("specified values will be used", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "2")
			values.Set("limit", "50")
			values.Set("status", "done")
			params := h.GetParams(values)
			want := ListTasksParams{
				Page:   2,
				Limit:  50,
				Status: "done",
			}
			assert.Equal(t, want, params)
		})
		t.Run("default value will be used if page or limit is invalid", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "foo")
			values.Set("limit", "bar")
			params := h.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			assert.Equal(t, want, params)
		})
		t.Run("default value will be used if page or limit is negative", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "-1")
			values.Set("limit", "-1")
			params := h.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			assert.Equal(t, want, params)
		})

		t.Run("too large page and limit", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "1001")
			values.Set("limit", "1001")
			params := h.GetParams(values)
			want := ListTasksParams{
				Page:   1000,
				Limit:  1000,
				Status: "",
			}
			assert.Equal(t, want, params)
		})
	})

	t.Run("GetTasks", func(t *testing.T) {
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
			(1, 'test4', 'created')`,
		)
		require.NoError(t, err)

		h := NewListTasksHandler(db)

		t.Run("get all", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: ""})
			require.NoError(t, err)

			require.Len(t, tasks, 4)

			wants := []struct {
				ID     int
				Title  string
				Status string
				UserID int
			}{
				{ID: 4, Title: "test4", Status: "created", UserID: 1},
				{ID: 1, Title: "test1", Status: "created", UserID: 1},
				{ID: 3, Title: "test3", Status: "done", UserID: 1},
				{ID: 2, Title: "test2", Status: "done", UserID: 1},
			}
			for i := range 4 {
				assert.Equal(t, wants[i].ID, tasks[i].ID)
				assert.Equal(t, wants[i].Title, tasks[i].Title)
				assert.Equal(t, wants[i].Status, tasks[i].Status)
				assert.Equal(t, wants[i].UserID, tasks[i].UserID)
			}
		})

		t.Run("get created", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: "created"})
			require.NoError(t, err)

			require.Len(t, tasks, 2)
			wants := []struct {
				ID     int
				Title  string
				Status string
				UserID int
			}{
				{ID: 4, Title: "test4", Status: "created", UserID: 1},
				{ID: 1, Title: "test1", Status: "created", UserID: 1},
			}
			for i := range 2 {
				assert.Equal(t, wants[i].ID, tasks[i].ID)
				assert.Equal(t, wants[i].Title, tasks[i].Title)
				assert.Equal(t, wants[i].Status, tasks[i].Status)
				assert.Equal(t, wants[i].UserID, tasks[i].UserID)
			}
		})

		t.Run("get done", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: "done"})
			require.NoError(t, err)

			require.Len(t, tasks, 2)
			wants := []struct {
				ID     int
				Title  string
				Status string
				UserID int
			}{
				{ID: 3, Title: "test3", Status: "done", UserID: 1},
				{ID: 2, Title: "test2", Status: "done", UserID: 1},
			}
			for i := range 2 {
				assert.Equal(t, wants[i].ID, tasks[i].ID)
				assert.Equal(t, wants[i].Title, tasks[i].Title)
				assert.Equal(t, wants[i].Status, tasks[i].Status)
				assert.Equal(t, wants[i].UserID, tasks[i].UserID)
			}
		})

		t.Run("no rows", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 2, Limit: 100, Status: ""})
			require.NoError(t, err)

			require.Len(t, tasks, 0)
		})

		t.Run("other user", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 2, ListTasksParams{Page: 1, Limit: 10, Status: ""})
			require.NoError(t, err)

			require.Len(t, tasks, 0)
		})
	})

	t.Run("ResponseHTTP", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []Task{
					{ID: 1, Title: "test title", Status: "created"},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)

			require.NoError(t, h.ResponseHTTP(rec, data))
			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			assert.Contains(t, rec.Body.String(), "<head>")
			assert.Contains(t, rec.Body.String(), "<body")
			assert.Contains(t, rec.Body.String(), "<footer")
		})

		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []Task{},
				LastIndex: -1,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)

			require.NoError(t, h.ResponseHTTP(rec, data))
			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			assert.Contains(t, rec.Body.String(), "<head>")
			assert.Contains(t, rec.Body.String(), "<body")
			assert.Contains(t, rec.Body.String(), "<footer")
		})
	})

	t.Run("ResponseHTMX", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []Task{
					{ID: 1, Title: "test title", Status: "created"},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)

			require.NoError(t, h.ResponseHTMX(rec, data))

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			assert.NotContains(t, rec.Body.String(), "<head>")
			assert.NotContains(t, rec.Body.String(), "<body")
			assert.NotContains(t, rec.Body.String(), "<footer")
		})

		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []Task{},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)

			require.NoError(t, h.ResponseHTMX(rec, data))

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			assert.Empty(t, strings.TrimSpace(rec.Body.String()))
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
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
			(1, 'test4', 'created')`,
		)
		require.NoError(t, err)

		t.Run("get without params", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, "<head>")
			for i := range 4 {
				assert.Contains(t, body, fmt.Sprintf(`id="task-%d"`, i+1))
			}
		})

		t.Run("get with params", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "3")
			values.Set("limit", "5")
			values.Set("status", "created")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("/tasks?%s", values.Encode()), nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			assert.Contains(t, rec.Body.String(), "<head>")
		})

		t.Run("get htmx", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "2")
			values.Set("limit", "10")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("/tasks?%s", values.Encode()), nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)

			assert.NotContains(t, rec.Body.String(), "<head>")
		})

		t.Run("render failed http", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.templateHTTP = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
		})

		t.Run("render failed htmx", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(db)
			h.templateHTMX = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	})
}

func TestGetTaskHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		db := NewTestDB(t)

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)

		t.Run("normal", func(t *testing.T) {
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
	})
}

func TestGetTaskEditHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		db := NewTestDB(t)

		_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
		require.NoError(t, err)
		_, err = db.ExecContext(t.Context(), "INSERT INTO tasks (user_id, title, status) VALUES (1, 'test', 'created')")
		require.NoError(t, err)

		t.Run("normal", func(t *testing.T) {
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
	})
}

func TestPostTaskHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db := NewTestDB(t)

		t.Run("title", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPostTaskHandler(db)
			params, err := h.GetParams(req)
			require.NoError(t, err)

			assert.Equal(t, "test", params.Title)
		})

		t.Run("empty title", func(t *testing.T) {
			values := url.Values{}

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPostTaskHandler(db)
			_, err := h.GetParams(req)
			assert.ErrorIs(t, err, ErrBadRequest)
		})

		t.Run("space only", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", " ")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			require.NoError(t, req.ParseForm())

			h := NewPostTaskHandler(db)
			_, err := h.GetParams(req)
			assert.ErrorIs(t, err, ErrBadRequest)
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db := NewTestDB(t)

			_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
			require.NoError(t, err)

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

		t.Run("invalid request body", func(t *testing.T) {
			db := NewTestDB(t)

			_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
			require.NoError(t, err)

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader("title=test&foo=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM tasks WHERE user_id = 1")
			var count int
			require.NoError(t, row.Scan(&count))
			require.NoError(t, err)

			assert.Equal(t, 0, count)
		})

		t.Run("render failed", func(t *testing.T) {
			db := NewTestDB(t)

			_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('user1', '')")
			require.NoError(t, err)

			values := url.Values{}
			values.Set("title", "test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(db)
			h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		})
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

	t.Run("delete successfully", func(t *testing.T) {
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
}
