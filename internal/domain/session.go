package domain

import (
	"context"
	"time"
)

type Session struct {
	UserID    int
	Token     string
	ExpiresAt int64
}

type SessionRepository interface {
	Create(ctx context.Context, db DBTX, userID int, digest string, token string, expiresAt time.Time) (*Session, error)
	GetByToken(ctx context.Context, db DBTX, token string) (*Session, error)
	UpdateByToken(ctx context.Context, db DBTX, token string, expiresAt time.Time) (bool, error)
	DeleteByUserID(ctx context.Context, db DBTX, userID int) (bool, error)
	DeleteByToken(ctx context.Context, db DBTX, token string) (bool, error)
	DeleteExpired(ctx context.Context, db DBTX) (bool, error)
}
