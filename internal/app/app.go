package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/fjnkt98/tasks/internal/adapter/sqlite"
	"github.com/fjnkt98/tasks/internal/adapter/web"
	"github.com/fjnkt98/tasks/internal/usecase"
)

func NewAdminUsecase(db *sql.DB) usecase.AdminUsecase {
	return usecase.NewAdminUsecase(
		db,
		sqlite.NewUserRepository(),
		sqlite.NewSessionRepository(),
		sqlite.NewRoleRepository(),
	)
}

func NewHandler(db *sql.DB) http.Handler {
	return web.NewHandler(web.Deps{
		Accounts: usecase.NewAccountUsecase(
			db,
			sqlite.NewUserRepository(),
			sqlite.NewSessionRepository(),
			sqlite.NewRoleRepository(),
			sqlite.NewActivityRepository(),
		),
		Sessions: usecase.NewSessionUsecase(
			db,
			sqlite.NewUserRepository(),
			sqlite.NewSessionRepository(),
		),
		Tasks: usecase.NewTaskUsecase(
			db,
			sqlite.NewTaskRepository(),
			sqlite.NewActivityRepository(),
		),
		Users: usecase.NewUserUsecase(
			db,
			sqlite.NewUserRepository(),
			sqlite.NewTaskRepository(),
			sqlite.NewActivityRepository(),
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
