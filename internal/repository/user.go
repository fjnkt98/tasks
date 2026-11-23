package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecase"
)

type userRepository struct{}

var _ usecase.UserRepository = &userRepository{}

func NewUserRepository() *userRepository {
	return &userRepository{}
}

func (r *userRepository) Save(ctx context.Context, db usecase.DBTX, user domain.User) (domain.User, error) {
	if user.IsNew() {
		q := "INSERT INTO users (name, digest) VALUES (?, ?) RETURNING id, name, digest"
		row := db.QueryRowContext(ctx, q, user.Name(), user.Digest())
		var id int
		var name, digest string
		if err := row.Scan(&id, &name, &digest); err != nil {
			if isUniqueConstraintError(err) {
				return domain.User{}, usecase.ErrUniqueConstraint
			}
			return domain.User{}, fmt.Errorf("execute: %w", err)
		}
		return domain.NewUser(id, name, digest), nil
	}

	q := "UPDATE users SET digest = ?, updated_at = UNIXEPOCH() WHERE id = ? RETURNING id, name, digest"
	row := db.QueryRowContext(ctx, q, user.Digest(), user.ID())
	var id int
	var name, digest string
	if err := row.Scan(&id, &name, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, usecase.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}
	return domain.NewUser(id, name, digest), nil
}

func (r *userRepository) FindByID(ctx context.Context, db usecase.DBTX, id int) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE id = ?"
	row := db.QueryRowContext(ctx, q, id)
	var userID int
	var name, digest string
	if err := row.Scan(&userID, &name, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, usecase.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("execute: %w", err)
	}
	return domain.NewUser(userID, name, digest), nil
}

func (r *userRepository) FindByName(ctx context.Context, db usecase.DBTX, name string) (domain.User, error) {
	q := "SELECT id, name, digest FROM users WHERE name = ?"
	row := db.QueryRowContext(ctx, q, name)
	var id int
	var userName, digest string
	if err := row.Scan(&id, &userName, &digest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, usecase.ErrNotFound
		}
		return domain.User{}, err
	}
	return domain.NewUser(id, userName, digest), nil
}

func (r *userRepository) List(ctx context.Context, db usecase.DBTX, limit int, offset int) ([]domain.User, error) {
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

func (r *userRepository) Delete(ctx context.Context, db usecase.DBTX, id int) (bool, error) {
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
