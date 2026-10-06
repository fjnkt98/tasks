package server

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fjnkt98/tasks/ent"
	enttask "github.com/fjnkt98/tasks/ent/task"
)

func TestListTasksHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		client := NewTestDB(t)
		h := NewListTasksHandler(client)

		t.Run("default values will be used if parameter is empty", func(t *testing.T) {
			values := url.Values{}
			params := h.GetParams(values)
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
			params := h.GetParams(values)
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
			params := h.GetParams(values)
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
			params := h.GetParams(values)
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
		client := NewTestDB(t)

		if _, err := client.User.CreateBulk(
			client.User.Create().SetName("user1").SetPassword("user1"),
			client.User.Create().SetName("user2").SetPassword("user2"),
		).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		if _, err := client.Task.CreateBulk(
			client.Task.Create().SetTitle("test1").SetStatus(enttask.StatusCreated).SetUserID(1),
			client.Task.Create().SetTitle("test2").SetStatus(enttask.StatusDone).SetUserID(1),
			client.Task.Create().SetTitle("test3").SetStatus(enttask.StatusDone).SetUserID(1),
			client.Task.Create().SetTitle("test4").SetStatus(enttask.StatusCreated).SetUserID(1),
		).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		h := NewListTasksHandler(client)

		t.Run("get all", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: ""})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 4 {
				t.Fatalf("length should be 4, but got %d", len(tasks))
			}

			wants := []struct {
				ID     int
				Title  string
				Status enttask.Status
				UserID int
			}{
				{ID: 4, Title: "test4", Status: enttask.StatusCreated, UserID: 1},
				{ID: 1, Title: "test1", Status: enttask.StatusCreated, UserID: 1},
				{ID: 3, Title: "test3", Status: enttask.StatusDone, UserID: 1},
				{ID: 2, Title: "test2", Status: enttask.StatusDone, UserID: 1},
			}
			for i := range 4 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id should be %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title should be %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status should be %v, but got %v", wants[i].Status, tasks[i].Status)
				}
				if wants[i].UserID != tasks[i].UserID {
					t.Errorf("user id should be %v, but got %v", wants[i].UserID, tasks[i].UserID)
				}
			}
		})

		t.Run("get created", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: "created"})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 2 {
				t.Fatalf("length should be 2, but got %d", len(tasks))
			}
			wants := []struct {
				ID     int
				Title  string
				Status enttask.Status
				UserID int
			}{
				{ID: 4, Title: "test4", Status: enttask.StatusCreated, UserID: 1},
				{ID: 1, Title: "test1", Status: enttask.StatusCreated, UserID: 1},
			}
			for i := range 2 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id should be %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title should be %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status should be %v, but got %v", wants[i].Status, tasks[i].Status)
				}
				if wants[i].UserID != tasks[i].UserID {
					t.Errorf("user id should be %v, but got %v", wants[i].UserID, tasks[i].UserID)
				}
			}
		})

		t.Run("get done", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 1, Limit: 10, Status: "done"})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 2 {
				t.Fatalf("length should be 4, but got %d", len(tasks))
			}
			wants := []struct {
				ID     int
				Title  string
				Status enttask.Status
				UserID int
			}{
				{ID: 3, Title: "test3", Status: enttask.StatusDone, UserID: 1},
				{ID: 2, Title: "test2", Status: enttask.StatusDone, UserID: 1},
			}
			for i := range 2 {
				if wants[i].ID != tasks[i].ID {
					t.Errorf("id should be %v, but got %v", wants[i].ID, tasks[i].ID)
				}
				if wants[i].Title != tasks[i].Title {
					t.Errorf("title should be %v, but got %v", wants[i].Title, tasks[i].Title)
				}
				if wants[i].Status != tasks[i].Status {
					t.Errorf("status should be %v, but got %v", wants[i].Status, tasks[i].Status)
				}
				if wants[i].UserID != tasks[i].UserID {
					t.Errorf("user id should be %v, but got %v", wants[i].UserID, tasks[i].UserID)
				}
			}
		})

		t.Run("no rows", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 1, ListTasksParams{Page: 2, Limit: 100, Status: ""})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 0 {
				t.Fatalf("length should be 0, but got %d", len(tasks))
			}
		})

		t.Run("other user", func(t *testing.T) {
			tasks, err := h.GetTasks(t.Context(), 2, ListTasksParams{Page: 1, Limit: 10, Status: ""})
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 0 {
				t.Fatalf("length should be 0, but got %d", len(tasks))
			}
		})
	})

	t.Run("ResponseHTTP", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []*ent.Task{
					{ID: 1, Title: "test title", Status: enttask.StatusCreated},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)

			if err := h.ResponseHTTP(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("body should contain <head> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<body") {
				t.Error("body should contain <body> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<footer") {
				t.Error("body should contain <footer> element, but not found")
			}
		})

		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []*ent.Task{},
				LastIndex: -1,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)

			if err := h.ResponseHTTP(rec, data); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("body should contain <head> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<body") {
				t.Error("body should contain <body> element, but not found")
			}
			if !strings.Contains(rec.Body.String(), "<footer") {
				t.Error("body should contain <footer> element, but not found")
			}
		})
	})

	t.Run("ResponseHTMX", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			data := TaskData{
				Tasks: []*ent.Task{
					{ID: 1, Title: "test title", Status: enttask.StatusCreated},
				},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)

			if err := h.ResponseHTMX(rec, data); err != nil {
				t.Fatal(err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			if strings.Contains(rec.Body.String(), "<head>") {
				t.Error("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(rec.Body.String(), "<body") {
				t.Error("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(rec.Body.String(), "<footer") {
				t.Error("body shouldn't contain <footer> element, but found")
			}
		})

		t.Run("no data", func(t *testing.T) {
			data := TaskData{
				Tasks:     []*ent.Task{},
				LastIndex: 0,
				NextPage:  2,
				Limit:     10,
				Status:    "",
			}
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)

			if err := h.ResponseHTMX(rec, data); err != nil {
				t.Fatal(err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			if body := strings.TrimSpace(rec.Body.String()); body != "" {
				t.Errorf("body should be empty, but got %s", body)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		client := NewTestDB(t)

		if _, err := client.User.CreateBulk(
			client.User.Create().SetName("user1").SetPassword("user1"),
			client.User.Create().SetName("user2").SetPassword("user2"),
		).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		if _, err := client.Task.CreateBulk(
			client.Task.Create().SetTitle("test1").SetStatus(enttask.StatusCreated).SetUserID(1),
			client.Task.Create().SetTitle("test2").SetStatus(enttask.StatusDone).SetUserID(1),
			client.Task.Create().SetTitle("test3").SetStatus(enttask.StatusDone).SetUserID(1),
			client.Task.Create().SetTitle("test4").SetStatus(enttask.StatusCreated).SetUserID(1),
		).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		t.Run("get without params", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, "<head>") {
				t.Error("body should contain <head> element, but not found")
			}
			for i := range 4 {
				if !strings.Contains(body, fmt.Sprintf(`id="task-%d"`, i+1)) {
					t.Error("body should contain task element, but not found")
				}
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

			h := NewListTasksHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			if !strings.Contains(rec.Body.String(), "<head>") {
				t.Error("body should contain <head> element, but not found")
			}
		})

		t.Run("get htmx", func(t *testing.T) {
			values := url.Values{}
			values.Set("page", "2")
			values.Set("limit", "10")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("/tasks?%s", values.Encode()), nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if strings.Contains(rec.Body.String(), "<head>") {
				t.Error("body shouldn't contain <head> element, but found")
			}
		})

		t.Run("render failed http", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)
			h.templateHTTP = template.Must(template.New("broken").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}
		})

		t.Run("render failed htmx", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks", nil)
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewListTasksHandler(client)
			h.templateHTMX = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}
		})
	})
}

func TestGetTaskHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		client := NewTestDB(t)

		if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Task.Create().SetTitle("test").SetStatus(enttask.StatusCreated).SetUserID(1).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		t.Run("normal", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body>") {
				t.Error("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer>") {
				t.Error("body shouldn't contain <footer> element, but found")
			}
		})

		t.Run("not found", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2", nil)
			req.SetPathValue("id", "2")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})

		t.Run("invalid path value", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo", nil)
			req.SetPathValue("id", "foo")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})

		t.Run("render failed", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskHandler(client)
			h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}
		})
	})
}

func TestGetTaskEditHandler(t *testing.T) {
	t.Run("ServeHTTP", func(t *testing.T) {
		client := NewTestDB(t)

		if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Task.Create().SetTitle("test").SetStatus(enttask.StatusCreated).SetUserID(1).Save(t.Context()); err != nil {
			t.Fatal(err)
		}

		t.Run("normal", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1/edit", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskEditHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body>") {
				t.Error("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer>") {
				t.Error("body shouldn't contain <footer> element, but found")
			}
		})

		t.Run("not found", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/2/edit", nil)
			req.SetPathValue("id", "2")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskEditHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})

		t.Run("invalid path value", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/foo/edit", nil)
			req.SetPathValue("id", "foo")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskEditHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}
		})

		t.Run("render failed", func(t *testing.T) {
			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/tasks/1/edit", nil)
			req.SetPathValue("id", "1")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewGetTaskEditHandler(client)
			h.t = template.Must(template.New("task_edit").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}
		})
	})
}

func TestPostTaskHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("title", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostTaskHandler(client)
			params, err := h.GetParams(req)
			if err != nil {
				t.Fatal(err)
			}

			if params.Title != "test" {
				t.Errorf("title should be 'test', but got %s", params.Title)
			}
		})

		t.Run("empty title", func(t *testing.T) {
			values := url.Values{}

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostTaskHandler(client)
			_, err := h.GetParams(req)
			if !errors.Is(err, ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, but got %s", err)
			}
		})

		t.Run("space only", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", " ")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPostTaskHandler(client)
			_, err := h.GetParams(req)
			if !errors.Is(err, ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, but got %s", err)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			client := NewTestDB(t)

			if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("title", "test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status created, but got %d", rec.Code)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Error("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Error("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Error("body shouldn't contain <footer> element, but found")
			}

			tasks, err := client.Task.Query().Where(enttask.UserID(1)).Limit(100).All(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 1 {
				t.Fatalf("length should be 1, but got %d", len(tasks))
			}

			if tasks[0].Title != "test" {
				t.Errorf("title should be 'test', but got %s", tasks[0].Title)
			}
			if tasks[0].Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", tasks[0].Status)
			}
		})

		t.Run("invalid request body", func(t *testing.T) {
			client := NewTestDB(t)

			if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
				t.Fatal(err)
			}

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader("title=test&foo=%zz"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			tasks, err := client.Task.Query().Where(enttask.UserID(1)).Limit(100).All(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if len(tasks) != 0 {
				t.Fatalf("length should be 0, but got %d", len(tasks))
			}
		})

		t.Run("render failed", func(t *testing.T) {
			client := NewTestDB(t)

			if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
				t.Fatal(err)
			}

			values := url.Values{}
			values.Set("title", "test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPostTaskHandler(client)
			h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}
		})
	})
}

func TestPutTaskHandler(t *testing.T) {
	t.Run("GetParams", func(t *testing.T) {
		client := NewTestDB(t)

		t.Run("normal", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPutTaskHandler(client)
			params := h.GetParams(req)

			if params.Title != "test" {
				t.Errorf("title should be 'test', but got %s", params.Title)
			}
			if params.Status != "done" {
				t.Errorf("status should be 'done', but got %s", params.Status)
			}
		})

		t.Run("empty", func(t *testing.T) {
			values := url.Values{}

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1?title=foo&status=bar", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPutTaskHandler(client)
			params := h.GetParams(req)

			if params.Title != "" {
				t.Errorf("title should be empty, but got %s", params.Title)
			}
			if params.Status != "" {
				t.Errorf("status should be empty, but got %s", params.Status)
			}
		})

		t.Run("title contains space", func(t *testing.T) {
			values := url.Values{}
			values.Set("title", "  ")
			values.Set("status", "")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1?title=foo", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}

			h := NewPutTaskHandler(client)
			params := h.GetParams(req)

			if params.Title != "" {
				t.Errorf("title should be empty, but got %s", params.Title)
			}
			if params.Status != "" {
				t.Errorf("status should be empty, but got %s", params.Status)
			}
		})
	})

	var fixture = func(client *ent.Client, t *testing.T) {
		t.Helper()
		if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Task.Create().SetTitle("test").SetStatus(enttask.StatusCreated).SetUserID(1).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("UpdateTask", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			params := UpdateTaskParams{
				Title:  "new test",
				Status: "done",
			}

			h := NewPutTaskHandler(client)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "new test" {
				t.Errorf("title should be 'new test', but got %s", updated.Title)
			}
			if updated.Status != enttask.StatusDone {
				t.Errorf("status should be 'done', but got %s", updated.Status)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.ID != updated.ID || task.Title != updated.Title ||
				task.Status != updated.Status || task.UserID != updated.UserID ||
				!task.CreatedAt.Equal(updated.CreatedAt) || !task.UpdatedAt.Equal(updated.UpdatedAt) {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})

		t.Run("update non-existing task", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			h := NewPutTaskHandler(client)
			_, err := h.UpdateTask(t.Context(), 2, 1, UpdateTaskParams{})
			if !ent.IsNotFound(err) {
				t.Fatalf("expected ent.NotFoundError, but got %+v", err)
			}
		})

		t.Run("update title only", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			params := UpdateTaskParams{
				Title: "new test",
			}

			h := NewPutTaskHandler(client)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "new test" {
				t.Errorf("title should be 'new test', but got %s", updated.Title)
			}
			if updated.Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", updated.Status)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.ID != updated.ID || task.Title != updated.Title ||
				task.Status != updated.Status || task.UserID != updated.UserID ||
				!task.CreatedAt.Equal(updated.CreatedAt) || !task.UpdatedAt.Equal(updated.UpdatedAt) {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})

		t.Run("update status only", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			params := UpdateTaskParams{
				Status: "done",
			}

			h := NewPutTaskHandler(client)
			updated, err := h.UpdateTask(t.Context(), 1, 1, params)
			if err != nil {
				t.Fatal(err)
			}

			if updated.Title != "test" {
				t.Errorf("title should be 'test', but got %s", updated.Title)
			}
			if updated.Status != enttask.StatusDone {
				t.Errorf("status should be 'done', but got %s", updated.Status)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.ID != updated.ID || task.Title != updated.Title ||
				task.Status != updated.Status || task.UserID != updated.UserID ||
				!task.CreatedAt.Equal(updated.CreatedAt) || !task.UpdatedAt.Equal(updated.UpdatedAt) {
				t.Errorf("updated task %+v must match saved one %+v", updated, task)
			}
		})
	})

	t.Run("ServeHTTP", func(t *testing.T) {
		t.Run("update title and status", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("title", "new test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("body shouldn't contain <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("body should contain `id=\"task-1\"`, but not found")
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "new test" {
				t.Errorf("title should be 'new test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusDone {
				t.Errorf("status should 'done', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("update title only", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("title", "new test")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("body shouldn't contain <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("body should contain `id=\"task-1\"`, but not found")
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "new test" {
				t.Errorf("title should be 'new test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("update status only", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status ok, but got %d", rec.Code)
			}

			if contentType := rec.Result().Header.Get("Content-Type"); contentType != "text/html; charset=utf-8" {
				t.Errorf("Content-Type header should be 'text/html; charset=utf-8', but got '%s'", contentType)
			}

			body := rec.Body.String()
			if strings.Contains(body, "<head>") {
				t.Errorf("body shouldn't contain <head> element, but found")
			}
			if strings.Contains(body, "<body") {
				t.Errorf("body shouldn't contain <body> element, but found")
			}
			if strings.Contains(body, "<footer") {
				t.Errorf("body shouldn't contain <footer> element, but found")
			}

			if !strings.Contains(body, `id="task-1"`) {
				t.Error("body should contain `id=\"task-1\"`, but not found")
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("title should be 'test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusDone {
				t.Errorf("status should be 'done', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("update non-existing task", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/2", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "2")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}

			if body := rec.Body.String(); body != "not found\n" {
				t.Errorf("body should be 'not found', but got '%s'", body)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("title should be 'test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("invalid path value", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "foo")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status not found, but got %d", rec.Code)
			}

			if body := rec.Body.String(); body != "not found\n" {
				t.Errorf("body should be 'not found', but got '%s'", body)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("title should be 'test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("invalid request body", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/foo", strings.NewReader("title=foo&bar=%zz"))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status bad request, but got %d", rec.Code)
			}

			task, err := client.Task.Query().Where(enttask.ID(1), enttask.UserID(1)).Only(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if task.Title != "test" {
				t.Errorf("title should be 'test', but got %s", task.Title)
			}
			if task.Status != enttask.StatusCreated {
				t.Errorf("status should be 'created', but got %s", task.Status)
			}
			if task.UserID != 1 {
				t.Errorf("user id should be 1, but got %d", task.UserID)
			}
		})

		t.Run("render failed", func(t *testing.T) {
			client := NewTestDB(t)

			fixture(client, t)

			values := url.Values{}
			values.Set("title", "new test")
			values.Set("status", "done")

			ctx := SetUserIDIntoContext(t.Context(), 1)
			req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/tasks/1", strings.NewReader(values.Encode()))
			req.SetPathValue("id", "1")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rec := httptest.NewRecorder()

			h := NewPutTaskHandler(client)
			h.t = template.Must(template.New("tasks").Parse("<p>{{ .MissingField }}</p>"))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected status internal server error, but got %d", rec.Code)
			}
		})
	})
}

func TestDeleteTaskHandler(t *testing.T) {
	var fixture = func(client *ent.Client, t *testing.T) {
		t.Helper()
		if _, err := client.User.Create().SetName("user1").SetPassword("user1").Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Task.Create().SetTitle("test").SetStatus(enttask.StatusCreated).SetUserID(1).Save(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("delete successfully", func(t *testing.T) {
		client := NewTestDB(t)

		fixture(client, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/1", nil)
		req.SetPathValue("id", "1")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(client)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		tasks, err := client.Task.Query().Limit(100).All(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 0 {
			t.Errorf("expected no tasks, but got %+v", tasks)
		}
	})

	t.Run("delete non-existing task", func(t *testing.T) {
		client := NewTestDB(t)

		fixture(client, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/2", nil)
		req.SetPathValue("id", "2")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(client)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status ok, but got %d", rec.Code)
		}

		tasks, err := client.Task.Query().Where(enttask.UserID(1)).Limit(100).All(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 1 {
			t.Fatalf("expected 1 tasks, but got %+v", tasks)
		}

		if tasks[0].Title != "test" {
			t.Errorf("title should be 'test', but got %s", tasks[0].Title)
		}
		if tasks[0].Status != enttask.StatusCreated {
			t.Errorf("status should be 'created', but got %s", tasks[0].Status)
		}
	})

	t.Run("invalid path value", func(t *testing.T) {
		client := NewTestDB(t)

		fixture(client, t)

		ctx := SetUserIDIntoContext(t.Context(), 1)
		req := httptest.NewRequestWithContext(ctx, http.MethodDelete, "/tasks/foo", nil)
		req.SetPathValue("id", "foo")

		rec := httptest.NewRecorder()

		h := NewDeleteTaskHandler(client)
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status not found, but got %d", rec.Code)
		}

		tasks, err := client.Task.Query().Where(enttask.UserID(1)).Limit(100).All(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if len(tasks) != 1 {
			t.Fatalf("expected 1 tasks, but got %+v", tasks)
		}

		if tasks[0].Title != "test" {
			t.Errorf("title should be 'test', but got %s", tasks[0].Title)
		}
		if tasks[0].Status != enttask.StatusCreated {
			t.Errorf("status should be 'created', but got %s", tasks[0].Status)
		}
	})
}
