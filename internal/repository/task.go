package repository

import (
	"context"

	"github.com/fjnkt98/tasks/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, db DBTX, userID int, title string) (domain.Task, error)
	Select(ctx context.Context, db DBTX, userID int, limit int, offset int) ([]domain.Task, error)
	SelectByStatus(ctx context.Context, db DBTX, userID int, status domain.TaskStatus, limit int, offset int) ([]domain.Task, error)
	Get(ctx context.Context, db DBTX, userID int, id int) (domain.Task, error)
	UpdateTitle(ctx context.Context, db DBTX, task domain.Task) (domain.Task, error)
	UpdateStatus(ctx context.Context, db DBTX, task domain.Task) (domain.Task, error)
	Delete(ctx context.Context, db DBTX, userID int, id int) (bool, error)
	Count(ctx context.Context, db DBTX, userID int) (int, error)
	TodoCount(ctx context.Context, db DBTX, userID int) (int, error)
	DoneCount(ctx context.Context, db DBTX, userID int) (int, error)
}
