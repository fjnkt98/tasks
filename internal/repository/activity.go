package repository

import (
	"context"
	"errors"
	"fmt"
)

type Activity struct {
	UserID    int
	Kind      string
	Message   string
	CreatedAt int64
}

type ActivityRepository interface {
	Create(ctx context.Context, db DBTX, userID int, kind string, message string) error
	Select(ctx context.Context, db DBTX, userID int) (activities []Activity, err error)
}

type activityRepository struct{}

func NewActivityRepository() *activityRepository {
	return &activityRepository{}
}

var _ ActivityRepository = &activityRepository{}

func (r *activityRepository) Create(ctx context.Context, db DBTX, userID int, kind string, message string) error {
	q := "INSERT INTO activities (user_id, kind, message) VALUES (?, ?, ?)"
	_, err := db.ExecContext(ctx, q, userID, kind, message)
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *activityRepository) Select(ctx context.Context, db DBTX, userID int) (activities []Activity, err error) {
	q := "SELECT kind, message, created_at FROM activities WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT 100"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	activities = make([]Activity, 0)
	for rows.Next() {
		activity := Activity{UserID: userID}
		if err := rows.Scan(&activity.Kind, &activity.Message, &activity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return activities, nil
}
