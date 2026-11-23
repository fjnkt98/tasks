package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
)

func TestNewSession(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)
	expires := time.Now().Add(time.Hour)

	tests := []struct {
		name    string
		user    *domain.User
		token   string
		wantErr error
	}{
		{
			name:    "ok: ユーザーとトークンがあればセッションを作れる",
			user:    &user,
			token:   "tok",
			wantErr: nil,
		},
		{
			name:    "ng: ユーザーが無いとセッションを作れない",
			user:    nil,
			token:   "tok",
			wantErr: domain.ErrSessionUserRequired,
		},
		{
			name:    "ng: トークンが空だとセッションを作れない",
			user:    &user,
			token:   "",
			wantErr: domain.ErrEmptySessionToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			session, err := domain.NewSession(tt.user, tt.token, expires)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "tok", session.Token())
		})
	}
}
