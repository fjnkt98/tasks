package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Session struct {
	UserID    int
	Token     string
	ExpiresAt int64
}

type SessionRepository interface {
	Create(ctx context.Context, db DBTX, userID int, digest string, token string, expiresAt time.Time) (*Session, error)
	GetByToken(ctx context.Context, db DBTX, token string) (*Session, error)
	UpdateByToken(ctx context.Context, db DBTX, token string, expiresAt time.Time) (bool, error)
	DeleteByUserID(ctx context.Context, db DBTX, userID int) (bool, error)
	DeleteByToken(ctx context.Context, db DBTX, token string) (bool, error)
	DeleteExpired(ctx context.Context, db DBTX) (bool, error)
}

type sessionRepository struct{}

func NewSessionRepository() *sessionRepository {
	return &sessionRepository{}
}

var _ SessionRepository = &sessionRepository{}

func (r *sessionRepository) Create(ctx context.Context, db DBTX, userID int, digest string, token string, expiresAt time.Time) (*Session, error) {
	q := "INSERT INTO sessions (user_id, token, expires_at) SELECT id, ?, ? FROM users WHERE id = ? AND digest = ? RETURNING user_id, token, expires_at"
	row := db.QueryRowContext(ctx, q, token, expiresAt.Unix(), userID, digest)

	var session Session
	if err := row.Scan(&session.UserID, &session.Token, &session.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) GetByToken(ctx context.Context, db DBTX, token string) (*Session, error) {
	q := "SELECT user_id, token, expires_at FROM sessions WHERE token = ?"
	row := db.QueryRowContext(ctx, q, token)

	var session Session
	if err := row.Scan(&session.UserID, &session.Token, &session.ExpiresAt); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) UpdateByToken(ctx context.Context, db DBTX, token string, expiresAt time.Time) (bool, error) {
	q := "UPDATE sessions SET expires_at = ? WHERE token = ?"
	result, err := db.ExecContext(ctx, q, expiresAt.Unix(), token)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected")
	}

	return count > 0, nil
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, db DBTX, userID int) (bool, error) {
	q := "DELETE FROM sessions WHERE user_id = ?"
	result, err := db.ExecContext(ctx, q, userID)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected")
	}

	return count > 0, nil
}

func (r *sessionRepository) DeleteByToken(ctx context.Context, db DBTX, token string) (bool, error) {
	q := "DELETE FROM sessions WHERE token = ?"
	result, err := db.ExecContext(ctx, q, token)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected")
	}

	return count > 0, nil
}

func (r *sessionRepository) DeleteExpired(ctx context.Context, db DBTX) (bool, error) {
	q := "DELETE FROM sessions WHERE expires_at <= ?"
	result, err := db.ExecContext(ctx, q, time.Now().Unix())
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected")
	}

	return count > 0, nil
}
