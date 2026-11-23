package e2e_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsers(t *testing.T) {
	for _, tt := range []struct {
		name         string
		asAdmin      bool
		wantCode     int
		wantPath     string
		wantContains string
	}{
		{
			name:         "ok: 管理者でユーザー一覧を開くと、ユーザー名が見える",
			asAdmin:      true,
			wantCode:     http.StatusOK,
			wantPath:     "/users",
			wantContains: "admin",
		},
		{
			name:         "ng: 一般ユーザーでユーザー一覧を開くと、トップへリダイレクトされる",
			asAdmin:      false,
			wantCode:     http.StatusOK,
			wantPath:     "/",
			wantContains: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.asAdmin {
				setupAdmin(t, e, "admin", "password")
				res := signin(t, e, "admin", "password")
				discard(t, res.Body)
				require.Equal(t, http.StatusOK, res.StatusCode)
			} else {
				signInAs(t, e, "user")
			}

			res, body := get(t, e, "/users")
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			if tt.wantContains != "" {
				assert.Contains(t, body, tt.wantContains)
			}
		})
	}
}

func TestAdminUserShow(t *testing.T) {
	for _, tt := range []struct {
		name            string
		wantContains    []string
		wantActivityMsg string
	}{
		{
			name:            "ok: 管理者でユーザー詳細を開くと、アクティビティが見える",
			wantContains:    []string{"member"},
			wantActivityMsg: "signin",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "member")
			memberID := userID(t, e.db, "member")
			require.Equal(t, 1, activityCount(t, e.db, memberID, "account", tt.wantActivityMsg))

			// New jar/client as admin on same DB/handler.
			setupAdmin(t, e, "admin", "password")
			res := signin(t, e, "admin", "password")
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)

			res, body := get(t, e, fmt.Sprintf("/users/%d", memberID))
			assert.Equal(t, http.StatusOK, res.StatusCode)
			for _, s := range tt.wantContains {
				assert.Contains(t, body, s)
			}
			assert.Contains(t, body, tt.wantActivityMsg)
		})
	}
}
