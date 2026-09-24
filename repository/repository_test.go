package repository

import (
	"database/sql"
	"errors"
	"slices"
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

func TestMigrations(t *testing.T) {
	db, err := createTestDB()
	if err != nil {
		t.Fatalf("failed to create test database: %s", err)
	}

	sqlDB, err := applyMigrations(db)
	if err != nil {
		t.Fatalf("failed to apply migrations: %s", err)
	}
	defer sqlDB.Close() // nolint:errcheck

	migrations, err := db.FindMigrations()
	if err != nil {
		t.Fatalf("failed to find migrations: %s", err)
	}

	for _, migration := range slices.Backward(migrations) {
		parsed, err := migration.Parse()
		if err != nil {
			t.Fatalf("failed to parse migration: %s", err)
		}

		for _, section := range parsed {
			if _, err := sqlDB.Exec(section.Down); err != nil {
				t.Fatalf("failed to execute down migration: %s", err)
			}
		}
	}
}
