package usecase

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserAlreadyExists = errors.New("user already exists")

func NewSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type AccountUsecase interface {
	Signup(ctx context.Context, username, password string) error
	Signin(ctx context.Context, username, password string) (string, error)
	ChangePassword(ctx context.Context, userID int, password string, newPassword string) error
	Signout(ctx context.Context, token string) error
	Delete(ctx context.Context, userID int, password string) error
}

type accountUsecase struct {
	db         *sql.DB
	users      repository.UserRepository
	sessions   repository.SessionRepository
	roles      repository.RoleRepository
	activities repository.ActivityRepository
}

var _ AccountUsecase = &accountUsecase{}

func NewAccountUsecase(db *sql.DB, users repository.UserRepository, sessions repository.SessionRepository, roles repository.RoleRepository, activities repository.ActivityRepository) *accountUsecase {
	return &accountUsecase{
		db:         db,
		users:      users,
		sessions:   sessions,
		roles:      roles,
		activities: activities,
	}
}

func (u *accountUsecase) Signup(ctx context.Context, username string, password string) error {
	digest, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	if _, err = u.users.Create(ctx, tx, username, digest); err != nil {
		if errors.Is(err, repository.ErrUniqueConstraint) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *accountUsecase) Signin(ctx context.Context, username string, password string) (string, error) {
	user, err := u.users.GetByName(ctx, u.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Digest()), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	session, err := u.sessions.Create(ctx, tx, user.ID(), user.Digest(), NewSessionToken(), time.Now().Add(24*time.Hour))
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("create session: %w", err)
	}

	activity := domain.NewActivity(user.ID(), "account", "signin", time.Time{})
	if err := u.activities.Create(ctx, tx, activity); err != nil {
		return "", fmt.Errorf("create activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return session.Token(), nil
}

func (u *accountUsecase) ChangePassword(ctx context.Context, userID int, password string, newPassword string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	user, err := u.users.Get(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("get user by id: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Digest()), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest for new password: %w", err)
	}

	user = user.WithDigest(string(digest))
	ok, err := u.users.UpdateDigest(ctx, tx, user)
	if err != nil {
		return fmt.Errorf("update user digest: %w", err)
	}

	if _, err := u.sessions.DeleteByUserID(ctx, tx, userID); err != nil {
		return fmt.Errorf("delete session by user id: %w", err)
	}

	if ok {
		activity := domain.NewActivity(userID, "account", "change password", time.Time{})
		if err := u.activities.Create(ctx, tx, activity); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *accountUsecase) Signout(ctx context.Context, token string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	session, err := u.sessions.GetByToken(ctx, tx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get session by token: %w", err)
	}

	ok, err := u.sessions.DeleteByToken(ctx, tx, token)
	if err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}

	if ok {
		activity := domain.NewActivity(session.UserID(), "account", "signout", time.Time{})
		if err := u.activities.Create(ctx, tx, activity); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *accountUsecase) Delete(ctx context.Context, userID int, password string) error {
	user, err := u.users.Get(ctx, u.db, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Digest()), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	ok, err := u.users.Delete(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if !ok {
		return ErrInvalidCredentials
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
