package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	_ "embed"

	"github.com/XSAM/otelsql"
	"github.com/mattn/go-sqlite3"
)

type DBTX interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ DBTX = &sql.DB{}
var _ DBTX = &sql.Tx{}

//go:embed schema.sql
var Schema []byte

func NewDB(ctx context.Context, dsn string) (db *sql.DB, err error) {
	dsn, err = EnforceForeignKeys(dsn)
	if err != nil {
		return nil, fmt.Errorf("enforce foreign keys: %w", err)
	}

	db, err = otelsql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer func(db *sql.DB) {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}(db)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func EnforceForeignKeys(dsn string) (string, error) {
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

func IsUniqueConstraintError(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}
