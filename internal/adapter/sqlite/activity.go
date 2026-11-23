package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/repository"
)

type activityRepository struct{}

func NewActivityRepository() *activityRepository {
	return &activityRepository{}
}

var _ repository.ActivityRepository = &activityRepository{}

func (r *activityRepository) Create(ctx context.Context, db repository.DBTX, activity domain.Activity) error {
	q := "INSERT INTO activities (user_id, kind, message) VALUES (?, ?, ?)"
	_, err := db.ExecContext(ctx, q, activity.UserID(), activity.Kind(), activity.Message())
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *activityRepository) Select(ctx context.Context, db repository.DBTX, userID int) ([]domain.Activity, error) {
	q := "SELECT kind, message, created_at FROM activities WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT 100"
	rows, err := db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	activities := make([]domain.Activity, 0)
	for rows.Next() {
		var kind, message string
		var createdAt int64
		if err := rows.Scan(&kind, &message, &createdAt); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		activities = append(activities, domain.NewActivity(userID, kind, message, time.Unix(createdAt, 0).UTC()))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return activities, nil
}
