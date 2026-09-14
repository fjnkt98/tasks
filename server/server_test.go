package server

import (
	"testing"

	"github.com/fjnkt98/tasks/repository"
)

func TestNewServer(t *testing.T) {
	db, err := repository.CreateTestDB()
	if err != nil {
		t.Fatalf("failed to create test db: %s", err)
	}
	defer db.Close() // nolint:errcheck

	_, err = NewServer(8000, db)
	if err != nil {
		t.Errorf("expected nil, but got %v", err)
	}
}
