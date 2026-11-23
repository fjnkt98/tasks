package usecases_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/domain"
)

func mustUser(t *testing.T, id int, name string, roles []domain.Role) domain.User {
	t.Helper()
	user, err := domain.NewUser(id, name, "digest", roles...)
	require.NoError(t, err)
	return user
}

func mustTask(t *testing.T, id int, user *domain.User, title string, status domain.TaskStatus) domain.Task {
	t.Helper()
	task, err := domain.NewTask(id, user, title, status)
	require.NoError(t, err)
	return task
}
