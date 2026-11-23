package usecases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
)

type TaskUsecase interface {
	List(ctx context.Context, userID int, status string, limit int, page int) ([]domain.Task, error)
	Get(ctx context.Context, userID int, id int) (domain.Task, error)
	Add(ctx context.Context, userID int, title string) (domain.Task, error)
	UpdateTitle(ctx context.Context, userID int, id int, title string) (domain.Task, error)
	UpdateStatus(ctx context.Context, userID int, id int, status string) (domain.Task, error)
	Delete(ctx context.Context, userID int, id int) error
	Count(ctx context.Context, userID int, status string) (int, error)
}

type taskUsecase struct {
	db         *sql.DB
	tasks      TaskRepository
	users      UserRepository
	activities ActivityRepository
}

func NewTaskUsecase(db *sql.DB, tasks TaskRepository, users UserRepository, activities ActivityRepository) *taskUsecase {
	return &taskUsecase{
		db:         db,
		tasks:      tasks,
		users:      users,
		activities: activities,
	}
}

var _ TaskUsecase = &taskUsecase{}

func (u *taskUsecase) List(ctx context.Context, userID int, status string, limit int, page int) ([]domain.Task, error) {
	offset := limit * (page - 1)

	if status == "" {
		tasks, err := u.tasks.List(ctx, u.db, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		return tasks, nil
	}

	st, err := domain.ParseTaskStatus(status)
	if err != nil {
		return nil, err
	}
	tasks, err := u.tasks.ListByStatus(ctx, u.db, userID, st, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tasks by status: %w", err)
	}
	return tasks, nil
}

func (u *taskUsecase) Get(ctx context.Context, userID int, id int) (domain.Task, error) {
	task, err := u.tasks.FindByID(ctx, u.db, userID, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task by id: %w", err)
	}
	return task, nil
}

func (u *taskUsecase) Add(ctx context.Context, userID int, title string) (domain.Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	user, err := u.users.FindByID(ctx, tx, userID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("find user: %w", err)
	}

	task, err := PrepareAddTask(user, title)
	if err != nil {
		return domain.Task{}, err
	}
	task, err = u.tasks.Save(ctx, tx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	event, err := EventTaskAdded(&user, task)
	if err != nil {
		return domain.Task{}, err
	}
	if err := saveEvents(ctx, tx, u.activities, []Event{event}); err != nil {
		return domain.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("commit transaction: %w", err)
	}

	return task, nil
}

func (u *taskUsecase) UpdateTitle(ctx context.Context, userID int, id int, title string) (domain.Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	task, err := u.tasks.FindByID(ctx, tx, userID, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("find task: %w", err)
	}

	task, err = PrepareUpdateTaskTitle(task, title)
	if err != nil {
		return domain.Task{}, err
	}
	task, err = u.tasks.Save(ctx, tx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	event, err := EventTaskTitleUpdated(task.User(), task)
	if err != nil {
		return domain.Task{}, err
	}
	if err := saveEvents(ctx, tx, u.activities, []Event{event}); err != nil {
		return domain.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("commit transaction: %w", err)
	}

	return task, nil
}

func (u *taskUsecase) UpdateStatus(ctx context.Context, userID int, id int, status string) (domain.Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Task{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	task, err := u.tasks.FindByID(ctx, tx, userID, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("find task: %w", err)
	}

	task, st, err := PrepareUpdateTaskStatus(task, status)
	if err != nil {
		return domain.Task{}, err
	}
	task, err = u.tasks.Save(ctx, tx, task)
	if err != nil {
		return domain.Task{}, fmt.Errorf("save task: %w", err)
	}

	event, err := EventTaskStatusUpdated(task.User(), task, st)
	if err != nil {
		return domain.Task{}, err
	}
	if err := saveEvents(ctx, tx, u.activities, []Event{event}); err != nil {
		return domain.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Task{}, fmt.Errorf("commit transaction: %w", err)
	}

	return task, nil
}

func (u *taskUsecase) Delete(ctx context.Context, userID int, id int) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	user, err := u.users.FindByID(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	ok, err := u.tasks.Delete(ctx, tx, userID, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if ok {
		event, err := EventTaskDeleted(&user, id)
		if err != nil {
			return err
		}
		if err := saveEvents(ctx, tx, u.activities, []Event{event}); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *taskUsecase) Count(ctx context.Context, userID int, status string) (int, error) {
	if status == "" {
		count, err := u.tasks.Count(ctx, u.db, userID)
		if err != nil {
			return 0, fmt.Errorf("count tasks: %w", err)
		}
		return count, nil
	}

	st, err := domain.ParseTaskStatus(status)
	if err != nil {
		return 0, err
	}
	count, err := u.tasks.CountByStatus(ctx, u.db, userID, st)
	if err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}
	return count, nil
}
