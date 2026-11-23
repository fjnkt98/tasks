package e2e_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/controllers"
)

func setAuthCookie(t *testing.T, e *env, value string) {
	t.Helper()

	u, err := url.Parse(e.server.URL)
	require.NoError(t, err)
	e.client.Jar.SetCookies(u, []*http.Cookie{{
		Name:  controllers.AuthCookieName,
		Value: value,
		Path:  "/",
	}})
}

func TestSessionInvalidation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		prepare func(t *testing.T, e *env)
	}{
		{
			name: "ng: 未知のセッショントークンだと、未ログイン扱いになる",
			prepare: func(t *testing.T, e *env) {
				t.Helper()
				setAuthCookie(t, e, "not-a-real-session-token")
			},
		},
		{
			name: "ng: 期限切れセッションだと、セッションが削除されて未ログインになる",
			prepare: func(t *testing.T, e *env) {
				t.Helper()
				signInAs(t, e, "test")
				require.NotNil(t, authCookie(t, e))
				require.Equal(t, 1, sessionCount(t, e.db))
				_, err := e.db.Exec(`UPDATE sessions SET expires_at = 0`)
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			tt.prepare(t, e)

			res, _ := get(t, e, "/tasks")
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/account/signin", res.Request.URL.Path)
			assert.Equal(t, 0, sessionCount(t, e.db))
			assert.Nil(t, authCookie(t, e))
		})
	}
}
