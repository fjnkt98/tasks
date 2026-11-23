package web

import "github.com/fjnkt98/tasks/internal/usecase"

type Deps struct {
	Accounts usecase.AccountUsecase
	Sessions usecase.SessionUsecase
	Tasks    usecase.TaskUsecase
	Users    usecase.UserUsecase
}
