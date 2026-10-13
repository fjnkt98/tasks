package repository

import (
	"context"
	"errors"
	"fmt"
)

type Role struct {
	Name string
}

type RoleRepository interface {
	Ensure(ctx context.Context, db DBTX, name string) error
	Select(ctx context.Context, db DBTX) ([]Role, error)
	Delete(ctx context.Context, db DBTX, name string) (bool, error)
}

type roleRepository struct{}

var _ RoleRepository = &roleRepository{}

func NewRoleRepository() *roleRepository {
	return &roleRepository{}
}

func (r *roleRepository) Ensure(ctx context.Context, db DBTX, name string) error {
	q := "INSERT INTO roles (name) VALUES (?) ON CONFLICT (name) DO NOTHING;"
	if _, err := db.ExecContext(ctx, q, name); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *roleRepository) Select(ctx context.Context, db DBTX) (roles []Role, err error) {
	q := "SELECT name FROM roles ORDER BY name ASC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	roles = make([]Role, 0)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.Name); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return roles, nil
}

func (r *roleRepository) Delete(ctx context.Context, db DBTX, name string) (bool, error) {
	q := "DELETE FROM roles WHERE name = ?"
	result, err := db.ExecContext(ctx, q, name)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return count > 0, nil
}
