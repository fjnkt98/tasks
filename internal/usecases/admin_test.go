package usecases

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/repository"
	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUsecase(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		for _, q := range []string{
			`INSERT INTO users (name, digest) VALUES
				('admin', 'digest'),
				('old_admin', 'digest')
			`,
			`INSERT INTO roles (name) VALUES
				('admin')
			`,
			`INSERT INTO user_role_relations (user_id, role) VALUES
				(1, 'admin'),
				(2, 'admin')
			`,
			`INSERT INTO sessions (user_id, token, expires_at) VALUES
				(1, 'token1', 10000),
				(2, 'token2', 10000)
			`,
		} {
			_, err := db.ExecContext(t.Context(), q)
			require.NoError(t, err)
		}
	}

	t.Run("initialize", func(t *testing.T) {
		db := testutil.NewDB(t)

		u := NewAdminUsecase(db)
		err := u.Setup(t.Context(), "admin", "password")
		require.NoError(t, err)

		userRepo := repository.NewUserRepository()
		user, err := userRepo.GetByName(t.Context(), db, "admin")
		require.NoError(t, err)
		assert.Equal(t, 1, user.ID)

		roles, err := userRepo.SelectRoles(t.Context(), db, user.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"admin"}, roles)
	})

	t.Run("overwrite", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAdminUsecase(db)
		err := u.Setup(t.Context(), "admin", "password")
		require.NoError(t, err)

		userRepo := repository.NewUserRepository()
		user, err := userRepo.GetByName(t.Context(), db, "admin")
		require.NoError(t, err)
		assert.Equal(t, 1, user.ID)
		assert.NotEqual(t, "digest", user.Digest)

		roles, err := userRepo.SelectRoles(t.Context(), db, user.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"admin"}, roles)

		user, err = userRepo.GetByName(t.Context(), db, "old_admin")
		require.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, user)

		sessionRepo := repository.NewSessionRepository()
		_, err = sessionRepo.GetByToken(t.Context(), db, "token1")
		require.ErrorIs(t, err, repository.ErrNotFound)
		_, err = sessionRepo.GetByToken(t.Context(), db, "token2")
		require.ErrorIs(t, err, repository.ErrNotFound)
	})
}
