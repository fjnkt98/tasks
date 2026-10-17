package usecases

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fjnkt98/tasks/internal/repository"
)

type Task struct {
	ID     int
	UserID int
	Title  string
	Status string
}

type TaskUsecase interface {
	List(ctx context.Context, userID int, status string, limit int, page int) ([]Task, error)
	Get(ctx context.Context, userID int, id int) (*Task, error)
	Add(ctx context.Context, userID int, title string) (*Task, error)
	UpdateTitle(ctx context.Context, userID int, id int, title string) (*Task, error)
	UpdateStatus(ctx context.Context, userID int, id int, status string) (*Task, error)
	Delete(ctx context.Context, userID int, id int) error
	Count(ctx context.Context, userID int, status string) (int, error)
}

type taskUsecase struct {
	db         *sql.DB
	tasks      repository.TaskRepository
	activities repository.ActivityRepository
}

func NewTaskUsecase(db *sql.DB) *taskUsecase {
	return &taskUsecase{
		db:         db,
		tasks:      repository.NewTaskRepository(),
		activities: repository.NewActivityRepository(),
	}
}

var _ TaskUsecase = &taskUsecase{}

func (u *taskUsecase) List(ctx context.Context, userID int, status string, limit int, page int) ([]Task, error) {
	offset := limit * (page - 1)

	var tasks []repository.Task
	var err error
	if status == "" {
		tasks, err = u.tasks.Select(ctx, u.db, userID, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("select tasks: %w", err)
		}
	} else {
		tasks, err = u.tasks.SelectByStatus(ctx, u.db, userID, status, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("select tasks by status: %w", err)
		}
	}

	results := make([]Task, len(tasks))
	for i, t := range tasks {
		results[i] = Task{
			ID:     t.ID,
			UserID: t.UserID,
			Title:  t.Title,
			Status: t.Status,
		}
	}

	return results, nil
}

func (u *taskUsecase) Get(ctx context.Context, userID int, id int) (*Task, error) {
	task, err := u.tasks.Get(ctx, u.db, userID, id)
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	return &Task{
		ID:     task.ID,
		UserID: task.UserID,
		Title:  task.Title,
		Status: task.Status,
	}, nil
}

func (u *taskUsecase) Add(ctx context.Context, userID int, title string) (*Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	task, err := u.tasks.Create(ctx, tx, userID, title)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := u.activities.Create(ctx, tx, userID, "task", fmt.Sprintf(`add task %d, title: "%s"`, task.ID, task.Title)); err != nil {
		return nil, fmt.Errorf("create activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &Task{
		ID:     task.ID,
		UserID: task.UserID,
		Title:  task.Title,
		Status: task.Status,
	}, nil
}

func (u *taskUsecase) UpdateTitle(ctx context.Context, userID int, id int, title string) (*Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	task, err := u.tasks.UpdateTitle(ctx, tx, userID, id, title)
	if err != nil {
		return nil, fmt.Errorf("update task title: %w", err)
	}

	if err := u.activities.Create(ctx, tx, userID, "task", fmt.Sprintf(`edit task %d, title: "%s"`, task.ID, task.Title)); err != nil {
		return nil, fmt.Errorf("create activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &Task{
		ID:     task.ID,
		UserID: task.UserID,
		Title:  task.Title,
		Status: task.Status,
	}, nil
}

func (u *taskUsecase) UpdateStatus(ctx context.Context, userID int, id int, status string) (*Task, error) {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	task, err := u.tasks.UpdateStatus(ctx, tx, userID, id, status)
	if err != nil {
		return nil, fmt.Errorf("update task title: %w", err)
	}

	action := "done"
	if status == "created" {
		action = "undone"
	}

	if err := u.activities.Create(ctx, tx, userID, "task", fmt.Sprintf(`%s task %d, "%s"`, action, task.ID, task.Title)); err != nil {
		return nil, fmt.Errorf("create activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &Task{
		ID:     task.ID,
		UserID: task.UserID,
		Title:  task.Title,
		Status: task.Status,
	}, nil
}

func (u *taskUsecase) Delete(ctx context.Context, userID int, id int) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	ok, err := u.tasks.Delete(ctx, tx, userID, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if ok {
		if err := u.activities.Create(ctx, tx, userID, "task", fmt.Sprintf("delete task %d", id)); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *taskUsecase) Count(ctx context.Context, userID int, status string) (int, error) {
	var count int
	var err error

	switch status {
	case "created":
		count, err = u.tasks.TodoCount(ctx, u.db, userID)
	case "done":
		count, err = u.tasks.DoneCount(ctx, u.db, userID)
	default:
		count, err = u.tasks.Count(ctx, u.db, userID)
	}

	if err != nil {
		return 0, fmt.Errorf("count tasks: %w", err)
	}

	return count, nil
}
