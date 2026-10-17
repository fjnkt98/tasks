package usecases

import (
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantRole(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewUserUsecase(db)
		err := u.GrantRole(t.Context(), 0, "test")
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewUserUsecase(db)

		err := u.GrantRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err := u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"test"}, roles)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		// grant again (no effect)
		err = u.GrantRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err = u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"test"}, roles)

		activities, err = u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 2)
	})
}

func TestRevokeRole(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewUserUsecase(db)
		err := u.RevokeRole(t.Context(), 0, "test")
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		_, err := db.ExecContext(t.Context(), `INSERT INTO user_role_relations (user_id, role) VALUES (1, 'test')`)
		require.NoError(t, err)

		u := NewUserUsecase(db)
		err = u.RevokeRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err := u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{}, roles)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		// revoke again
		err = u.RevokeRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err = u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{}, roles)

		activities, err = u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}
