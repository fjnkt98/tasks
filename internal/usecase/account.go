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
)

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
	users      UserRepository
	sessions   SessionRepository
	roles      RoleRepository
	activities ActivityRepository
}

var _ AccountUsecase = &accountUsecase{}

func NewAccountUsecase(db *sql.DB, users UserRepository, sessions SessionRepository, roles RoleRepository, activities ActivityRepository) *accountUsecase {
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

	user := domain.NewUser(0, username, string(digest))
	if _, err = u.users.Save(ctx, tx, user); err != nil {
		if errors.Is(err, ErrUniqueConstraint) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("save user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *accountUsecase) Signin(ctx context.Context, username string, password string) (string, error) {
	user, err := u.users.FindByName(ctx, u.db, username)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("find user by name: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Digest()), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	session := domain.NewSession(user.ID(), NewSessionToken(), time.Now().Add(24*time.Hour))
	session, err = u.sessions.Save(ctx, tx, session)
	if err != nil {
		return "", fmt.Errorf("save session: %w", err)
	}

	activity := domain.NewActivity(user.ID(), "account", "signin", time.Time{})
	if err := u.activities.Save(ctx, tx, activity); err != nil {
		return "", fmt.Errorf("save activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return session.Token(), nil
}

func (u *accountUsecase) ChangePassword(ctx context.Context, userID int, password string, newPassword string) error {
	user, err := u.users.FindByID(ctx, u.db, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("find user by id: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Digest()), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest for new password: %w", err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	user = user.WithDigest(string(digest))
	if _, err := u.users.Save(ctx, tx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	if _, err := u.sessions.DeleteByUserID(ctx, tx, userID); err != nil {
		return fmt.Errorf("delete session by user id: %w", err)
	}

	activity := domain.NewActivity(userID, "account", "change password", time.Time{})
	if err := u.activities.Save(ctx, tx, activity); err != nil {
		return fmt.Errorf("save activity: %w", err)
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

	session, err := u.sessions.FindByToken(ctx, tx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return fmt.Errorf("find session by token: %w", err)
	}

	ok, err := u.sessions.DeleteByToken(ctx, tx, token)
	if err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}

	if ok {
		activity := domain.NewActivity(session.UserID(), "account", "signout", time.Time{})
		if err := u.activities.Save(ctx, tx, activity); err != nil {
			return fmt.Errorf("save activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *accountUsecase) Delete(ctx context.Context, userID int, password string) error {
	user, err := u.users.FindByID(ctx, u.db, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("find user: %w", err)
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
