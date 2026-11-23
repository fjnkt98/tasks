package sqlite

import (
	"context"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
)

type activityRepository struct{}

func NewActivityRepository() *activityRepository {
	return &activityRepository{}
}

var _ domain.ActivityRepository = &activityRepository{}

func (r *activityRepository) Create(ctx context.Context, db domain.DBTX, userID int, kind string, message string) error {
	q := "INSERT INTO activities (user_id, kind, message) VALUES (?, ?, ?)"
	_, err := db.ExecContext(ctx, q, userID, kind, message)
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *activityRepository) Select(ctx context.Context, db domain.DBTX, userID int) ([]domain.Activity, error) {
	q := "SELECT kind, message, created_at FROM activities WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT 100"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	activities := make([]domain.Activity, 0)
	for rows.Next() {
		var activity domain.Activity
		if err := rows.Scan(&activity.Kind, &activity.Message, &activity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		activity.UserID = userID
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return activities, nil
}
