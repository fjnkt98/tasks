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
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "tok", session.Token())
			assert.Equal(t, &user, session.User())
			assert.True(t, session.ExpiresAt().Equal(expires))
		})
	}
}

func TestSessionExpired(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "ok: 期限より前なら有効である",
			expiresAt: now.Add(time.Minute),
			want:      false,
		},
		{
			name:      "ok: 期限ちょうどなら期限切れである",
			expiresAt: now,
			want:      true,
		},
		{
			name:      "ok: 期限より後なら期限切れである",
			expiresAt: now.Add(-time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			session, err := domain.NewSession(&user, "tok", tt.expiresAt)
			require.NoError(t, err)
			assert.Equal(t, tt.want, session.Expired(now))
		})
	}
}

func TestSessionWithExpiresAt(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "digest")
	require.NoError(t, err)
	old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	next := old.Add(24 * time.Hour)

	session, err := domain.NewSession(&user, "tok", old)
	require.NoError(t, err)

	got := session.WithExpiresAt(next)
	assert.True(t, got.ExpiresAt().Equal(next))
	assert.True(t, session.ExpiresAt().Equal(old))
}
