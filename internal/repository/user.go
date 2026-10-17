package repository

import (
	"context"
	"errors"
	"fmt"
)

type User struct {
	ID     int
	Name   string
	Digest string
}

type UserRepository interface {
	Create(ctx context.Context, db DBTX, name string, digest []byte) (*User, error)
	CreateOrReplace(ctx context.Context, db DBTX, name string, digest []byte) (*User, error)
	Get(ctx context.Context, db DBTX, id int) (*User, error)
	GetByName(ctx context.Context, db DBTX, name string) (*User, error)
	Select(ctx context.Context, db DBTX, limit int, offset int) ([]User, error)
	UpdateDigest(ctx context.Context, db DBTX, id int, digest []byte) (bool, error)
	Delete(ctx context.Context, db DBTX, id int) (bool, error)
	SelectRoles(ctx context.Context, db DBTX, id int) ([]string, error)
	EnsureRole(ctx context.Context, db DBTX, id int, role string) error
	DeleteRole(ctx context.Context, db DBTX, id int, role string) (bool, error)
	SelectByRoles(ctx context.Context, db DBTX, role string) ([]User, error)
}

type userRepository struct{}

var _ UserRepository = &userRepository{}

func NewUserRepository() *userRepository {
	return &userRepository{}
}

func (r *userRepository) Create(ctx context.Context, db DBTX, name string, digest []byte) (*User, error) {
	q := "INSERT INTO users (name, digest) VALUES (?, ?) RETURNING id, name, digest"
	row := db.QueryRowContext(ctx, q, name, string(digest))

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &user, nil
}

func (r *userRepository) CreateOrReplace(ctx context.Context, db DBTX, name string, digest []byte) (*User, error) {
	q := "INSERT INTO users (name, digest) VALUES (?, ?) ON CONFLICT (name) DO UPDATE SET digest = EXCLUDED.digest RETURNING id, name, digest"
	row := db.QueryRowContext(ctx, q, name, string(digest))

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &user, nil
}

func (r *userRepository) Get(ctx context.Context, db DBTX, id int) (*User, error) {
	q := "SELECT id, name, digest FROM users WHERE id = ?"
	row := db.QueryRowContext(ctx, q, id)

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByName(ctx context.Context, db DBTX, name string) (*User, error) {
	q := "SELECT id, name, digest FROM users WHERE name = ?"
	row := db.QueryRowContext(ctx, q, name)

	var user User
	if err := row.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Select(ctx context.Context, db DBTX, limit int, offset int) (users []User, err error) {
	q := "SELECT id, name, digest FROM users ORDER BY id ASC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	users = make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return users, nil
}

func (r *userRepository) UpdateDigest(ctx context.Context, db DBTX, id int, digest []byte) (bool, error) {
	q := "UPDATE users SET digest = ?, updated_at = UNIXEPOCH() WHERE id = ?"
	result, err := db.ExecContext(ctx, q, string(digest), id)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) Delete(ctx context.Context, db DBTX, id int) (bool, error) {
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

func (r *userRepository) SelectRoles(ctx context.Context, db DBTX, id int) (roles []string, err error) {
	q := "SELECT role FROM user_role_relations WHERE user_id = ?"
	rows, err := db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	roles = make([]string, 0)
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

func (r *userRepository) EnsureRole(ctx context.Context, db DBTX, id int, role string) error {
	q := "INSERT INTO user_role_relations (user_id, role) VALUES (?, ?) ON CONFLICT (user_id, role) DO NOTHING"
	if _, err := db.ExecContext(ctx, q, id, role); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *userRepository) DeleteRole(ctx context.Context, db DBTX, id int, role string) (bool, error) {
	q := "DELETE FROM user_role_relations WHERE user_id = ? AND role = ?"
	result, err := db.ExecContext(ctx, q, id, role)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}

	return count > 0, nil
}

func (r *userRepository) SelectByRoles(ctx context.Context, db DBTX, role string) (users []User, err error) {
	q := "SELECT id, name, digest FROM users WHERE id IN (SELECT user_id FROM user_role_relations WHERE role = ?)"
	rows, err := db.QueryContext(ctx, q, role)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	users = make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Digest); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}

		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return users, nil
}
