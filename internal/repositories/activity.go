package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecases"
)

type activityRepository struct {
	users *userRepository
}

func NewActivityRepository() *activityRepository {
	return &activityRepository{
		users: NewUserRepository(),
	}
}

var _ usecases.ActivityRepository = &activityRepository{}

func (r *activityRepository) Save(ctx context.Context, db usecases.DBTX, activity domain.Activity) error {
	user := activity.User()
	if user == nil {
		return fmt.Errorf("activity user is required")
	}
	q := "INSERT INTO activities (user_id, kind, message) VALUES (?, ?, ?)"
	_, err := db.ExecContext(ctx, q, user.ID(), activity.Kind(), activity.Message())
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	return nil
}

func (r *activityRepository) ListByUserID(ctx context.Context, db usecases.DBTX, userID int) ([]domain.Activity, error) {
	user, err := r.users.FindByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	owner := &user

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
		activity, err := domain.NewActivity(owner, kind, message, time.Unix(createdAt, 0).UTC())
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return activities, nil
}
