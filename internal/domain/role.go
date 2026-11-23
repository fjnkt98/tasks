package domain

import "context"

type RoleRepository interface {
	Ensure(ctx context.Context, db DBTX, name string) (string, error)
	Select(ctx context.Context, db DBTX) ([]string, error)
	Delete(ctx context.Context, db DBTX, name string) (bool, error)
}
