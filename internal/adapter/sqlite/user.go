package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/repository"
)

type userRepository struct{}

var _ repository.UserRepository = &userRepository{}

func NewUserRepository() *userRepository {
	return &userRepository{}
}

func (r *userRepository) Create(ctx context.Context, db repository.DBTX, name string, digest []byte) (domain.User, error) {
	q := "INSERT INTO users (name, digest) VALUES (?, ?) RETURNING id, name, digest"
	row := db.QueryRowContext(ctx, q, name, string(digest))

	var id int
	var userName, userDigest string
	if err := row.Scan(&id, &userName, &userDigest); err != nil {
		if isUniqueConstraintError(err) {
			return domain.User{}, repository.ErrUniqueConstraint
		}
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}

	return domain.NewUser(id, userName, userDigest), nil
}

func (r *userRepository) CreateOrReplace(ctx context.Context, db repository.DBTX, name string, digest []byte) (domain.User, error) {
	q := "INSERT INTO users (name, digest) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET digest = EXCLUDED.digest RETURNING id, name, digest"
	row := db.QueryRowContext(ctx, q, name, string(digest))

	var id int
	var userName, userDigest string
	if err := row.Scan(&id, &userName, &userDigest); err != nil {
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}

	return domain.NewUser(id, userName, userDigest), nil
}

func (r *userRepository) Get(ctx context.Context, db repository.DBTX, id int) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE id = ?"
	row := db.QueryRowContext(ctx, q, id)

	var userID int
	var name, digest string
	if err := row.Scan(&userID, &name, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, repository.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}

	return domain.NewUser(userID, name, digest), nil
}

func (r *userRepository) GetByName(ctx context.Context, db repository.DBTX, name string) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE name = ?"
	row := db.QueryRowContext(ctx, q, name)

	var id int
	var userName, digest string
	if err := row.Scan(&id, &userName, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, repository.ErrNotFound
		}
		return domain.User{}, err
	}

	return domain.NewUser(id, userName, digest), nil
}

func (r *userRepository) Select(ctx context.Context, db repository.DBTX, limit int, offset int) ([]domain.User, error) {
	q := "SELECT id, name, digest FROM users ORDER BY id ASC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, limit, offset)
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

func (r *userRepository) UpdateDigest(ctx context.Context, db repository.DBTX, user domain.User) (bool, error) {
	q := "UPDATE users SET digest = ?, updated_at = UNIXEPOCH() WHERE id = ?"
	result, err := db.ExecContext(ctx, q, user.Digest(), user.ID())
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) Delete(ctx context.Context, db repository.DBTX, id int) (bool, error) {
	q := "DELETE FROM users WHERE id = ?"
	result, err := db.ExecContext(ctx, q, id)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) SelectRoles(ctx context.Context, db repository.DBTX, id int) ([]domain.Role, error) {
	q := "SELECT role FROM user_role_relations WHERE user_id = ?"
	rows, err := db.QueryContext(ctx, q, id)
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

func (r *userRepository) EnsureRole(ctx context.Context, db repository.DBTX, id int, role domain.Role) error {
	q := "INSERT INTO user_role_relations (user_id, role) VALUES (?, ?) ON CONFLICT (user_id, role) DO NOTHING"
	if _, err := db.ExecContext(ctx, q, id, string(role)); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *userRepository) DeleteRole(ctx context.Context, db repository.DBTX, id int, role domain.Role) (bool, error) {
	q := "DELETE FROM user_role_relations WHERE user_id = ? AND role = ?"
	result, err := db.ExecContext(ctx, q, id, string(role))
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) SelectByRoles(ctx context.Context, db repository.DBTX, role domain.Role) ([]domain.User, error) {
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
