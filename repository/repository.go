// Package repository provide data operation interface
package repository

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/XSAM/otelsql"
	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
)

//go:embed migrations/*.sql
var fs embed.FS

func NewDB(dsn string) (*sql.DB, error) {
	db, err := otelsql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := applyPragma(db); err != nil {
		err = errors.Join(err, db.Close())
		return nil, fmt.Errorf("apply pragma: %w", err)
	}

	return db, nil
}

func applyPragma(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA foreign_keys = true"); err != nil {
		return fmt.Errorf("activate foreign keys constraint: %w", err)
	}
	return nil
}

func newTestDB() (db *dbmate.DB, err error) {
	u, err := url.Parse("sqlite3::memory:")
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	db = dbmate.New(u)
	db.AutoDumpSchema = false
	db.FS = fs
	db.MigrationsDir = []string{"migrations"}

	return db, nil
}

func applyMigrations(db *dbmate.DB) (sqlDB *sql.DB, err error) {
	drv, err := db.Driver()
	if err != nil {
		return nil, fmt.Errorf("get driver: %w", err)
	}
	sqlDB, err = drv.Open()
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err != nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	if _, err := sqlDB.Exec("PRAGMA foreign_keys = true"); err != nil {
		return nil, fmt.Errorf("activate foreign keys constraint: %w", err)
	}

	if err := drv.CreateMigrationsTable(sqlDB); err != nil {
		return nil, fmt.Errorf("create migration table: %w", err)
	}

	migrations, err := db.FindMigrations()
	if err != nil {
		return nil, fmt.Errorf("find migrations: %w", err)
	}

	for _, migration := range migrations {
		parsed, err := migration.Parse()
		if err != nil {
			return nil, fmt.Errorf("parse migration: %w", err)
		}

		for _, section := range parsed {
			if _, err := sqlDB.Exec(section.Up); err != nil {
				return nil, fmt.Errorf("exec migration: %w", err)
			}
		}
	}

	return sqlDB, nil
}

func NewTestDB() (*sql.DB, error) {
	db, err := newTestDB()
	if err != nil {
		return nil, fmt.Errorf("create test db: %w", err)
	}

	sqlDB, err := applyMigrations(db)
	if err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	if err := applyPragma(sqlDB); err != nil {
		err = errors.Join(err, sqlDB.Close())
		return nil, fmt.Errorf("apply pragma: %w", err)
	}

	return sqlDB, nil
}
