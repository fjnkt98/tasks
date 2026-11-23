package apps

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/fjnkt98/tasks/internal/controllers"
	"github.com/fjnkt98/tasks/internal/repositories"
	"github.com/fjnkt98/tasks/internal/usecases"
)

func NewAdminUsecase(db *sql.DB) usecases.AdminUsecase {
	return usecases.NewAdminUsecase(
		db,
		repositories.NewUserRepository(),
		repositories.NewSessionRepository(),
		repositories.NewRoleRepository(),
	)
}

func NewHandler(db *sql.DB) http.Handler {
	return controllers.NewHandler(controllers.Deps{
		Accounts: usecases.NewAccountUsecase(
			db,
			repositories.NewUserRepository(),
			repositories.NewSessionRepository(),
			repositories.NewRoleRepository(),
			repositories.NewActivityRepository(),
		),
		Sessions: usecases.NewSessionUsecase(
			db,
			repositories.NewSessionRepository(),
		),
		Tasks: usecases.NewTaskUsecase(
			db,
			repositories.NewTaskRepository(),
			repositories.NewUserRepository(),
			repositories.NewActivityRepository(),
		),
		Users: usecases.NewUserUsecase(
			db,
			repositories.NewUserRepository(),
			repositories.NewTaskRepository(),
			repositories.NewActivityRepository(),
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
