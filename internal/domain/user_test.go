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
		name     string
		id       int
		uname    string
		digest   string
		roles    []domain.Role
		wantName string
		wantErr  error
	}{
		{
			name:     "ok: 名前とダイジェストがあればユーザーを作れる",
			id:       0,
			uname:    "alice",
			digest:   "hash",
			roles:    nil,
			wantName: "alice",
			wantErr:  nil,
		},
		{
			name:     "ok: 役割付きでユーザーを作れる",
			id:       1,
			uname:    "bob",
			digest:   "hash",
			roles:    []domain.Role{domain.RoleAdmin},
			wantName: "bob",
			wantErr:  nil,
		},
		{
			name:     "ok: 前後の空白はユーザー名から取り除かれる",
			id:       0,
			uname:    "  alice  ",
			digest:   "hash",
			roles:    nil,
			wantName: "alice",
			wantErr:  nil,
		},
		{
			name:     "ng: ユーザー名が空だと作れない",
			id:       0,
			uname:    "  ",
			digest:   "hash",
			roles:    nil,
			wantName: "",
			wantErr:  domain.ErrEmptyUserName,
		},
		{
			name:     "ng: ダイジェストが空だと作れない",
			id:       0,
			uname:    "alice",
			digest:   "",
			roles:    nil,
			wantName: "",
			wantErr:  domain.ErrEmptyDigest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			user, err := domain.NewUser(tt.id, tt.uname, tt.digest, tt.roles...)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, user.ID())
			assert.Equal(t, tt.wantName, user.Name())
			assert.Equal(t, tt.digest, user.Digest())
			assert.Equal(t, tt.roles, user.Roles())
			assert.Equal(t, tt.id == 0, user.IsNew())
		})
	}
}

func TestUserWithDigest(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "old")
	require.NoError(t, err)

	tests := []struct {
		name    string
		digest  string
		want    string
		wantErr error
	}{
		{
			name:    "ok: ダイジェストを更新できる",
			digest:  "new",
			want:    "new",
			wantErr: nil,
		},
		{
			name:    "ng: 空のダイジェストには更新できない",
			digest:  "",
			want:    "",
			wantErr: domain.ErrEmptyDigest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := user.WithDigest(tt.digest)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Digest())
			assert.Equal(t, "old", user.Digest())
		})
	}
}

func TestUserRoles(t *testing.T) {
	t.Parallel()

	user, err := domain.NewUser(1, "alice", "hash")
	require.NoError(t, err)

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "ok: 役割を付与すると HasRole が真になる",
			run: func(t *testing.T) {
				t.Helper()
				got := user.GrantRole(domain.RoleAdmin)
				assert.True(t, got.HasRole(domain.RoleAdmin))
				assert.Equal(t, []domain.Role{domain.RoleAdmin}, got.Roles())
				assert.False(t, user.HasRole(domain.RoleAdmin))
			},
		},
		{
			name: "ok: 同じ役割を再度付与しても重複しない",
			run: func(t *testing.T) {
				t.Helper()
				got := user.GrantRole(domain.RoleAdmin).GrantRole(domain.RoleAdmin)
				assert.Equal(t, []domain.Role{domain.RoleAdmin}, got.Roles())
			},
		},
		{
			name: "ok: 役割を剥奪すると HasRole が偽になる",
			run: func(t *testing.T) {
				t.Helper()
				got := user.GrantRole(domain.RoleAdmin).RevokeRole(domain.RoleAdmin)
				assert.False(t, got.HasRole(domain.RoleAdmin))
				assert.Empty(t, got.Roles())
			},
		},
		{
			name: "ok: 持っていない役割を剥奪しても変化しない",
			run: func(t *testing.T) {
				t.Helper()
				got := user.RevokeRole(domain.RoleAdmin)
				assert.Empty(t, got.Roles())
			},
		},
		{
			name: "ok: Roles の戻り値を書き換えても元のユーザーは変わらない",
			run: func(t *testing.T) {
				t.Helper()
				got := user.GrantRole(domain.RoleAdmin)
				roles := got.Roles()
				roles[0] = domain.Role("mutated")
				assert.Equal(t, []domain.Role{domain.RoleAdmin}, got.Roles())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
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
			name:    "ok: 72文字ちょうどのパスワードを作れる",
			value:   string(make([]byte, domain.MaxPasswordLength)),
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
