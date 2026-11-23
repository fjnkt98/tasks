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

func (r *roleRepository) ListByUser(ctx context.Context, db usecase.DBTX, userID int) ([]domain.Role, error) {
	q := "SELECT role FROM user_role_relations WHERE user_id = ?"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	roles := make([]domain.Role, 0)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		roles = append(roles, domain.Role(role))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return roles, nil
}

func (r *roleRepository) Assign(ctx context.Context, db usecase.DBTX, userID int, role domain.Role) error {
	q := "INSERT INTO user_role_relations (user_id, role) VALUES (?, ?) ON CONFLICT (user_id, role) DO NOTHING"
	if _, err := db.ExecContext(ctx, q, userID, string(role)); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *roleRepository) Revoke(ctx context.Context, db usecase.DBTX, userID int, role domain.Role) (bool, error) {
	q := "DELETE FROM user_role_relations WHERE user_id = ? AND role = ?"
	result, err := db.ExecContext(ctx, q, userID, string(role))
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return count > 0, nil
}

func (r *roleRepository) ListUsers(ctx context.Context, db usecase.DBTX, role domain.Role) ([]domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE id IN (SELECT user_id FROM user_role_relations WHERE role = ?)"
	rows, err := db.QueryContext(ctx, q, string(role))
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var id int
		var name, digest string
		if err := rows.Scan(&id, &name, &digest); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		users = append(users, domain.NewUser(id, name, digest))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return users, nil
}
