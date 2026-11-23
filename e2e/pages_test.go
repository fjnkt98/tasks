package e2e_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicPages(t *testing.T) {
	for _, tt := range []struct {
		name         string
		path         string
		wantCode     int
		wantContains string
	}{
		{
			name:         "ok: トップページを開くと、200 になる",
			path:         "/",
			wantCode:     http.StatusOK,
			wantContains: "Go",
		},
		{
			name:         "ok: サインアップページを開くと、フォームが見える",
			path:         "/account/signup",
			wantCode:     http.StatusOK,
			wantContains: "username",
		},
		{
			name:         "ok: サインインページを開くと、フォームが見える",
			path:         "/account/signin",
			wantCode:     http.StatusOK,
			wantContains: "password",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			res, body := get(t, e, tt.path)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Contains(t, body, tt.wantContains)
		})
	}
}

func TestAuthenticatedPages(t *testing.T) {
	for _, tt := range []struct {
		name         string
		path         string
		wantCode     int
		wantContains string
	}{
		{
			name:         "ok: パスワード変更ページを開くと、フォームが見える",
			path:         "/account/password_change",
			wantCode:     http.StatusOK,
			wantContains: "current-password",
		},
		{
			name:         "ok: アカウント削除ページを開くと、確認フォームが見える",
			path:         "/account/delete",
			wantCode:     http.StatusOK,
			wantContains: "confirmation",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			signInAs(t, e, "test")
			res, body := get(t, e, tt.path)
			assert.Equal(t, tt.wantCode, res.StatusCode)
			assert.Contains(t, body, tt.wantContains)
		})
	}
}

func TestAuthRequiredRedirect(t *testing.T) {
	for _, tt := range []struct {
		name     string
		path     string
		wantPath string
	}{
		{
			name:     "ng: 未サインインでタスク一覧を開くと、サインインへリダイレクトされる",
			path:     "/tasks",
			wantPath: "/account/signin",
		},
		{
			name:     "ng: 未サインインでマイページを開くと、サインインへリダイレクトされる",
			path:     "/account/me",
			wantPath: "/account/signin",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			res, _ := get(t, e, tt.path)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.wantPath, res.Request.URL.Path)
		})
	}
}
