package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRepository(t *testing.T) {
	date := time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)

	var fixture = func(t *testing.T, db *sql.DB) {
		_, err := db.ExecContext(t.Context(), `INSERT INTO users (name, digest) VALUES
			('user1', 'digest'),
			('user2', 'digest')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(1, 'token1', ?),
			(1, 'token2', ?),
			(2, 'token3', ?)`,
			date.Unix(),
			time.Now().Add(24*time.Hour).Unix(),
			date.Unix(),
		)
		require.NoError(t, err)
	}

	t.Run("no results", func(t *testing.T) {
		db := testutil.NewDB(t)

		repo := NewSessionRepository()

		// get non-existent session
		session, err := repo.GetByToken(t.Context(), db, "foo")
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, session)

		// update non-existent session
		ok, err := repo.UpdateByToken(t.Context(), db, "foo", date.Add(24*time.Hour))
		require.NoError(t, err)
		assert.False(t, ok)

		// delete non-existent session
		ok, err = repo.DeleteByUserID(t.Context(), db, 0)
		require.NoError(t, err)
		assert.False(t, ok)

		ok, err = repo.DeleteByToken(t.Context(), db, "foo")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("get", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		session, err := repo.GetByToken(t.Context(), db, "token1")
		require.NoError(t, err)
		assert.Equal(t, &Session{UserID: 1, Token: "token1", ExpiresAt: date.Unix()}, session)

		// get non-existent session
		session, err = repo.GetByToken(t.Context(), db, "foo")
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, session)
	})

	t.Run("create", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		// create session
		session, err := repo.Create(t.Context(), db, 1, "digest", "token4", date)
		require.NoError(t, err)
		assert.Equal(t, &Session{UserID: 1, Token: "token4", ExpiresAt: date.Unix()}, session)

		// create session with invalid digest
		session, err = repo.Create(t.Context(), db, 2, "foo", "token", date)
		require.ErrorIs(t, err, ErrInvalidCredentials)
		session, err = repo.GetByToken(t.Context(), db, "token")
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, session)
	})

	t.Run("update", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		ok, err := repo.UpdateByToken(t.Context(), db, "token1", date.Add(24*time.Hour))
		require.NoError(t, err)
		assert.True(t, ok)
		session, err := repo.GetByToken(t.Context(), db, "token1")
		require.NoError(t, err)
		assert.Equal(t, &Session{UserID: 1, Token: "token1", ExpiresAt: date.Add(24 * time.Hour).Unix()}, session)
	})

	t.Run("delete by user", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		ok, err := repo.DeleteByUserID(t.Context(), db, 1)
		require.NoError(t, err)
		assert.True(t, ok)

		_, err = repo.GetByToken(t.Context(), db, "token1")
		require.ErrorIs(t, err, sql.ErrNoRows)
		_, err = repo.GetByToken(t.Context(), db, "token2")
		require.ErrorIs(t, err, sql.ErrNoRows)

		session, err := repo.GetByToken(t.Context(), db, "token3")
		require.NoError(t, err)
		assert.Equal(t, &Session{UserID: 2, Token: "token3", ExpiresAt: date.Unix()}, session)
	})

	t.Run("delete by token", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		ok, err := repo.DeleteByToken(t.Context(), db, "token3")
		require.NoError(t, err)
		assert.True(t, ok)

		_, err = repo.GetByToken(t.Context(), db, "token1")
		require.NoError(t, err)
		_, err = repo.GetByToken(t.Context(), db, "token2")
		require.NoError(t, err)
		_, err = repo.GetByToken(t.Context(), db, "token3")
		require.ErrorIs(t, err, sql.ErrNoRows)
	})

	t.Run("delete expired", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewSessionRepository()

		ok, err := repo.DeleteExpired(t.Context(), db)
		require.NoError(t, err)
		assert.True(t, ok)

		_, err = repo.GetByToken(t.Context(), db, "token1")
		require.ErrorIs(t, err, sql.ErrNoRows)
		_, err = repo.GetByToken(t.Context(), db, "token2")
		require.NoError(t, err)
		_, err = repo.GetByToken(t.Context(), db, "token3")
		require.ErrorIs(t, err, sql.ErrNoRows)
	})
}
