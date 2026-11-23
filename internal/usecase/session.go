package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/repository"
)

type Session struct {
	UserID   int
	UserName string
	Roles    []string
}

// ErrSessionInvalid means the token is missing, expired, or no longer usable.
var ErrSessionInvalid = errors.New("session invalid")

type SessionUsecase interface {
	Continue(ctx context.Context, token string) (*Session, error)
}

type sessionUsecase struct {
	db       *sql.DB
	users    repository.UserRepository
	sessions repository.SessionRepository
}

var _ SessionUsecase = &sessionUsecase{}

func NewSessionUsecase(db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository) *sessionUsecase {
	return &sessionUsecase{
		db:       db,
		users:    users,
		sessions: sessions,
	}
}

func (u *sessionUsecase) Continue(ctx context.Context, token string) (*Session, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	session, err := u.sessions.GetByToken(ctx, tx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrSessionInvalid
		}

		return nil, fmt.Errorf("get session by token: %w", err)
	}

	now := time.Now()
	if session.Expired(now) {
		if _, err := u.sessions.DeleteByToken(ctx, tx, session.Token()); err != nil {
			return nil, fmt.Errorf("delete session by token: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		return nil, ErrSessionInvalid
	}

	user, err := u.users.Get(ctx, tx, session.UserID())
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, err := u.users.SelectRoles(ctx, tx, session.UserID())
	if err != nil {
		return nil, fmt.Errorf("select roles: %w", err)
	}

	session = session.WithExpiresAt(now.Add(24 * time.Hour))
	ok, err := u.sessions.UpdateByToken(ctx, tx, session)
	if err != nil {
		return nil, fmt.Errorf("update session by token: %w", err)
	}

	if !ok {
		return nil, ErrSessionInvalid
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = string(role)
	}

	return &Session{
		UserID:   user.ID(),
		UserName: user.Name(),
		Roles:    roleNames,
	}, nil
}
