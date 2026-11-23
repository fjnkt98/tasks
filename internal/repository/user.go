package repository

import (
	"context"

	"github.com/fjnkt98/tasks/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, db DBTX, name string, digest []byte) (domain.User, error)
	CreateOrReplace(ctx context.Context, db DBTX, name string, digest []byte) (domain.User, error)
	Get(ctx context.Context, db DBTX, id int) (domain.User, error)
	GetByName(ctx context.Context, db DBTX, name string) (domain.User, error)
	Select(ctx context.Context, db DBTX, limit int, offset int) ([]domain.User, error)
	UpdateDigest(ctx context.Context, db DBTX, user domain.User) (bool, error)
	Delete(ctx context.Context, db DBTX, id int) (bool, error)
	SelectRoles(ctx context.Context, db DBTX, id int) ([]domain.Role, error)
	EnsureRole(ctx context.Context, db DBTX, id int, role domain.Role) error
	DeleteRole(ctx context.Context, db DBTX, id int, role domain.Role) (bool, error)
	SelectByRoles(ctx context.Context, db DBTX, role domain.Role) ([]domain.User, error)
}
