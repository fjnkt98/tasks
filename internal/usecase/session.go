package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

// CurrentSession is the authenticated principal for a request (not the persisted domain.Session).
type CurrentSession struct {
	UserID   int
	UserName string
	Roles    []domain.Role
}

var ErrSessionInvalid = errors.New("session invalid")

type SessionUsecase interface {
	Continue(ctx context.Context, token string) (*CurrentSession, error)
}

type sessionUsecase struct {
	db       *sql.DB
	users    UserRepository
	sessions SessionRepository
	roles    RoleRepository
}

var _ SessionUsecase = &sessionUsecase{}

func NewSessionUsecase(db *sql.DB, users UserRepository, sessions SessionRepository, roles RoleRepository) *sessionUsecase {
	return &sessionUsecase{
		db:       db,
		users:    users,
		sessions: sessions,
		roles:    roles,
	}
}

func (u *sessionUsecase) Continue(ctx context.Context, token string) (*CurrentSession, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	session, err := u.sessions.FindByToken(ctx, tx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrSessionInvalid
		}
		return nil, fmt.Errorf("find session by token: %w", err)
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

	user, err := u.users.FindByID(ctx, tx, session.UserID())
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	roles, err := u.roles.ListByUser(ctx, tx, session.UserID())
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	session = session.WithExpiresAt(now.Add(24 * time.Hour))
	if _, err := u.sessions.Save(ctx, tx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &CurrentSession{
		UserID:   user.ID(),
		UserName: user.Name(),
		Roles:    roles,
	}, nil
}
