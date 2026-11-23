package domain

import "context"

type User struct {
	ID     int
	Name   string
	Digest string
}

type UserRepository interface {
	Create(ctx context.Context, db DBTX, name string, digest []byte) (*User, error)
	CreateOrReplace(ctx context.Context, db DBTX, name string, digest []byte) (*User, error)
	Get(ctx context.Context, db DBTX, id int) (*User, error)
	GetByName(ctx context.Context, db DBTX, name string) (*User, error)
	Select(ctx context.Context, db DBTX, limit int, offset int) ([]User, error)
	UpdateDigest(ctx context.Context, db DBTX, id int, digest []byte) (bool, error)
	Delete(ctx context.Context, db DBTX, id int) (bool, error)
	SelectRoles(ctx context.Context, db DBTX, id int) ([]string, error)
	EnsureRole(ctx context.Context, db DBTX, id int, role string) error
	DeleteRole(ctx context.Context, db DBTX, id int, role string) (bool, error)
	SelectByRoles(ctx context.Context, db DBTX, role string) ([]User, error)
}
