package repository

import (
	"context"

	"github.com/fjnkt98/tasks/internal/domain"
)

type ActivityRepository interface {
	Create(ctx context.Context, db DBTX, activity domain.Activity) error
	Select(ctx context.Context, db DBTX, userID int) ([]domain.Activity, error)
}
