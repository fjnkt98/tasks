package usecase

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrUniqueConstraint = errors.New("unique constraint violation")
var ErrInvalidCredentials = errors.New("invalid credentials")

type DBTX interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ DBTX = &sql.DB{}
var _ DBTX = &sql.Tx{}

type TaskRepository interface {
	Save(ctx context.Context, db DBTX, task domain.Task) (domain.Task, error)
	FindByID(ctx context.Context, db DBTX, userID int, id int) (domain.Task, error)
	List(ctx context.Context, db DBTX, userID int, limit int, offset int) ([]domain.Task, error)
	ListByStatus(ctx context.Context, db DBTX, userID int, status domain.TaskStatus, limit int, offset int) ([]domain.Task, error)
	Delete(ctx context.Context, db DBTX, userID int, id int) (bool, error)
	Count(ctx context.Context, db DBTX, userID int) (int, error)
	CountByStatus(ctx context.Context, db DBTX, userID int, status domain.TaskStatus) (int, error)
}

type UserRepository interface {
	Save(ctx context.Context, db DBTX, user domain.User) (domain.User, error)
	FindByID(ctx context.Context, db DBTX, id int) (domain.User, error)
	FindByName(ctx context.Context, db DBTX, name string) (domain.User, error)
	List(ctx context.Context, db DBTX, limit int, offset int) ([]domain.User, error)
	ListByRole(ctx context.Context, db DBTX, role domain.Role) ([]domain.User, error)
	Delete(ctx context.Context, db DBTX, id int) (bool, error)
}

type SessionRepository interface {
	Save(ctx context.Context, db DBTX, session domain.Session) (domain.Session, error)
	FindByToken(ctx context.Context, db DBTX, token string) (domain.Session, error)
	DeleteByToken(ctx context.Context, db DBTX, token string) (bool, error)
	DeleteByUserID(ctx context.Context, db DBTX, userID int) (bool, error)
	DeleteExpired(ctx context.Context, db DBTX, now time.Time) (bool, error)
}

type ActivityRepository interface {
	Save(ctx context.Context, db DBTX, activity domain.Activity) error
	ListByUserID(ctx context.Context, db DBTX, userID int) ([]domain.Activity, error)
}

type RoleRepository interface {
	Ensure(ctx context.Context, db DBTX, role domain.Role) (domain.Role, error)
	List(ctx context.Context, db DBTX) ([]domain.Role, error)
	Delete(ctx context.Context, db DBTX, role domain.Role) (bool, error)
}
