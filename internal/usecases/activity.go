package usecases

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/repository"
)

type Activity struct {
	UserID    int
	Kind      string
	Message   string
	CreatedAt time.Time
}

type ActivityUsecase interface {
	List(ctx context.Context, userID int) ([]Activity, error)
}

type activityUsecase struct {
	db         *sql.DB
	activities repository.ActivityRepository
}

var _ ActivityUsecase = &activityUsecase{}

func NewActivityUsecase(db *sql.DB) *activityUsecase {
	return &activityUsecase{
		db:         db,
		activities: repository.NewActivityRepository(),
	}
}

func (u *activityUsecase) List(ctx context.Context, userID int) ([]Activity, error) {
	data, err := u.activities.Select(ctx, u.db, userID)
	if err != nil {
		return nil, fmt.Errorf("select activities: %w", err)
	}

	activities := make([]Activity, len(data))
	for i, d := range data {
		activities[i] = Activity{
			UserID:    d.UserID,
			Kind:      d.Kind,
			Message:   d.Message,
			CreatedAt: time.Unix(d.CreatedAt, 0).UTC(),
		}
	}

	return activities, nil
}
