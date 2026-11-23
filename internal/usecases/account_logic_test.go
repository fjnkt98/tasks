package usecases_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
)

func TestEventSignin(t *testing.T) {
	user := mustUser(t, 1, "alice", nil)

	tests := []struct {
		name        string
		wantKind    string
		wantMessage string
	}{
		{
			name:        "ok: サインインイベントはaccount kindである",
			wantKind:    domain.ActivityKindAccount,
			wantMessage: "signin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventSignin(&user)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
			assert.Equal(t, user.ID(), event.User().ID())
		})
	}
}

func TestEventChangePassword(t *testing.T) {
	user := mustUser(t, 2, "bob", nil)

	tests := []struct {
		name        string
		wantMessage string
	}{
		{
			name:        "ok: パスワード変更イベントのメッセージは固定文言である",
			wantMessage: "change password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventChangePassword(&user)
			require.NoError(t, err)
			assert.Equal(t, domain.ActivityKindAccount, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
		})
	}
}

func TestEventSignout(t *testing.T) {
	user := mustUser(t, 3, "carol", nil)

	tests := []struct {
		name        string
		wantMessage string
	}{
		{
			name:        "ok: サインアウトイベントのメッセージは固定文言である",
			wantMessage: "signout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := usecases.EventSignout(&user)
			require.NoError(t, err)
			assert.Equal(t, domain.ActivityKindAccount, event.Kind())
			assert.Equal(t, tt.wantMessage, event.Message())
		})
	}
}
