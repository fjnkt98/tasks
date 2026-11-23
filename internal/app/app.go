package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/fjnkt98/tasks/internal/adapter/web"
	"github.com/fjnkt98/tasks/internal/repository"
	"github.com/fjnkt98/tasks/internal/usecase"
)

func NewAdminUsecase(db *sql.DB) usecase.AdminUsecase {
	return usecase.NewAdminUsecase(
		db,
		repository.NewUserRepository(),
		repository.NewSessionRepository(),
		repository.NewRoleRepository(),
	)
}

func NewHandler(db *sql.DB) http.Handler {
	return web.NewHandler(web.Deps{
		Accounts: usecase.NewAccountUsecase(
			db,
			repository.NewUserRepository(),
			repository.NewSessionRepository(),
			repository.NewRoleRepository(),
			repository.NewActivityRepository(),
		),
		Sessions: usecase.NewSessionUsecase(
			db,
			repository.NewUserRepository(),
			repository.NewSessionRepository(),
		),
		Tasks: usecase.NewTaskUsecase(
			db,
			repository.NewTaskRepository(),
			repository.NewUserRepository(),
			repository.NewActivityRepository(),
		),
		Users: usecase.NewUserUsecase(
			db,
			repository.NewUserRepository(),
			repository.NewTaskRepository(),
			repository.NewActivityRepository(),
		),
	})
}

func NewServer(port int, db *sql.DB) (*http.Server, error) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           NewHandler(db),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return server, nil
}
