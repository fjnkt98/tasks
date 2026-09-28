package server

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fjnkt98/tasks/repository"
)

func TestListTasksService(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck
		s := NewListTaskService(db)

		t.Run("default values will be used if parameter is empty", func(t *testing.T) {
			values := url.Values{}
			params := s.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			if params != want {
				t.Errorf("expected %+v, but got %+v", want, params)
			}
		})
		t.Run("specified values will be used", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "2")
			values.Set("limit", "50")
			values.Set("status", "done")
			params := s.GetParams(values)
			want := ListTasksParams{
				Page:   2,
				Limit:  50,
				Status: "done",
			}
			if params != want {
				t.Errorf("expected %+v, but got %+v", want, params)
			}
		})
		t.Run("default value will be used if page or limit is invalid", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "foo")
			values.Set("limit", "bar")
			params := s.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			if params != want {
				t.Errorf("expected %+v, but got %+v", want, params)
			}
		})
		t.Run("default value will be used if page or limit is negative", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "-1")
			values.Set("limit", "-1")
			params := s.GetParams(values)
			want := ListTasksParams{
				Page:   1,
				Limit:  10,
				Status: "",
			}
			if params != want {
				t.Errorf("expected %+v, but got %+v", want, params)
			}
		})
	})

	t.Run("GetTasks", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		for i, data := range []struct {
			Title  string
			Status string
		}{
			{Title: "test1", Status: "created"},
			{Title: "test2", Status: "done"},
			{Title: "test3", Status: "done"},
			{Title: "test4", Status: "created"},
		} {
			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", i+1, data.Title, data.Status); err != nil {
				t.Fatal(err)
			}
		}

		s := NewListTaskService(db)

		t.Run("get all", func(t *testing.T) {
			tasks, err := s.GetTasks(t.Context(), ListTasksParams{Page: 1, Limit: 10, Status: ""})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 4 {
				t.Fatalf("expected length is 4, but got %d", len(tasks))
			}
			wants := []struct {
				ID     int64
				Title  string
				Status string
			}{
				{ID: 4, Title: "test4", Status: "created"},
				{ID: 1, Title: "test1", Status: "created"},
				{ID: 3, Title: "test3", Status: "done"},
				{ID: 2, Title: "test2", Status: "done"},
			}
			for i := range 4 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id expected %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title expected %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status expected %v, but got %v", wants[i].Status, tasks[i].Status)
				}
			}
		})
		t.Run("get created", func(t *testing.T) {
			tasks, err := s.GetTasks(t.Context(), ListTasksParams{Page: 1, Limit: 10, Status: "created"})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 2 {
				t.Fatalf("expected length is 2, but got %d", len(tasks))
			}
			wants := []struct {
				ID     int64
				Title  string
				Status string
			}{
				{ID: 4, Title: "test4", Status: "created"},
				{ID: 1, Title: "test1", Status: "created"},
			}
			for i := range 2 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id expected %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title expected %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status expected %v, but got %v", wants[i].Status, tasks[i].Status)
				}
			}
		})
		t.Run("get done", func(t *testing.T) {
			tasks, err := s.GetTasks(t.Context(), ListTasksParams{Page: 1, Limit: 10, Status: "done"})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 2 {
				t.Fatalf("expected length is 4, but got %d", len(tasks))
			}
			wants := []struct {
				ID     int64
				Title  string
				Status string
			}{
				{ID: 3, Title: "test3", Status: "done"},
				{ID: 2, Title: "test2", Status: "done"},
			}
			for i := range 2 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id expected %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title expected %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status expected %v, but got %v", wants[i].Status, tasks[i].Status)
				}
			}
		})
		t.Run("no rows", func(t *testing.T) {
			tasks, err := s.GetTasks(t.Context(), ListTasksParams{Page: 2, Limit: 100, Status: ""})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 0 {
				t.Fatalf("expected length is 0, but got %d", len(tasks))
			}
		})
	})

	t.Run("ResponseHTTP", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		s := NewListTaskService(db)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []repository.Task{
					{ID: 1, Title: "test title", Status: "created"},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			if err := s.ResponseHTTP(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected contains <head> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<body") {
				t.Error("expected contains <body> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<footer") {
				t.Error("expected contains <footer> element, but not found")
			}
		})
		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []repository.Task{},
				LastIndex: -1,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			if err := s.ResponseHTTP(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected contains <head> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<body") {
				t.Error("expected contains <body> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<footer") {
				t.Error("expected contains <footer> element, but not found")
			}
		})
	})
	t.Run("ResponseHTMX", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		s := NewListTaskService(db)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []repository.Task{
					{ID: 1, Title: "test title", Status: "created"},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			if err := s.ResponseHTMX(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected not contains <head> element, but found")
			}
			if strings.Contains(rec.Body.String(), "<body") {
				t.Error("expected not contains <body> element, but found")
			}
			if strings.Contains(rec.Body.String(), "<footer") {
				t.Error("expected not contains <footer> element, but found")
			}
		})
		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []repository.Task{},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			if err := s.ResponseHTMX(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if body := strings.TrimSpace(rec.Body.String()); body != "" {
				t.Errorf("expected empty respons body, but got %s", body)
			}
		})
	})
	t.Run("ServeHTTP", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		for i, data := range []struct {
			Title  string
			Status string
		}{
			{Title: "test1", Status: "created"},
			{Title: "test2", Status: "done"},
			{Title: "test3", Status: "done"},
			{Title: "test4", Status: "created"},
		} {
			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", i+1, data.Title, data.Status); err != nil {
				t.Fatal(err)
			}
		}

		s := NewListTaskService(db)

		t.Run("get without params", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks", nil)
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected contains <head> element, but not found")
			}
		})
		t.Run("get with params", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "3")
			values.Set("limit", "5")
			values.Set("status", "created")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("/tasks?%s", values.Encode()), nil)
			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected contains <head> element, but not found")
			}
		})
		t.Run("get htmx", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "2")
			values.Set("limit", "10")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("/tasks?%s", values.Encode()), nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "<head>") {
				t.Error("expected not contains <head> element, but not found")
			}
		})
	})
}

func TestGetTaskService(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
			t.Fatal(err)
		}

		t.Run("normal", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/1", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("expected not contains <head> element, but fouond")
			}
			if strings.Contains(body, "<body>") {
				t.Error("expected not contains <body> element, but fouond")
			}
			if strings.Contains(body, "<footer>") {
				t.Error("expected not contains <footer> element, but fouond")
			}
		})

		t.Run("not found", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/2", nil)
			req.SetPathValue("id", "2")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})

		t.Run("invalid path value", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/foo", nil)
			req.SetPathValue("id", "foo")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})
	})
}

func TestGetTaskEditService(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
			t.Fatal(err)
		}

		t.Run("normal", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/1/edit", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskEditService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("expected not contains <head> element, but fouond")
			}
			if strings.Contains(body, "<body>") {
				t.Error("expected not contains <body> element, but fouond")
			}
			if strings.Contains(body, "<footer>") {
				t.Error("expected not contains <footer> element, but fouond")
			}
		})
		t.Run("not found", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/2/edit", nil)
			req.SetPathValue("id", "2")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskEditService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})
		t.Run("invalid path value", func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/foo/edit", nil)
			req.SetPathValue("id", "foo")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewGetTaskEditService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})
	})
}

func TestPostTaskService(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		t.Run("title", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPostTaskService(db)
			params, err := s.GetParams(req)
			if err != nil {
				t.Fatal(err)
			}

			if params.Title != "test" {
				t.Errorf("expected title value = 'test', but got %s", params.Title)
			}
		})

		t.Run("empty title", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "")
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPostTaskService(db)
			_, err := s.GetParams(req)
			if !errors.Is(err, ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, but got %s", err)
			}
		})

		t.Run("space only", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", " ")
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPostTaskService(db)
			_, err := s.GetParams(req)
			if !errors.Is(err, ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, but got %s", err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("normal", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			values := url.Values{}
			values.Set("title", "test")
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPostTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status created, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("expected not contains <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Error("expected not contains <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Error("expected not contains <footer> element, but found")
			}

			q := repository.New(db)
			tasks, err := q.ListTasks(t.Context(), repository.ListTasksParams{Offset: 0, Limit: 100})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 1 {
				t.Fatalf("expected tasks count is 1, but got %d", len(tasks))
			}

			if tasks[0].Title != "test" {
				t.Errorf("expected title is 'test', but got %s", tasks[0].Title)
			}
			if tasks[0].Status != "created" {
				t.Errorf("expected status is 'created', but got %s", tasks[0].Status)
			}
		})
	})
}

func TestPutTaskService(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		t.Run("normal", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")
			values.Set("status", "done")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPutTaskService(db)
			params := s.GetParams(req)

			if params.Title != "test" {
				t.Errorf("expected title is 'test', but got %s", params.Title)
			}
			if params.Status != "done" {
				t.Errorf("expected status is 'done', but got %s", params.Status)
			}
		})

		t.Run("empty", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "")
			values.Set("status", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPutTaskService(db)
			params := s.GetParams(req)

			if params.Title != "" {
				t.Errorf("expected title is empty, but got %s", params.Title)
			}
			if params.Status != "" {
				t.Errorf("expected status is empty, but got %s", params.Status)
			}
		})

		t.Run("title contains space", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "  ")
			values.Set("status", "")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			s := NewPutTaskService(db)
			params := s.GetParams(req)

			if params.Title != "" {
				t.Errorf("expected title is empty, but got %s", params.Title)
			}
			if params.Status != "" {
				t.Errorf("expected status is empty, but got %s", params.Status)
			}
		})
	})

	t.Run("UpdateTask", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			params := UpdateTaskParams{
				Title:  "new test",
				Status: "done",
			}

			s := NewPutTaskService(db)
			updated, err := s.UpdateTask(t.Context(), 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "new test" {
				t.Errorf("expected title is 'new test', but got %s", updated.Title)
			}
			if updated.Status != "done" {
				t.Errorf("expected status is 'done', but got %s", updated.Status)
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task != updated {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})
		t.Run("update non-existing task", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			s := NewPutTaskService(db)
			_, err = s.UpdateTask(t.Context(), 2, UpdateTaskParams{})
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("expected sql.ErrNoRows, but got %+v", err)
			}
		})
		t.Run("update title only", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			params := UpdateTaskParams{
				Title: "new test",
			}

			s := NewPutTaskService(db)
			updated, err := s.UpdateTask(t.Context(), 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "new test" {
				t.Errorf("expected title is 'new test', but got %s", updated.Title)
			}
			if updated.Status != "created" {
				t.Errorf("expected status is 'created', but got %s", updated.Status)
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task != updated {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})
		t.Run("update status only", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			params := UpdateTaskParams{
				Status: "done",
			}

			s := NewPutTaskService(db)
			updated, err := s.UpdateTask(t.Context(), 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "test" {
				t.Errorf("expected title is 'test', but got %s", updated.Title)
			}
			if updated.Status != "done" {
				t.Errorf("expected status is 'done', but got %s", updated.Status)
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task != updated {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("title", "new test")
			values.Set("status", "done")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPutTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("expected not contains <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("expected not contains <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("expected not contains <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("expected contains `id=\"task-1\"`, but not found")
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "new test" {
				t.Errorf("expected title is 'new test', but got %s", task.Title)
			}
			if task.Status != "done" {
				t.Errorf("expected status is 'done', but got %s", task.Status)
			}
		})
		t.Run("update title only", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("title", "new test")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPutTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("expected not contains <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("expected not contains <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("expected not contains <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("expected contains `id=\"task-1\"`, but not found")
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "new test" {
				t.Errorf("expected title is 'new test', but got %s", task.Title)
			}
			if task.Status != "created" {
				t.Errorf("expected status is 'created', but got %s", task.Status)
			}
		})
		t.Run("update status only", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("status", "done")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPutTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("expected not contains <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("expected not contains <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("expected not contains <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("expected contains `id=\"task-1\"`, but not found")
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("expected title is 'test', but got %s", task.Title)
			}
			if task.Status != "done" {
				t.Errorf("expected status is 'done', but got %s", task.Status)
			}
		})
		t.Run("update non-existing task", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("status", "done")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/2", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "2")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPutTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}

			if body := rec.Body.String(); body != "not found\n" {
				t.Errorf("expected response body is 'not found', but got '%s'", body)
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("expected title is 'test', but got %s", task.Title)
			}
			if task.Status != "created" {
				t.Errorf("expected status is 'created', but got %s", task.Status)
			}
		})
		t.Run("invalid path value", func(t *testing.T) {
			db, err := repository.NewTestDB()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close() // nolint:errcheck

			if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("status", "done")

			req := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/tasks/foo", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "foo")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			s := NewPutTaskService(db)
			s.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}

			if body := rec.Body.String(); body != "not found\n" {
				t.Errorf("expected response body is 'not found', but got '%s'", body)
			}

			q := repository.New(db)
			task, err := q.GetTaskByID(t.Context(), 1)
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("expected title is 'test', but got %s", task.Title)
			}
			if task.Status != "created" {
				t.Errorf("expected status is 'created', but got %s", task.Status)
			}
		})
	})
}

func TestDeleteTaskService(t *testing.T) {
	t.Run("delete successfully", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/tasks/1", nil)
		req.SetPathValue("id", "1")

		rec := httptest.NewRecorder()

		s := NewDeleteTaskService(db)
		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		q := repository.New(db)
		tasks, err := q.ListTasks(t.Context(), repository.ListTasksParams{Offset: 0, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected no tasks, but got %+v", tasks)
		}
	})
	t.Run("delete non-existing task", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/tasks/2", nil)
		req.SetPathValue("id", "2")

		rec := httptest.NewRecorder()

		s := NewDeleteTaskService(db)
		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		q := repository.New(db)
		tasks, err := q.ListTasks(t.Context(), repository.ListTasksParams{Offset: 0, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 tasks, but got %+v", tasks)
		}

		if tasks[0].Title != "test" {
			t.Errorf("expected title is 'test', but got %s", tasks[0].Title)
		}
		if tasks[0].Status != "created" {
			t.Errorf("expected status is 'created', but got %s", tasks[0].Status)
		}
	})
	t.Run("invalid path value", func(t *testing.T) {
		db, err := repository.NewTestDB()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close() // nolint:errcheck

		if _, err := db.ExecContext(t.Context(), "INSERT INTO tasks (id, title, status) VALUES (?, ?, ?);", 1, "test", "created"); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/tasks/foo", nil)
		req.SetPathValue("id", "foo")

		rec := httptest.NewRecorder()

		s := NewDeleteTaskService(db)
		s.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status not found, but got %d", rec.Code)
		}

		q := repository.New(db)
		tasks, err := q.ListTasks(t.Context(), repository.ListTasksParams{Offset: 0, Limit: 100})
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 1 {
			t.Errorf("expected 1 tasks, but got %+v", tasks)
		}

		if tasks[0].Title != "test" {
			t.Errorf("expected title is 'test', but got %s", tasks[0].Title)
		}
		if tasks[0].Status != "created" {
			t.Errorf("expected status is 'created', but got %s", tasks[0].Status)
		}
	})
}
