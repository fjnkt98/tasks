package repository

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fjnkt98/tasks/internal/testutil"
)

func TestRoleRepository(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), "INSERT INTO roles (name) VALUES ('test')")
		require.NoError(t, err)
	}

	t.Run("create", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewRoleRepository()

		role, err := repo.Ensure(t.Context(), db, "admin")
		require.NoError(t, err)
		assert.Equal(t, "admin", role)

		role, err = repo.Ensure(t.Context(), db, "admin")
		require.NoError(t, err)
		assert.Equal(t, "admin", role)
	})

	t.Run("get", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewRoleRepository()

		roles, err := repo.Select(t.Context(), db)
		require.NoError(t, err)
		assert.Equal(t, []string{"test"}, roles)
	})

	t.Run("delete", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewRoleRepository()

		ok, err := repo.Delete(t.Context(), db, "foo")
		require.NoError(t, err)
		assert.False(t, ok)
		roles, err := repo.Select(t.Context(), db)
		require.NoError(t, err)
		assert.Equal(t, []string{"test"}, roles)

		ok, err = repo.Delete(t.Context(), db, "test")
		require.NoError(t, err)
		assert.True(t, ok)
		roles, err = repo.Select(t.Context(), db)
		require.NoError(t, err)
		assert.Equal(t, []string{}, roles)
	})
}
