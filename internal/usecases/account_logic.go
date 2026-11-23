package usecases

import (
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

func EventSignin(user *domain.User) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindAccount, "signin", time.Time{})
}

func EventChangePassword(user *domain.User) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindAccount, "change password", time.Time{})
}

func EventSignout(user *domain.User) (Event, error) {
	return domain.NewActivity(user, domain.ActivityKindAccount, "signout", time.Time{})
}
