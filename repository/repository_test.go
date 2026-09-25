package repository

import (
	"slices"
	"testing"
)

func TestCreateTestDB(t *testing.T) {
	db, err := NewTestDB()
	if err != nil {
		t.Fatalf("failed create test database: %s", err)
	}
	defer db.Close() // nolint:errcheck
}

func TestMigrations(t *testing.T) {
	db, err := newTestDB()
	if err != nil {
		t.Fatalf("failed to create test database: %s", err)
	}

	sqlDB, err := applyMigrations(db)
	if err != nil {
		t.Fatalf("failed to apply migrations: %s", err)
	}
	defer sqlDB.Close() // nolint:errcheck

	// rollback test
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

	// rollback verification
	row := sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE sql IS NOT NULL AND name != 'schema_migrations'")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("failed to scan schema: %s", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, but got %d", count)
	}
}
