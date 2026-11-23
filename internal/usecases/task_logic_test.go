package usecases_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
)

func TestPrepareAddTask(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "ok: タイトルが妥当ならタスクを準備できる",
			title: "buy milk",
			want:  "",
		},
		{
			name:  "ng: タイトルが空ならエラーになる",
			title: "",
			want:  "task title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := usecases.PrepareAddTask(user, tt.title)
			if tt.want != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.want)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, task.ID())
			assert.Equal(t, tt.title, task.Title())
			assert.Equal(t, domain.TaskStatusCreated, task.Status())
		})
	}
}

func TestEventTaskAdded(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)
	task := mustTask(t, 7, &user, "buy milk", domain.TaskStatusCreated)

	tests := []struct {
		name        string
		wantKind    string
		wantMessage string
	}{
		{
			name:        "ok: 追加イベントのメッセージはタスクIDとタイトルを含む",
			wantKind:    domain.ActivityKindTask,
			wantMessage: `add task 7, title: "buy milk"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventTaskAdded(&user, task)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
			assert.Equal(t, user.ID(), event.User().ID())
		})
	}
}

func TestPrepareUpdateTaskTitle(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)
	task := mustTask(t, 3, &user, "old", domain.TaskStatusCreated)

	tests := []struct {
		name      string
		title     string
		wantTitle string
		wantErr   string
	}{
		{
			name:      "ok: 新しいタイトルでタスクを更新できる",
			title:     "new title",
			wantTitle: "new title",
			wantErr:   "",
		},
		{
			name:      "ng: 空タイトルでは更新できない",
			title:     "  ",
			wantTitle: "",
			wantErr:   "task title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := usecases.PrepareUpdateTaskTitle(task, tt.title)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, got.Title())
			assert.Equal(t, task.ID(), got.ID())
		})
	}
}

func TestEventTaskTitleUpdated(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)
	task := mustTask(t, 3, &user, "renamed", domain.TaskStatusCreated)

	tests := []struct {
		name        string
		wantMessage string
	}{
		{
			name:        "ok: タイトル更新イベントのメッセージは編集内容を表す",
			wantMessage: `edit task 3, title: "renamed"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventTaskTitleUpdated(&user, task)
			require.NoError(t, err)
			assert.Equal(t, domain.ActivityKindTask, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
		})
	}
}

func TestPrepareUpdateTaskStatus(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)
	created := mustTask(t, 5, &user, "work", domain.TaskStatusCreated)

	tests := []struct {
		name       string
		task       domain.Task
		status     string
		wantStatus domain.TaskStatus
		wantErr    string
	}{
		{
			name:       "ok: createdからdoneへ遷移できる",
			task:       created,
			status:     "done",
			wantStatus: domain.TaskStatusDone,
			wantErr:    "",
		},
		{
			name:       "ng: 不正なステータスはエラーになる",
			task:       created,
			status:     "archived",
			wantStatus: "",
			wantErr:    "invalid task status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, st, err := usecases.PrepareUpdateTaskStatus(tt.task, tt.status)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, st)
			assert.Equal(t, tt.wantStatus, got.Status())
		})
	}
}

func TestEventTaskStatusUpdated(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)
	task := mustTask(t, 5, &user, "work", domain.TaskStatusDone)

	tests := []struct {
		name        string
		status      domain.TaskStatus
		wantMessage string
	}{
		{
			name:        "ok: doneへ更新するとdoneイベントになる",
			status:      domain.TaskStatusDone,
			wantMessage: `done task 5, "work"`,
		},
		{
			name:        "ok: createdへ戻すとundoneイベントになる",
			status:      domain.TaskStatusCreated,
			wantMessage: `undone task 5, "work"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventTaskStatusUpdated(&user, task, tt.status)
			require.NoError(t, err)
			assert.Equal(t, domain.ActivityKindTask, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
		})
	}
}

func TestEventTaskDeleted(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)

	tests := []struct {
		name        string
		taskID      int
		wantMessage string
	}{
		{
			name:        "ok: 削除イベントのメッセージはタスクIDを含む",
			taskID:      9,
			wantMessage: "delete task 9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventTaskDeleted(&user, tt.taskID)
			require.NoError(t, err)
			assert.Equal(t, domain.ActivityKindTask, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
		})
	}
}
