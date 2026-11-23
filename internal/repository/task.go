package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
	"github.com/fjnkt98/tasks/internal/usecase"
)

type taskRepository struct{}

func NewTaskRepository() *taskRepository {
	return &taskRepository{}
}

var _ usecase.TaskRepository = &taskRepository{}

func scanTask(id, userID int, title, status string) (domain.Task, error) {
	st, err := domain.ParseTaskStatus(status)
	if err != nil {
		return domain.Task{}, err
	}
	return domain.NewTask(id, userID, title, st), nil
}

func (r *taskRepository) Save(ctx context.Context, db usecase.DBTX, task domain.Task) (domain.Task, error) {
	// id=0 inserts; non-zero upserts only if the row still exists (no resurrection).
	// Ownership WHERE: wrong owner → no update → ErrNotFound.
	q := `
INSERT INTO tasks (id, user_id, title, status)
SELECT NULLIF(?, 0), ?, ?, ?
WHERE ? = 0 OR EXISTS (SELECT 1 FROM tasks WHERE id = ?)
ON CONFLICT(id) DO UPDATE SET
	title = excluded.title,
	status = excluded.status,
	updated_at = UNIXEPOCH()
WHERE tasks.user_id = excluded.user_id
RETURNING id, user_id, title, status`
	row := db.QueryRowContext(ctx, q, task.ID(), task.UserID(), task.Title(), string(task.Status()), task.ID(), task.ID())
	var id, owner int
	var title, status string
	if err := row.Scan(&id, &owner, &title, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, usecase.ErrNotFound
		}
		return domain.Task{}, fmt.Errorf("execute: %w", err)
	}
	return scanTask(id, owner, title, status)
}

func (r *taskRepository) FindByID(ctx context.Context, db usecase.DBTX, userID int, id int) (domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND id = ?"
	row := db.QueryRowContext(ctx, q, userID, id)
	var taskID, owner int
	var title, status string
	if err := row.Scan(&taskID, &owner, &title, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, usecase.ErrNotFound
		}
		return domain.Task{}, fmt.Errorf("execute: %w", err)
	}
	return scanTask(taskID, owner, title, status)
}

func (r *taskRepository) List(ctx context.Context, db usecase.DBTX, userID int, limit int, offset int) ([]domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var id, owner int
		var title, status string
		if err := rows.Scan(&id, &owner, &title, &status); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		task, err := scanTask(id, owner, title, status)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return tasks, nil
}

func (r *taskRepository) ListByStatus(ctx context.Context, db usecase.DBTX, userID int, status domain.TaskStatus, limit int, offset int) ([]domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND status = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, userID, string(status), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var id, owner int
		var title, st string
		if err := rows.Scan(&id, &owner, &title, &st); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		task, err := scanTask(id, owner, title, st)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return tasks, nil
}

func (r *taskRepository) Delete(ctx context.Context, db usecase.DBTX, userID int, id int) (bool, error) {
	q := "DELETE FROM tasks WHERE user_id = ? AND id = ?"
	result, err := db.ExecContext(ctx, q, userID, id)
	if err != nil {
		return false, fmt.Errorf("execute: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected: %w", err)
	}
	return count > 0, nil
}

func (r *taskRepository) Count(ctx context.Context, db usecase.DBTX, userID int) (int, error) {
	q := "SELECT COUNT(*) FROM tasks WHERE user_id = ?"
	row := db.QueryRowContext(ctx, q, userID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("execute: %w", err)
	}
	return count, nil
}

func (r *taskRepository) CountByStatus(ctx context.Context, db usecase.DBTX, userID int, status domain.TaskStatus) (int, error) {
	q := "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = ?"
	row := db.QueryRowContext(ctx, q, userID, string(status))
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("execute: %w", err)
	}
	return count, nil
}
