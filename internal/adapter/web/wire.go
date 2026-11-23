package web

import (
	"database/sql"

	"github.com/fjnkt98/tasks/internal/adapter/sqlite"
	"github.com/fjnkt98/tasks/internal/usecase"
)

func newAccountUsecase(db *sql.DB) usecase.AccountUsecase {
	return usecase.NewAccountUsecase(
		db,
		sqlite.NewUserRepository(),
		sqlite.NewSessionRepository(),
		sqlite.NewRoleRepository(),
		sqlite.NewActivityRepository(),
	)
}

func newSessionUsecase(db *sql.DB) usecase.SessionUsecase {
	return usecase.NewSessionUsecase(
		db,
		sqlite.NewUserRepository(),
		sqlite.NewSessionRepository(),
	)
}

func newTaskUsecase(db *sql.DB) usecase.TaskUsecase {
	return usecase.NewTaskUsecase(
		db,
		sqlite.NewTaskRepository(),
		sqlite.NewActivityRepository(),
	)
}

func newUserUsecase(db *sql.DB) usecase.UserUsecase {
	return usecase.NewUserUsecase(
		db,
		sqlite.NewUserRepository(),
		sqlite.NewTaskRepository(),
		sqlite.NewActivityRepository(),
	)
}

func NewAdminUsecase(db *sql.DB) usecase.AdminUsecase {
	return usecase.NewAdminUsecase(
		db,
		sqlite.NewUserRepository(),
		sqlite.NewSessionRepository(),
		sqlite.NewRoleRepository(),
	)
}
