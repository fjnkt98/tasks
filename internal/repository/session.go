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

type sessionRepository struct {
	users *userRepository
}

func NewSessionRepository() *sessionRepository {
	return &sessionRepository{
		users: NewUserRepository(),
	}
}

var _ usecase.SessionRepository = &sessionRepository{}

func (r *sessionRepository) Save(ctx context.Context, db usecase.DBTX, session domain.Session) (domain.Session, error) {
	user := session.User()
	if user == nil {
		return domain.Session{}, fmt.Errorf("session user is required")
	}

	q := `
INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)
ON CONFLICT(token) DO UPDATE SET expires_at = excluded.expires_at
RETURNING token, expires_at`
	row := db.QueryRowContext(ctx, q, user.ID(), session.Token(), session.ExpiresAt().Unix())
	var token string
	var expiresUnix int64
	if err := row.Scan(&token, &expiresUnix); err != nil {
		return domain.Session{}, fmt.Errorf("execute: %w", err)
	}
	return domain.NewSession(user, token, time.Unix(expiresUnix, 0).UTC())
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

	user, err := r.users.FindByID(ctx, db, owner)
	if err != nil {
		return domain.Session{}, err
	}
	return domain.NewSession(&user, sessionToken, time.Unix(expiresUnix, 0).UTC())
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
