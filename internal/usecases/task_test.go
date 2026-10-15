package usecases

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskUsecase(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', 'digest'),
			(2, 'user2', 'digest')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO tasks (user_id, title, status) VALUES
			(1, 'foo', 'created'),
			(1, 'bar', 'created'),
			(1, 'baz', 'done'),
			(2, 'qux', 'created')`,
		)
		require.NoError(t, err)
	}

	t.Run("list", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		for _, test := range []struct {
			Name   string
			Status string
			Want   []Task
		}{
			{Name: "all", Status: "", Want: []Task{
				{ID: 2, UserID: 1, Title: "bar", Status: "created"},
				{ID: 1, UserID: 1, Title: "foo", Status: "created"},
				{ID: 3, UserID: 1, Title: "baz", Status: "done"},
			}},
			{Name: "created", Status: "created", Want: []Task{
				{ID: 2, UserID: 1, Title: "bar", Status: "created"},
				{ID: 1, UserID: 1, Title: "foo", Status: "created"},
			}},
			{Name: "done", Status: "done", Want: []Task{
				{ID: 3, UserID: 1, Title: "baz", Status: "done"},
			}},
		} {
			t.Run(test.Name, func(t *testing.T) {
				tasks, err := u.List(t.Context(), 1, test.Status, 10, 1)
				require.NoError(t, err)
				assert.Equal(t, test.Want, tasks)
			})
		}
	})

	t.Run("pagination", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		// page 1
		tasks, err := u.List(t.Context(), 1, "", 2, 1)
		require.NoError(t, err)

		want := []Task{
			{ID: 2, UserID: 1, Title: "bar", Status: "created"},
			{ID: 1, UserID: 1, Title: "foo", Status: "created"},
		}
		assert.Equal(t, want, tasks)

		// page 2
		tasks, err = u.List(t.Context(), 1, "", 2, 2)
		require.NoError(t, err)

		want = []Task{
			{ID: 3, UserID: 1, Title: "baz", Status: "done"},
		}
		assert.Equal(t, want, tasks)

		// page 3
		tasks, err = u.List(t.Context(), 1, "", 2, 3)
		require.NoError(t, err)
		assert.Equal(t, []Task{}, tasks)
	})

	t.Run("get", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		task, err := u.Get(t.Context(), 1, 1)
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 1, UserID: 1, Title: "foo", Status: "created"}, task)
	})

	t.Run("add", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		task, err := u.Add(t.Context(), 1, "hoge")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 5, UserID: 1, Title: "hoge", Status: "created"}, task)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		activities, err = u.activities.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("update title", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		task, err := u.UpdateTitle(t.Context(), 1, 1, "new")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 1, UserID: 1, Title: "new", Status: "created"}, task)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		activities, err = u.activities.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("update status", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		task, err := u.UpdateStatus(t.Context(), 2, 4, "done")
		require.NoError(t, err)
		assert.Equal(t, &Task{ID: 4, UserID: 2, Title: "qux", Status: "done"}, task)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 0)

		activities, err = u.activities.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})

	t.Run("delete", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		err := u.Delete(t.Context(), 1, 2)
		require.NoError(t, err)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		activities, err = u.activities.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("count", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewTaskUsecase(db)

		for _, test := range []struct {
			Name   string
			Status string
			Want   int
		}{
			{Name: "all", Status: "", Want: 3},
			{Name: "created", Status: "created", Want: 2},
			{Name: "done", Status: "done", Want: 1},
		} {
			t.Run(test.Name, func(t *testing.T) {
				count, err := u.Count(t.Context(), 1, test.Status)
				require.NoError(t, err)
				assert.Equal(t, test.Want, count)
			})
		}
	})
}
