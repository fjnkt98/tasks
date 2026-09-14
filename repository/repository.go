// Package repository provide data operation interface
package repository

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
)

//go:embed migrations/*.sql
var fs embed.FS

func CreateTestDB() (sqlDB *sql.DB, err error) {
	u, err := url.Parse("sqlite3::memory:")
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	db := dbmate.New(u)
	db.AutoDumpSchema = false
	db.FS = fs
	db.MigrationsDir = []string{"migrations"}

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
			err = errors.Join(err, sqlDB.Close())
		}
	}()

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
