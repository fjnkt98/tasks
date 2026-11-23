package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
)

func TestNewTask(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)

	tests := []struct {
		name    string
		user    *domain.User
		title   string
		status  domain.TaskStatus
		wantErr error
	}{
		{
			name:    "ok: タイトルと所有者があればタスクを作れる",
			user:    &user,
			title:   "buy milk",
			status:  domain.TaskStatusCreated,
			wantErr: nil,
		},
		{
			name:    "ng: 所有者が無いとタスクを作れない",
			user:    nil,
			title:   "buy milk",
			status:  domain.TaskStatusCreated,
			wantErr: domain.ErrTaskUserRequired,
		},
		{
			name:    "ng: タイトルが空だとタスクを作れない",
			user:    &user,
			title:   "   ",
			status:  domain.TaskStatusCreated,
			wantErr: domain.ErrEmptyTaskTitle,
		},
		{
			name:    "ng: 不正なステータスだとタスクを作れない",
			user:    &user,
			title:   "buy milk",
			status:  domain.TaskStatus("bogus"),
			wantErr: domain.ErrInvalidTaskStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			task, err := domain.NewTask(0, tt.user, tt.title, tt.status)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "buy milk", task.Title())
			assert.Equal(t, domain.TaskStatusCreated, task.Status())
		})
	}
}

func TestTaskWithTitle(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)
	task, err := domain.NewTask(1, &user, "old", domain.TaskStatusCreated)
	require.NoError(t, err)

	tests := []struct {
		name    string
		title   string
		want    string
		wantErr error
	}{
		{
			name:    "ok: タイトルを更新できる",
			title:   "new title",
			want:    "new title",
			wantErr: nil,
		},
		{
			name:    "ng: 空のタイトルには更新できない",
			title:   "",
			want:    "",
			wantErr: domain.ErrEmptyTaskTitle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := task.WithTitle(tt.title)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Title())
		})
	}
}

func TestParseTaskStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    domain.TaskStatus
		wantErr error
	}{
		{
			name:    "ok: created をパースできる",
			input:   "created",
			want:    domain.TaskStatusCreated,
			wantErr: nil,
		},
		{
			name:    "ok: done をパースできる",
			input:   "done",
			want:    domain.TaskStatusDone,
			wantErr: nil,
		},
		{
			name:    "ng: 未知のステータスは拒否する",
			input:   "pending",
			want:    "",
			wantErr: domain.ErrInvalidTaskStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseTaskStatus(tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
