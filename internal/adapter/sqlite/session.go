package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

type sessionRepository struct{}

func NewSessionRepository() *sessionRepository {
	return &sessionRepository{}
}

var _ domain.SessionRepository = &sessionRepository{}

func (r *sessionRepository) Create(ctx context.Context, db domain.DBTX, userID int, digest string, token string, expiresAt time.Time) (*domain.Session, error) {
	q := "INSERT INTO sessions (user_id, token, expires_at) SELECT id, ?, ? FROM users WHERE id = ? AND digest = ? RETURNING user_id, token, expires_at"
	row := db.QueryRowContext(ctx, q, token, expiresAt.Unix(), userID, digest)

	var session domain.Session
	if err := row.Scan(&session.UserID, &session.Token, &session.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) GetByToken(ctx context.Context, db domain.DBTX, token string) (*domain.Session, error) {
	q := "SELECT user_id, token, expires_at FROM sessions WHERE token = ?"
	row := db.QueryRowContext(ctx, q, token)

	var session domain.Session
	if err := row.Scan(&session.UserID, &session.Token, &session.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) UpdateByToken(ctx context.Context, db domain.DBTX, token string, expiresAt time.Time) (bool, error) {
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

func (r *sessionRepository) DeleteByUserID(ctx context.Context, db domain.DBTX, userID int) (bool, error) {
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

func (r *sessionRepository) DeleteByToken(ctx context.Context, db domain.DBTX, token string) (bool, error) {
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

func (r *sessionRepository) DeleteExpired(ctx context.Context, db domain.DBTX) (bool, error) {
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
