package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
)

func TestNewUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		uname   string
		digest  string
		wantErr error
	}{
		{
			name:    "ok: 名前とダイジェストがあればユーザーを作れる",
			uname:   "alice",
			digest:  "hash",
			wantErr: nil,
		},
		{
			name:    "ng: ユーザー名が空だと作れない",
			uname:   "  ",
			digest:  "hash",
			wantErr: domain.ErrEmptyUserName,
		},
		{
			name:    "ng: ダイジェストが空だと作れない",
			uname:   "alice",
			digest:  "",
			wantErr: domain.ErrEmptyDigest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			user, err := domain.NewUser(0, tt.uname, tt.digest)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "alice", user.Name())
			assert.Equal(t, "hash", user.Digest())
		})
	}
}

func TestNewPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{
			name:    "ok: 有効なパスワードを作れる",
			value:   "secret",
			wantErr: nil,
		},
		{
			name:    "ng: パスワードが空だと作れない",
			value:   "",
			wantErr: domain.ErrEmptyPassword,
		},
		{
			name:    "ng: 72文字を超えるパスワードは作れない",
			value:   string(make([]byte, domain.MaxPasswordLength+1)),
			wantErr: domain.ErrPasswordTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pw, err := domain.NewPassword(tt.value)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.value, pw.String())
		})
	}
}
