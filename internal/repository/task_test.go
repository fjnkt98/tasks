package repository

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', 'digest'),
			(2, 'user2', 'digest')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO tasks (user_id, title, status, updated_at, created_at) VALUES
			(1, 'foo', 'created', 1136171045, 1136171045),
			(1, 'bar', 'done', 1136171045, 1136171045),
			(2, 'baz', 'created', 1136171045, 1136171045)`,
		)
		require.NoError(t, err)
	}

	t.Run("create", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewTaskRepository()

		task, err := repo.Create(t.Context(), db, 1, "test")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 4, UserID: 1, Title: "test", Status: "created"}, task)
	})

	t.Run("get", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewTaskRepository()

		tasks, err := repo.Select(t.Context(), db, 1, 100, 0)
		require.NoError(t, err)
		want := []Task{
			{ID: 1, UserID: 1, Title: "foo", Status: "created"},
			{ID: 2, UserID: 1, Title: "bar", Status: "done"},
		}
		assert.Equal(t, want, tasks)

		tasks, err = repo.SelectByStatus(t.Context(), db, 1, "done", 100, 0)
		require.NoError(t, err)
		want = []Task{
			{ID: 2, UserID: 1, Title: "bar", Status: "done"},
		}
		assert.Equal(t, want, tasks)

		task, err := repo.Get(t.Context(), db, 2, 3)
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 3, UserID: 2, Title: "baz", Status: "created"}, task)
	})

	t.Run("update", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewTaskRepository()

		// update title
		task, err := repo.UpdateTitle(t.Context(), db, 1, 1, "qux")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 1, UserID: 1, Title: "qux", Status: "created"}, task)

		// update status
		task, err = repo.UpdateStatus(t.Context(), db, 1, 2, "created")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 2, UserID: 1, Title: "bar", Status: "created"}, task)

		tasks, err := repo.Select(t.Context(), db, 1, 100, 0)
		require.NoError(t, err)
		want := []Task{
			{ID: 2, UserID: 1, Title: "bar", Status: "created"},
			{ID: 1, UserID: 1, Title: "qux", Status: "created"},
		}
		assert.Equal(t, want, tasks)

		// not found
		task, err = repo.UpdateTitle(t.Context(), db, 2, 1, "test")
		require.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, task)
	})

	t.Run("delete", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewTaskRepository()

		ok, err := repo.Delete(t.Context(), db, 1, 1)
		require.NoError(t, err)
		assert.True(t, ok)

		tasks, err := repo.Select(t.Context(), db, 1, 100, 0)
		require.NoError(t, err)
		want := []Task{
			{ID: 2, UserID: 1, Title: "bar", Status: "done"},
		}
		assert.Equal(t, want, tasks)

		// not found
		ok, err = repo.Delete(t.Context(), db, 2, 1)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("count", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewTaskRepository()

		count, err := repo.Count(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		count, err = repo.TodoCount(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		count, err = repo.DoneCount(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}
