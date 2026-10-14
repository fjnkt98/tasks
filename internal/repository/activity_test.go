package repository

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityRepository(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', 'digest'),
			(2, 'user2', 'digest')`,
		)
		require.NoError(t, err)
	}

	t.Run("create activity", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewActivityRepository()
		err := repo.Create(t.Context(), db, 1, "test", "test create")
		require.NoError(t, err)

		row := db.QueryRowContext(t.Context(), "SELECT user_id, kind, message FROM activities")
		var userID int
		var kind string
		var message string
		require.NoError(t, row.Scan(&userID, &kind, &message))
		assert.Equal(t, 1, userID)
		assert.Equal(t, "test", kind)
		assert.Equal(t, "test create", message)
	})

	t.Run("select activities", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		_, err := db.ExecContext(t.Context(), `INSERT INTO activities (id, user_id, kind, message, created_at) VALUES
			(1, 1, 'foo', 'foo', 100),
			(2, 1, 'bar', 'bar', 200),
			(3, 1, 'baz', 'baz', 200),
			(4, 2, 'hoge', 'hoge', 300),
			(5, 2, 'fuga', 'fuga', 300)`,
		)
		require.NoError(t, err)

		repo := NewActivityRepository()
		activities, err := repo.Select(t.Context(), db, 1)
		require.NoError(t, err)

		want := []Activity{
			{UserID: 1, Kind: "baz", Message: "baz", CreatedAt: 200},
			{UserID: 1, Kind: "bar", Message: "bar", CreatedAt: 200},
			{UserID: 1, Kind: "foo", Message: "foo", CreatedAt: 100},
		}
		assert.Equal(t, want, activities)
	})

	t.Run("select no activities", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewActivityRepository()
		activities, err := repo.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []Activity{}, activities)
	})
}
