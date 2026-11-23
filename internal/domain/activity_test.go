package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
)

func TestNewActivity(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)

	tests := []struct {
		name    string
		user    *domain.User
		kind    string
		message string
		wantErr error
	}{
		{
			name:    "ok: ユーザーと種別とメッセージがあればアクティビティを作れる",
			user:    &user,
			kind:    "task",
			message: "add task",
			wantErr: nil,
		},
		{
			name:    "ng: ユーザーが無いとアクティビティを作れない",
			user:    nil,
			kind:    "task",
			message: "add task",
			wantErr: domain.ErrActivityUserRequired,
		},
		{
			name:    "ng: 種別が空だとアクティビティを作れない",
			user:    &user,
			kind:    " ",
			message: "add task",
			wantErr: domain.ErrEmptyActivityKind,
		},
		{
			name:    "ng: メッセージが空だとアクティビティを作れない",
			user:    &user,
			kind:    "task",
			message: "",
			wantErr: domain.ErrEmptyActivityMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			activity, err := domain.NewActivity(tt.user, tt.kind, tt.message, time.Time{})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "task", activity.Kind())
			assert.Equal(t, "add task", activity.Message())
		})
	}
}
