package usecases

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/repository"
)

type User struct {
	ID    int
	Name  string
	Roles []string
}

type SessionUsecase interface {
	Continue(ctx context.Context, token string) (*User, error)
}

type sessionUsecase struct {
	db       *sql.DB
	users    repository.UserRepository
	sessions repository.SessionRepository
}

var _ SessionUsecase = &sessionUsecase{}

func NewSessionUsecase(db *sql.DB) *sessionUsecase {
	return &sessionUsecase{
		db:       db,
		users:    repository.NewUserRepository(),
		sessions: repository.NewSessionRepository(),
	}
}

func (u *sessionUsecase) Continue(ctx context.Context, token string) (*User, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	// get session
	session, err := u.sessions.GetByToken(ctx, tx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get session by token: %w", err)
	}

	// session expiration check
	if session.ExpiresAt <= time.Now().Unix() {
		if _, err := u.sessions.DeleteByToken(ctx, tx, session.Token); err != nil {
			return nil, fmt.Errorf("delete session by token: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}

		return nil, nil
	}

	// user identification
	user, err := u.users.GetByID(ctx, tx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	roles, err := u.users.SelectRoles(ctx, tx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("select roles: %w", err)
	}

	// extend expiration
	ok, err := u.sessions.UpdateByToken(ctx, tx, session.Token, time.Now().Add(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("update session by token: %w", err)
	}

	if !ok {
		return nil, nil
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &User{
		ID:    user.ID,
		Name:  user.Name,
		Roles: roleNames,
	}, nil
}
