package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
)

type userRepository struct{}

var _ usecases.UserRepository = &userRepository{}

func NewUserRepository() *userRepository {
	return &userRepository{}
}

func (r *userRepository) loadRoles(ctx context.Context, db usecases.DBTX, userID int) ([]domain.Role, error) {
	q := "SELECT role FROM user_role_relations WHERE user_id = ? ORDER BY role ASC"
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

func (r *userRepository) saveRoles(ctx context.Context, db usecases.DBTX, userID int, roles []domain.Role) error {
	q := "DELETE FROM user_role_relations WHERE user_id = ?"
	if _, err := db.ExecContext(ctx, q, userID); err != nil {
		return fmt.Errorf("delete roles: %w", err)
	}
	q = "INSERT INTO user_role_relations (user_id, role) VALUES (?, ?)"
	for _, role := range roles {
		if _, err := db.ExecContext(ctx, q, userID, string(role)); err != nil {
			return fmt.Errorf("insert role: %w", err)
		}
	}
	return nil
}

func (r *userRepository) hydrate(ctx context.Context, db usecases.DBTX, id int, name, digest string) (domain.User, error) {
	roles, err := r.loadRoles(ctx, db, id)
	if err != nil {
		return domain.User{}, err
	}
	return domain.NewUser(id, name, digest, roles...)
}

func (r *userRepository) Save(ctx context.Context, db usecases.DBTX, user domain.User) (domain.User, error) {
	var id int
	var name, digest string

	if user.IsNew() {
		q := "INSERT INTO users (name, digest) VALUES (?, ?) RETURNING id, name, digest"
		row := db.QueryRowContext(ctx, q, user.Name(), user.Digest())
		if err := row.Scan(&id, &name, &digest); err != nil {
			if isUniqueConstraintError(err) {
				return domain.User{}, usecases.ErrUniqueConstraint
			}
			return domain.User{}, fmt.Errorf("execute: %w", err)
		}
	} else {
		q := "UPDATE users SET digest = ?, updated_at = UNIXEPOCH() WHERE id = ? RETURNING id, name, digest"
		row := db.QueryRowContext(ctx, q, user.Digest(), user.ID())
		if err := row.Scan(&id, &name, &digest); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.User{}, usecases.ErrNotFound
			}
			return domain.User{}, fmt.Errorf("execute: %w", err)
		}
	}

	if err := r.saveRoles(ctx, db, id, user.Roles()); err != nil {
		return domain.User{}, err
	}
	return domain.NewUser(id, name, digest, user.Roles()...)
}

func (r *userRepository) FindByID(ctx context.Context, db usecases.DBTX, id int) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE id = ?"
	row := db.QueryRowContext(ctx, q, id)
	var userID int
	var name, digest string
	if err := row.Scan(&userID, &name, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, usecases.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}
	return r.hydrate(ctx, db, userID, name, digest)
}

func (r *userRepository) FindByName(ctx context.Context, db usecases.DBTX, name string) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE name = ?"
	row := db.QueryRowContext(ctx, q, name)
	var id int
	var userName, digest string
	if err := row.Scan(&id, &userName, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, usecases.ErrNotFound
		}
		return domain.User{}, err
	}
	return r.hydrate(ctx, db, id, userName, digest)
}

func (r *userRepository) List(ctx context.Context, db usecases.DBTX, limit int, offset int) ([]domain.User, error) {
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
		user, err := r.hydrate(ctx, db, id, name, digest)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return users, nil
}

func (r *userRepository) ListByRole(ctx context.Context, db usecases.DBTX, role domain.Role) ([]domain.User, error) {
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
		user, err := r.hydrate(ctx, db, id, name, digest)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return users, nil
}

func (r *userRepository) Delete(ctx context.Context, db usecases.DBTX, id int) (bool, error) {
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
