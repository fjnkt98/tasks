package repository

import (
	"database/sql"
	"errors"
	"testing"
)

func TestCreateTestDB(t *testing.T) {
	db, err := CreateTestDB()
	if err != nil {
		t.Fatalf("failed create test database: %s", err)
	}
	defer db.Close() // nolint:errcheck

	rows, err := db.Query("SELECT * FROM tasks;")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("got error %+v", err)
	}
	defer rows.Close() //nolint:errcheck
}
