package usecases

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/repository"
)

type User struct {
	ID   int
	Name string
}

type UserList struct {
	Users   []User
	HasNext bool
}

type UserDetail struct {
	User
	TaskCount  int
	DoneCount  int
	Roles      []string
	Activities []Activity
}

type Activity struct {
	UserID    int
	Kind      string
	Message   string
	CreatedAt time.Time
}

type UserUsecase interface {
	List(ctx context.Context, page int, limit int) (*UserList, error)
	Get(ctx context.Context, id int) (*UserDetail, error)
	GrantRole(ctx context.Context, id int, role string) error
	RevokeRole(ctx context.Context, id int, role string) error
	// Delete(ctx context.Context, id int) error
}

type userUsecase struct {
	db         *sql.DB
	activities repository.ActivityRepository
	tasks      repository.TaskRepository
	users      repository.UserRepository
}

func NewUserUsecase(db *sql.DB) *userUsecase {
	return &userUsecase{
		db:         db,
		activities: repository.NewActivityRepository(),
		tasks:      repository.NewTaskRepository(),
		users:      repository.NewUserRepository(),
	}
}

var _ UserUsecase = &userUsecase{}

func (u *userUsecase) List(ctx context.Context, page int, limit int) (*UserList, error) {
	offset := (page - 1) * limit
	users, err := u.users.Select(ctx, u.db, limit+1, offset)
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}

	results := make([]User, len(users))
	for i, user := range users {
		results[i] = User{
			ID:   user.ID,
			Name: user.Name,
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}

	return &UserList{
		Users:   results,
		HasNext: len(users) == limit+1,
	}, nil
}

func (u *userUsecase) Get(ctx context.Context, id int) (*UserDetail, error) {
	user, err := u.users.Get(ctx, u.db, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	taskCount, err := u.tasks.Count(ctx, u.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("count tasks: %w", err)
	}

	doneCount, err := u.tasks.DoneCount(ctx, u.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("count done tasks: %w", err)
	}

	roles, err := u.users.SelectRoles(ctx, u.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("select roles: %w", err)
	}

	activities, err := u.activities.Select(ctx, u.db, user.ID)
	if err != nil {
		return nil, fmt.Errorf("select activities: %w", err)
	}

	activities2 := make([]Activity, len(activities))
	for i, a := range activities {
		activities2[i] = Activity{
			UserID:    a.UserID,
			Kind:      a.Kind,
			Message:   a.Message,
			CreatedAt: time.Unix(a.CreatedAt, 0).UTC(),
		}
	}

	return &UserDetail{
		User: User{
			ID:   user.ID,
			Name: user.Name,
		},
		TaskCount:  taskCount,
		DoneCount:  doneCount,
		Roles:      roles,
		Activities: activities2,
	}, nil
}

func (u *userUsecase) GrantRole(ctx context.Context, userID int, role string) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint:errcheck

	if err := u.users.EnsureRole(ctx, tx, userID, role); err != nil {
		return fmt.Errorf("ensure role: %w", err)
	}

	if err := u.activities.Create(ctx, tx, userID, "account", fmt.Sprintf(`grant role: "%s"`, role)); err != nil {
		return fmt.Errorf("create activity: %w", err)
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
	defer tx.Rollback() // nolint:errcheck

	ok, err := u.users.DeleteRole(ctx, tx, userID, role)
	if err != nil {
		return fmt.Errorf("add role: %w", err)
	}

	if ok {
		if err := u.activities.Create(ctx, tx, userID, "account", fmt.Sprintf(`revoke role: "%s"`, role)); err != nil {
			return fmt.Errorf("create activity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
