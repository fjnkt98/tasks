package domain

import "context"

type Task struct {
	ID     int
	UserID int
	Title  string
	Status string
}

type TaskRepository interface {
	Create(ctx context.Context, db DBTX, userID int, title string) (*Task, error)
	Select(ctx context.Context, db DBTX, userID int, limit int, offset int) ([]Task, error)
	SelectByStatus(ctx context.Context, db DBTX, userID int, status string, limit int, offset int) ([]Task, error)
	Get(ctx context.Context, db DBTX, userID int, id int) (*Task, error)
	UpdateTitle(ctx context.Context, db DBTX, userID int, id int, title string) (*Task, error)
	UpdateStatus(ctx context.Context, db DBTX, userID int, id int, status string) (*Task, error)
	Delete(ctx context.Context, db DBTX, userID int, id int) (bool, error)
	Count(ctx context.Context, db DBTX, userID int) (int, error)
	TodoCount(ctx context.Context, db DBTX, userID int) (int, error)
	DoneCount(ctx context.Context, db DBTX, userID int) (int, error)
}
