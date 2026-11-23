package domain

import "context"

type Activity struct {
	UserID    int
	Kind      string
	Message   string
	CreatedAt int64
}

type ActivityRepository interface {
	Create(ctx context.Context, db DBTX, userID int, kind string, message string) error
	Select(ctx context.Context, db DBTX, userID int) ([]Activity, error)
}
