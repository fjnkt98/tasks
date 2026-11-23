package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/repository"
)

type AdminUsecase interface {
	Setup(ctx context.Context, username, password string) error
}

type adminUsecase struct {
	db       *sql.DB
	accounts repository.UserRepository
	sessions repository.SessionRepository
	roles    repository.RoleRepository
}

func NewAdminUsecase(db *sql.DB, accounts repository.UserRepository, sessions repository.SessionRepository, roles repository.RoleRepository) *adminUsecase {
	return &adminUsecase{
		db:       db,
		accounts: accounts,
		sessions: sessions,
		roles:    roles,
	}
}

var _ AdminUsecase = &adminUsecase{}

func (u *adminUsecase) Setup(ctx context.Context, username, password string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	role, err := u.roles.Ensure(ctx, tx, domain.RoleAdmin)
	if err != nil {
		return fmt.Errorf("ensure admin role: %w", err)
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generate digest for admin password: %w", err)
	}

	admin, err := u.accounts.CreateOrReplace(ctx, tx, username, digest)
	if err != nil {
		return fmt.Errorf("create or replace admin: %w", err)
	}

	if err := u.accounts.EnsureRole(ctx, tx, admin.ID(), role); err != nil {
		return fmt.Errorf("grant admin role for admin user: %w", err)
	}

	users, err := u.accounts.SelectByRoles(ctx, tx, role)
	if err != nil {
		return fmt.Errorf("select users which have admin role: %w", err)
	}
	for _, user := range users {
		if user.ID() == admin.ID() {
			continue
		}
		if _, err := u.accounts.Delete(ctx, tx, user.ID()); err != nil {
			return fmt.Errorf("delete old admin user: %w", err)
		}
	}

	if _, err := u.sessions.DeleteByUserID(ctx, tx, admin.ID()); err != nil {
		return fmt.Errorf("delete admin's sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
