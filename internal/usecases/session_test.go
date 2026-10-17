package usecases

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionUsecase(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', 'digest'),
			(2, 'user2', 'digest')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(1, 'token1', 1136171045),
			(2, 'token2', 32693079845)`,
		)
		require.NoError(t, err)
	}

	t.Run("session not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewSessionUsecase(db)

		user, err := u.Continue(t.Context(), "foo")
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("session expired", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewSessionUsecase(db)

		user, err := u.Continue(t.Context(), "token1")
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewSessionUsecase(db)

		s, err := u.Continue(t.Context(), "token2")
		require.NoError(t, err)
		assert.Equal(t, 2, s.UserID)

		session, err := u.sessions.GetByToken(t.Context(), db, "token2")
		require.NoError(t, err)
		assert.NotEqual(t, 32693079845, session.ExpiresAt)
	})
}
