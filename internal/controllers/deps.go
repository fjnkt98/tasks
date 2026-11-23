package controllers

import "github.com/fjnkt98/tasks/internal/usecases"

type Deps struct {
	Accounts usecases.AccountUsecase
	Sessions usecases.SessionUsecase
	Tasks    usecases.TaskUsecase
	Users    usecases.UserUsecase
}
