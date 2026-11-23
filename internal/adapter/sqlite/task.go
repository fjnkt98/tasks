package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fjnkt98/tasks/internal/domain"
)

type taskRepository struct{}

func NewTaskRepository() *taskRepository {
	return &taskRepository{}
}

var _ domain.TaskRepository = &taskRepository{}

func (r *taskRepository) Create(ctx context.Context, db domain.DBTX, userID int, title string) (*domain.Task, error) {
	q := "INSERT INTO tasks (user_id, title) VALUES (?, ?) RETURNING id, user_id, title, status"
	row := db.QueryRowContext(ctx, q, userID, title)

	var task domain.Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &task, nil
}

func (r *taskRepository) Select(ctx context.Context, db domain.DBTX, userID int, limit int, offset int) ([]domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}

func (r *taskRepository) SelectByStatus(ctx context.Context, db domain.DBTX, userID int, status string, limit int, offset int) ([]domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND status = ? ORDER BY status ASC, id DESC LIMIT ? OFFSET ?"
	rows, err := db.QueryContext(ctx, q, userID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
			return nil, fmt.Errorf("scan rows: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tasks, nil
}

func (r *taskRepository) Get(ctx context.Context, db domain.DBTX, userID int, id int) (*domain.Task, error) {
	q := "SELECT id, user_id, title, status FROM tasks WHERE user_id = ? AND id = ?"
	row := db.QueryRowContext(ctx, q, userID, id)
	var task domain.Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("execute: %w", err)
	}
	return &task, nil
}

func (r *taskRepository) UpdateTitle(ctx context.Context, db domain.DBTX, userID int, id int, title string) (*domain.Task, error) {
	q := "UPDATE tasks SET title = ?, updated_at = UNIXEPOCH() WHERE user_id = ? AND id = ? RETURNING id, user_id, title, status"
	row := db.QueryRowContext(ctx, q, title, userID, id)
	var task domain.Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("execute: %w", err)
	}
	return &task, nil
}

func (r *taskRepository) UpdateStatus(ctx context.Context, db domain.DBTX, userID int, id int, status string) (*domain.Task, error) {
	q := "UPDATE tasks SET status = ?, updated_at = UNIXEPOCH() WHERE user_id = ? AND id = ? RETURNING id, user_id, title, status"
	row := db.QueryRowContext(ctx, q, status, userID, id)
	var task domain.Task
	if err := row.Scan(&task.ID, &task.UserID, &task.Title, &task.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("execute: %w", err)
	}
	return &task, nil
}

func (r *taskRepository) Delete(ctx context.Context, db domain.DBTX, userID int, id int) (bool, error) {
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

func (r *taskRepository) Count(ctx context.Context, db domain.DBTX, userID int) (int, error) {
	q := "SELECT COUNT(*) FROM tasks WHERE user_id = ?"
	row := db.QueryRowContext(ctx, q, userID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("execute: %w", err)
	}

	return count, nil
}

func (r *taskRepository) TodoCount(ctx context.Context, db domain.DBTX, userID int) (int, error) {
	q := "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'created'"
	row := db.QueryRowContext(ctx, q, userID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("execute: %w", err)
	}

	return count, nil
}

func (r *taskRepository) DoneCount(ctx context.Context, db domain.DBTX, userID int) (int, error) {
	q := "SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'done'"
	row := db.QueryRowContext(ctx, q, userID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("execute: %w", err)
	}

	return count, nil
}
