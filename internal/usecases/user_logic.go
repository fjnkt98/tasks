package usecases

import (
	"fmt"
	"time"

	"github.com/fjnkt98/tasks/internal/domain"
)

func PrepareGrantRole(user domain.User, role string) (domain.User, Event, error) {
	next := user.GrantRole(domain.Role(role))
	event, err := domain.NewActivity(&next, domain.ActivityKindAccount, fmt.Sprintf("grant role: %q", role), time.Time{})
	if err != nil {
		return domain.User{}, Event{}, err
	}
	return next, event, nil
}

func PrepareRevokeRole(user domain.User, role string) (domain.User, []Event, error) {
	had := user.HasRole(domain.Role(role))
	next := user.RevokeRole(domain.Role(role))
	if !had {
		return next, nil, nil
	}
	event, err := domain.NewActivity(&next, domain.ActivityKindAccount, fmt.Sprintf("revoke role: %q", role), time.Time{})
	if err != nil {
		return domain.User{}, nil, err
	}
	return next, []Event{event}, nil
}
