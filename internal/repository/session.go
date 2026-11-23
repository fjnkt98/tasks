package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecase"
)

type sessionRepository struct{}

func NewSessionRepository() *sessionRepository {
	return &sessionRepository{}
}

var _ usecase.SessionRepository = &sessionRepository{}

func (r *sessionRepository) Save(ctx context.Context, db usecase.DBTX, session domain.Session) (domain.Session, error) {
	// Insert when no row exists for token; otherwise refresh expiry.
	existing, err := r.FindByToken(ctx, db, session.Token())
	if err != nil && !errors.Is(err, usecase.ErrNotFound) {
		return domain.Session{}, err
	}
	if errors.Is(err, usecase.ErrNotFound) {
		q := "INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?) RETURNING user_id, token, expires_at"
		row := db.QueryRowContext(ctx, q, session.UserID(), session.Token(), session.ExpiresAt().Unix())
		var owner int
		var token string
		var expiresUnix int64
		if scanErr := row.Scan(&owner, &token, &expiresUnix); scanErr != nil {
			return domain.Session{}, fmt.Errorf("execute: %w", scanErr)
		}
		return domain.NewSession(owner, token, time.Unix(expiresUnix, 0).UTC()), nil
	}

	_ = existing
	q := "UPDATE sessions SET expires_at = ? WHERE token = ? RETURNING user_id, token, expires_at"
	row := db.QueryRowContext(ctx, q, session.ExpiresAt().Unix(), session.Token())
	var owner int
	var token string
	var expiresUnix int64
	if err := row.Scan(&owner, &token, &expiresUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, usecase.ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("execute: %w", err)
	}
	return domain.NewSession(owner, token, time.Unix(expiresUnix, 0).UTC()), nil
}

func (r *sessionRepository) FindByToken(ctx context.Context, db usecase.DBTX, token string) (domain.Session, error) {
	q := "SELECT user_id, token, expires_at FROM sessions WHERE token = ?"
	row := db.QueryRowContext(ctx, q, token)
	var owner int
	var sessionToken string
	var expiresUnix int64
	if err := row.Scan(&owner, &sessionToken, &expiresUnix); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, usecase.ErrNotFound
		}
		return domain.Session{}, fmt.Errorf("execute: %w", err)
	}
	return domain.NewSession(owner, sessionToken, time.Unix(expiresUnix, 0).UTC()), nil
}

func (r *sessionRepository) DeleteByToken(ctx context.Context, db usecase.DBTX, token string) (bool, error) {
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

func (r *sessionRepository) DeleteByUserID(ctx context.Context, db usecase.DBTX, userID int) (bool, error) {
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

func (r *sessionRepository) DeleteExpired(ctx context.Context, db usecase.DBTX, now time.Time) (bool, error) {
	q := "DELETE FROM sessions WHERE expires_at <= ?"
	result, err := db.ExecContext(ctx, q, now.Unix())
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected")
	}
	return count > 0, nil
}
