package usecases

import (
	"context"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
)

// Event is a domain activity produced by usecase logic and persisted at the I/O boundary.
type Event = domain.Activity

func saveEvents(ctx context.Context, db DBTX, activities ActivityRepository, events []Event) error {
	for _, event := range events {
		if err := activities.Save(ctx, db, event); err != nil {
			return fmt.Errorf("save activity: %w", err)
		}
	}
	return nil
}
