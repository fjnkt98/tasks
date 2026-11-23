package sqlite

import (
	"context"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
)

type roleRepository struct{}

var _ domain.RoleRepository = &roleRepository{}

func NewRoleRepository() *roleRepository {
	return &roleRepository{}
}

func (r *roleRepository) Ensure(ctx context.Context, db domain.DBTX, name string) (string, error) {
	q := "INSERT INTO roles (name) VALUES (?) ON CONFLICT (name) DO NOTHING"
	if _, err := db.ExecContext(ctx, q, name); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	q = "SELECT name FROM roles WHERE name = ?"
	row := db.QueryRowContext(ctx, q, name)
	var role string
	if err := row.Scan(&role); err != nil {
		return "", fmt.Errorf("scan row: %w", err)
	}
	return role, nil
}

func (r *roleRepository) Select(ctx context.Context, db domain.DBTX) ([]string, error) {
	q := "SELECT name FROM roles ORDER BY name ASC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return roles, nil
}

func (r *roleRepository) Delete(ctx context.Context, db domain.DBTX, name string) (bool, error) {
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
