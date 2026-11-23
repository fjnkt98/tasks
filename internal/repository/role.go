package repository

import (
	"context"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecase"
)

type roleRepository struct{}

var _ usecase.RoleRepository = &roleRepository{}

func NewRoleRepository() *roleRepository {
	return &roleRepository{}
}

func (r *roleRepository) Ensure(ctx context.Context, db usecase.DBTX, role domain.Role) (domain.Role, error) {
	q := "INSERT INTO roles (name) VALUES (?) ON CONFLICT (name) DO NOTHING"
	if _, err := db.ExecContext(ctx, q, string(role)); err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	q = "SELECT name FROM roles WHERE name = ?"
	row := db.QueryRowContext(ctx, q, string(role))
	var name string
	if err := row.Scan(&name); err != nil {
		return "", fmt.Errorf("scan row: %w", err)
	}
	return domain.Role(name), nil
}

func (r *roleRepository) List(ctx context.Context, db usecase.DBTX) ([]domain.Role, error) {
	q := "SELECT name FROM roles ORDER BY name ASC"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	roles := make([]domain.Role, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		roles = append(roles, domain.Role(name))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return roles, nil
}

func (r *roleRepository) Delete(ctx context.Context, db usecase.DBTX, role domain.Role) (bool, error) {
	q := "DELETE FROM roles WHERE name = ?"
	result, err := db.ExecContext(ctx, q, string(role))
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return count > 0, nil
}
