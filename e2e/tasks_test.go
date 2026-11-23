package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTasksCreate(t *testing.T) {
	for _, tt := range []struct {
		name      string
		signedIn  bool
		title     string
		wantCode  int
		wantPath  string
		wantTasks int
	}{
		{
			name:      "ok: タスクを作成すると、DB に保存される",
			signedIn:  true,
			title:     "buy milk",
			wantCode:  http.StatusCreated,
			wantPath:  "",
			wantTasks: 1,
		},
		{
			name:      "ng: 未サインインでタスクを作ろうとすると、サインインへリダイレクトされる",
			signedIn:  false,
			title:     "buy milk",
			wantCode:  http.StatusOK,
			wantPath:  "/account/signin",
			wantTasks: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.signedIn {
				signInAs(t, e, "test")
			}

			res := postForm(t, e, "/tasks", url.Values{
				"title": []string{tt.title},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			if tt.wantPath != "" {
				assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			}
			assert.Equal(t, tt.wantTasks, taskCount(t, e.db, tt.title))

			if tt.signedIn && tt.wantTasks > 0 {
				uid := userID(t, e.db, "test")
				id := 0
				require.NoError(t, e.db.QueryRow(`SELECT id FROM tasks WHERE title = ?`, tt.title).Scan(&id))
				assert.Equal(t, 1, activityCount(t, e.db, uid, "task", fmt.Sprintf("add task %d, title: %q", id, tt.title)))
			}
		})
	}
}

func TestTasksUpdateTitle(t *testing.T) {
	for _, tt := range []struct {
		name      string
		initial   string
		newTitle  string
		wantCode  int
		wantTitle string
	}{
		{
			name:      "ok: タイトルを更新すると、DB のタイトルが変わる",
			initial:   "buy milk",
			newTitle:  "buy oat milk",
			wantCode:  http.StatusOK,
			wantTitle: "buy oat milk",
		},
		{
			name:      "ng: 空のタイトルで更新すると、400 になる",
			initial:   "buy milk",
			newTitle:  "   ",
			wantCode:  http.StatusBadRequest,
			wantTitle: "buy milk",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			id := createTask(t, e, tt.initial)
			uid := userID(t, e.db, "test")

			res := doForm(t, e, http.MethodPut, taskPath(id, "title"), url.Values{
				"title": []string{tt.newTitle},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantTitle, taskTitle(t, e.db, id))

			wantEdit := 0
			if tt.wantCode == http.StatusOK {
				wantEdit = 1
			}
			assert.Equal(t, wantEdit, activityCount(t, e.db, uid, "task", fmt.Sprintf("edit task %d, title: %q", id, tt.wantTitle)))
		})
	}
}

func TestTasksUpdateStatus(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     string
		wantCode   int
		wantStatus string
		wantActMsg string
	}{
		{
			name:       "ok: ステータスを done にすると、DB のステータスが変わる",
			status:     "done",
			wantCode:   http.StatusNoContent,
			wantStatus: "done",
			wantActMsg: "done task %d, %q",
		},
		{
			name:       "ng: 不正なステータスで更新すると、400 になる",
			status:     "archived",
			wantCode:   http.StatusBadRequest,
			wantStatus: "created",
			wantActMsg: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			id := createTask(t, e, "buy milk")
			uid := userID(t, e.db, "test")
			require.Equal(t, "created", taskStatus(t, e.db, id))

			res := doForm(t, e, http.MethodPut, taskPath(id, "status"), url.Values{
				"status": []string{tt.status},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantStatus, taskStatus(t, e.db, id))

			if tt.wantActMsg != "" {
				msg := fmt.Sprintf(tt.wantActMsg, id, "buy milk")
				assert.Equal(t, 1, activityCount(t, e.db, uid, "task", msg))
			}
		})
	}
}

func TestTasksDelete(t *testing.T) {
	for _, tt := range []struct {
		name      string
		title     string
		wantCode  int
		wantTasks int
	}{
		{
			name:      "ok: タスクを削除すると、DB から消える",
			title:     "buy milk",
			wantCode:  http.StatusOK,
			wantTasks: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			id := createTask(t, e, tt.title)
			uid := userID(t, e.db, "test")
			require.Equal(t, 1, taskCount(t, e.db, tt.title))

			res := doForm(t, e, http.MethodDelete, taskPath(id, ""), url.Values{})
			discard(t, res.Body)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantTasks, taskCount(t, e.db, tt.title))
			assert.Equal(t, 1, activityCount(t, e.db, uid, "task", fmt.Sprintf("delete task %d", id)))
		})
	}
}

func TestTasksPages(t *testing.T) {
	for _, tt := range []struct {
		name         string
		pathSuffix   string
		wantCode     int
		wantContains string
	}{
		{
			name:         "ok: タスク一覧を開くと、作成したタスクが見える",
			pathSuffix:   "",
			wantCode:     http.StatusOK,
			wantContains: "buy milk",
		},
		{
			name:         "ok: タスク詳細を開くと、タイトルが見える",
			pathSuffix:   "show",
			wantCode:     http.StatusOK,
			wantContains: "buy milk",
		},
		{
			name:         "ok: タスク編集を開くと、タイトルが見える",
			pathSuffix:   "edit",
			wantCode:     http.StatusOK,
			wantContains: "buy milk",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			id := createTask(t, e, "buy milk")

			path := "/tasks"
			switch tt.pathSuffix {
			case "show":
				path = taskPath(id, "")
			case "edit":
				path = taskPath(id, "edit")
			}

			res, body := get(t, e, path)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Contains(t, body, tt.wantContains)
		})
	}
}
