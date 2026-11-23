package repository

import (
	"context"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

type SessionRepository interface {
	Create(ctx context.Context, db DBTX, userID int, digest string, token string, expiresAt time.Time) (domain.Session, error)
	GetByToken(ctx context.Context, db DBTX, token string) (domain.Session, error)
	UpdateByToken(ctx context.Context, db DBTX, session domain.Session) (bool, error)
	DeleteByUserID(ctx context.Context, db DBTX, userID int) (bool, error)
	DeleteByToken(ctx context.Context, db DBTX, token string) (bool, error)
	DeleteExpired(ctx context.Context, db DBTX) (bool, error)
}
