package testutil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExtractCookies(t *testing.T, header http.Header) map[string]*http.Cookie {
	t.Helper()

	cookies := make(map[string]*http.Cookie)
	for _, setCookieHeader := range header.Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(setCookieHeader)
		require.NoError(t, err)

		cookies[cookie.Name] = cookie
	}
	return cookies
}
