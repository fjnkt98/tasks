package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	client := NewTestDB(t)

	_, err := NewServer(8000, client)
	assert.NoError(t, err)
}

func TestNewHandler(t *testing.T) {
	client := NewTestDB(t)
	h := newHandler(client)

	t.Run("index", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestCrossOriginProtection(t *testing.T) {
	client := NewTestDB(t)
	h := newHandler(client)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com/signout", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
