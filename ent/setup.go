package ent

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	"github.com/fjnkt98/tasks/ent/migrate"
	_ "github.com/mattn/go-sqlite3"
)

func SetupClient(ctx context.Context, dsn string) (client *Client, err error) {
	dsn, err = enforceForeignKeys(dsn)
	if err != nil {
		return nil, fmt.Errorf("enforce foreign keys: %w", err)
	}

	db, err := otelsql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	drv := sql.OpenDB(dialect.SQLite, db)
	client = NewClient(Driver(drv))

	if err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return client, nil
}

func enforceForeignKeys(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}

	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", fmt.Errorf("parse query parameters: %w", err)
	}

	q.Del("_foreign_keys")
	q.Set("_fk", "1")
	u.RawQuery = q.Encode()

	return u.String(), nil
}
