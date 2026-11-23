package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountSignupAndSignin(t *testing.T) {
	for _, tt := range []struct {
		name           string
		signupPassword string
		signinPassword string
		wantStatus     int
		wantPath       string
		wantCookie     bool
		wantSessions   int
		wantSigninAct  bool
	}{
		{
			name:           "ok: 正しいパスワードを入力すると、サインインに成功する",
			signupPassword: " contains whitespace ",
			signinPassword: " contains whitespace ",
			wantStatus:     http.StatusOK,
			wantPath:       "/tasks",
			wantCookie:     true,
			wantSessions:   1,
			wantSigninAct:  true,
		},
		{
			name:           "ng: トリム後のパスワードでサインインすると、認証に失敗する",
			signupPassword: " contains whitespace ",
			signinPassword: "contains whitespace",
			wantStatus:     http.StatusBadRequest,
			wantPath:       "/account/signin",
			wantCookie:     false,
			wantSessions:   0,
			wantSigninAct:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, "test", tt.signupPassword)
			assert.Equal(t, 1, userCount(t, e.db, "test"))
			uid := userID(t, e.db, "test")

			res := signin(t, e, "test", tt.signinPassword)
			discard(t, res.Body)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)

			cookie := authCookie(t, e)
			if tt.wantCookie {
				require.NotNil(t, cookie)
				assert.NotEmpty(t, cookie.Value)
			} else {
				assert.Nil(t, cookie)
			}
			assert.Equal(t, tt.wantSessions, sessionCount(t, e.db))

			wantActs := 0
			if tt.wantSigninAct {
				wantActs = 1
			}
			assert.Equal(t, wantActs, activityCount(t, e.db, uid, "account", "signin"))
		})
	}
}

func TestAccountSignupLimits(t *testing.T) {
	for _, tt := range []struct {
		name       string
		username   string
		password   string
		wantStatus int
		wantUsers  int
	}{
		{
			name:       "ng: 過大なリクエストボディを送ると、413 になる",
			username:   strings.Repeat("a", 1<<20),
			password:   "test",
			wantStatus: http.StatusRequestEntityTooLarge,
			wantUsers:  0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			res := postForm(t, e, "/account/signup", url.Values{
				"username":         []string{tt.username},
				"password":         []string{tt.password},
				"confirm-password": []string{tt.password},
			})
			discard(t, res.Body)
			assert.Equal(t, tt.wantStatus, res.StatusCode)
			assert.Equal(t, tt.wantUsers, userCount(t, e.db, tt.username))
		})
	}
}

func TestAccountPasswordChange(t *testing.T) {
	for _, tt := range []struct {
		name        string
		username    string
		oldPassword string
		newPassword string
		wantPath    string
	}{
		{
			name:        "ok: パスワードを変更すると、新しいパスワードでサインインできる",
			username:    "test",
			oldPassword: "password",
			newPassword: "new-password",
			wantPath:    "/account/password_change/success",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, tt.username, tt.oldPassword)

			res := signin(t, e, tt.username, tt.oldPassword)
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)
			require.Equal(t, "/tasks", res.Request.URL.Path)
			uid := userID(t, e.db, tt.username)

			res = postForm(t, e, "/account/password_change", url.Values{
				"current-password":     []string{tt.oldPassword},
				"new-password":         []string{tt.newPassword},
				"confirm-new-password": []string{tt.newPassword},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			assert.Nil(t, authCookie(t, e))
			assert.Equal(t, 0, sessionCount(t, e.db))
			assert.Equal(t, 1, activityCount(t, e.db, uid, "account", "change password"))

			res = signin(t, e, tt.username, tt.newPassword)
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, "/tasks", res.Request.URL.Path)

			cookie := authCookie(t, e)
			require.NotNil(t, cookie)
			assert.NotEmpty(t, cookie.Value)
			assert.Equal(t, 1, sessionCount(t, e.db))
		})
	}
}

func TestAccountDelete(t *testing.T) {
	for _, tt := range []struct {
		name         string
		username     string
		password     string
		confirmation string
		wantPath     string
	}{
		{
			name:         "ok: アカウントを削除すると、ユーザー行が消える",
			username:     "test",
			password:     "password",
			confirmation: "delete account",
			wantPath:     "/account/delete/success",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signup(t, e, tt.username, tt.password)
			require.Equal(t, 1, userCount(t, e.db, tt.username))

			res := signin(t, e, tt.username, tt.password)
			discard(t, res.Body)
			require.Equal(t, http.StatusOK, res.StatusCode)

			res = postForm(t, e, "/account/delete", url.Values{
				"password":     []string{tt.password},
				"confirmation": []string{tt.confirmation},
			})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
			assert.Equal(t, 0, userCount(t, e.db, tt.username))
			assert.Equal(t, 0, sessionCount(t, e.db))
		})
	}
}

func TestAccountSignout(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ok: サインアウトすると、セッションが消えアクティビティが残る"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			uid := userID(t, e.db, "test")
			require.Equal(t, 1, sessionCount(t, e.db))

			res := postForm(t, e, "/account/signout", url.Values{})
			discard(t, res.Body)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Nil(t, authCookie(t, e))
			assert.Equal(t, 0, sessionCount(t, e.db))
			assert.Equal(t, 1, activityCount(t, e.db, uid, "account", "signout"))
		})
	}
}

func TestAccountMe(t *testing.T) {
	for _, tt := range []struct {
		name string
	}{
		{name: "ok: マイページを開くと、アカウントとタスクのアクティビティが見える"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			uid := userID(t, e.db, "test")
			id := createTask(t, e, "buy milk")
			require.Equal(t, 1, activityCount(t, e.db, uid, "account", "signin"))
			addMsg := fmt.Sprintf("add task %d, title: %q", id, "buy milk")
			require.Equal(t, 1, activityCount(t, e.db, uid, "task", addMsg))

			res, body := get(t, e, "/account/me")
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Contains(t, body, "test")
			assert.Contains(t, body, "signin")
			assert.Contains(t, body, fmt.Sprintf("add task %d, title:", id))
			assert.Contains(t, body, "buy milk")
		})
	}
}
