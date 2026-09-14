package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fjnkt98/tasks/server"
)

func TestGetIndex(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := &server.IndexHandler{}
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("expected status see other, but got %d", rec.Code)
	}
}
