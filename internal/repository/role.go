package repository

import (
	"context"

	"github.com/fjnkt98/tasks/internal/domain"
)

type RoleRepository interface {
	Ensure(ctx context.Context, db DBTX, role domain.Role) (domain.Role, error)
	Select(ctx context.Context, db DBTX) ([]domain.Role, error)
	Delete(ctx context.Context, db DBTX, role domain.Role) (bool, error)
}
