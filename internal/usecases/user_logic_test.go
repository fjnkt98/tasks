package usecases_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
)

func TestPrepareGrantRole(t *testing.T) {
	user := mustUser(t, 10, "member", nil)

	tests := []struct {
		name        string
		role        string
		wantHasRole bool
		wantMessage string
	}{
		{
			name:        "ok: ロール付与はユーザーを更新しイベントを返す",
			role:        "admin",
			wantHasRole: true,
			wantMessage: `grant role: "admin"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, event, err := usecases.PrepareGrantRole(user, tt.role)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHasRole, got.HasRole(domain.Role(tt.role)))
			assert.Equal(t, domain.ActivityKindAccount, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
			assert.Equal(t, got.ID(), event.User().ID())
		})
	}
}

func TestPrepareRevokeRole(t *testing.T) {
	admin := mustUser(t, 11, "admin-user", []domain.Role{domain.RoleAdmin})
	member := mustUser(t, 12, "plain", nil)

	tests := []struct {
		name        string
		user        domain.User
		role        string
		wantHasRole bool
		wantEvents  int
		wantMessage string
	}{
		{
			name:        "ok: 保持しているロールを剥奪するとイベントが1件出る",
			user:        admin,
			role:        "admin",
			wantHasRole: false,
			wantEvents:  1,
			wantMessage: `revoke role: "admin"`,
		},
		{
			name:        "ok: 持っていないロールを剥奪してもイベントは出ない",
			user:        member,
			role:        "admin",
			wantHasRole: false,
			wantEvents:  0,
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, events, err := usecases.PrepareRevokeRole(tt.user, tt.role)
			require.NoError(t, err)
			assert.Equal(t, tt.wantHasRole, got.HasRole(domain.Role(tt.role)))
			require.Len(t, events, tt.wantEvents)
			if tt.wantEvents > 0 {
				assert.Equal(t, domain.ActivityKindAccount, events[0].Kind())
				assert.Equal(t, tt.wantMessage, events[0].Message())
			}
		})
	}
}
