package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

type UserList struct {
	Users   []domain.User
	HasNext bool
}

type UserDetail struct {
	User       domain.User
	TaskCount  int
	DoneCount  int
	Roles      []domain.Role
	Activities []domain.Activity
}

type UserUsecase interface {
	List(ctx context.Context, page int, limit int) (*UserList, error)
	Get(ctx context.Context, id int) (*UserDetail, error)
	GrantRole(ctx context.Context, id int, role string) error
	RevokeRole(ctx context.Context, id int, role string) error
}

type userUsecase struct {
	db         *sql.DB
	activities ActivityRepository
	tasks      TaskRepository
	users      UserRepository
	roles      RoleRepository
}

func NewUserUsecase(db *sql.DB, users UserRepository, tasks TaskRepository, activities ActivityRepository, roles RoleRepository) *userUsecase {
	return &userUsecase{
		db:         db,
		users:      users,
		tasks:      tasks,
		activities: activities,
		roles:      roles,
	}
}

var _ UserUsecase = &userUsecase{}

func (u *userUsecase) List(ctx context.Context, page int, limit int) (*UserList, error) {
	offset := (page - 1) * limit
	users, err := u.users.List(ctx, u.db, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	hasNext := len(users) == limit+1
	if hasNext {
		users = users[:limit]
	}

	return &UserList{
		Users:   users,
		HasNext: hasNext,
	}, nil
}

func (u *userUsecase) Get(ctx context.Context, id int) (*UserDetail, error) {
	user, err := u.users.FindByID(ctx, u.db, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	taskCount, err := u.tasks.Count(ctx, u.db, user.ID())
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}

	doneCount, err := u.tasks.CountByStatus(ctx, u.db, user.ID(), domain.TaskStatusDone)
	if err != nil {
		return nil, fmt.Errorf("count done tasks: %w", err)
	}

	roles, err := u.roles.ListByUser(ctx, u.db, user.ID())
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	activities, err := u.activities.ListByUserID(ctx, u.db, user.ID())
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}

	return &UserDetail{
		User:       user,
		TaskCount:  taskCount,
		DoneCount:  doneCount,
		Roles:      roles,
		Activities: activities,
	}, nil
}

func (u *userUsecase) GrantRole(ctx context.Context, userID int, role string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	if err := u.roles.Assign(ctx, tx, userID, domain.Role(role)); err != nil {
		return fmt.Errorf("assign role: %w", err)
	}

	activity := domain.NewActivity(userID, "account", fmt.Sprintf("grant role: %q", role), time.Time{})
	if err := u.activities.Save(ctx, tx, activity); err != nil {
		return fmt.Errorf("save activity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (u *userUsecase) RevokeRole(ctx context.Context, userID int, role string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback errors after Commit are expected

	ok, err := u.roles.Revoke(ctx, tx, userID, domain.Role(role))
	if err != nil {
		return fmt.Errorf("revoke role: %w", err)
	}

	if ok {
		activity := domain.NewActivity(userID, "account", fmt.Sprintf("revoke role: %q", role), time.Time{})
		if err := u.activities.Save(ctx, tx, activity); err != nil {
			return fmt.Errorf("save activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
