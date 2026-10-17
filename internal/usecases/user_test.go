package usecases

import (
	"database/sql"
	"testing"
	"time"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserUsecase(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', ?),
			(2, 'user2', ?)`,
			string(getdigest()),
			string(getdigest()),
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO roles (name) VALUES
			('test')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO user_role_relations (user_id, role) VALUES
			(1, 'test')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO tasks (user_id, title, status) VALUES
			(1, 'foo', 'created'),
			(1, 'bar', 'done'),
			(1, 'baz', 'done')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO activities (user_id, kind, message, created_at) VALUES
			(1, 'account', 'activity 1', 1136084645),
			(1, 'account', 'activity 2', 1136171045),
			(1, 'account', 'activity 3', 1136257445)`,
		)
		require.NoError(t, err)
	}

	t.Run("list users", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		for _, test := range []struct {
			Name  string
			Page  int
			Limit int
			Want  *UserList
		}{
			{Name: "no next", Page: 1, Limit: 10, Want: &UserList{Users: []User{{ID: 1, Name: "user1"}, {ID: 2, Name: "user2"}}, HasNext: false}},
			{Name: "has next", Page: 1, Limit: 1, Want: &UserList{Users: []User{{ID: 1, Name: "user1"}}, HasNext: true}},
			{Name: "paginated", Page: 2, Limit: 1, Want: &UserList{Users: []User{{ID: 2, Name: "user2"}}, HasNext: false}},
			{Name: "no data", Page: 2, Limit: 2, Want: &UserList{Users: []User{}, HasNext: false}},
		} {
			t.Run(test.Name, func(t *testing.T) {
				u := NewUserUsecase(db)
				users, err := u.List(t.Context(), test.Page, test.Limit)
				require.NoError(t, err)
				assert.Equal(t, test.Want, users)
			})
		}
	})

	t.Run("get user", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		t.Run("success", func(t *testing.T) {
			u := NewUserUsecase(db)
			user, err := u.Get(t.Context(), 1)
			require.NoError(t, err)

			want := &UserDetail{
				User: User{
					ID:   1,
					Name: "user1",
				},
				TaskCount: 3,
				DoneCount: 2,
				Roles:     []string{"test"},
				Activities: []Activity{
					{UserID: 1, Kind: "account", Message: "activity 3", CreatedAt: time.Date(2006, 1, 3, 3, 4, 5, 0, time.UTC)},
					{UserID: 1, Kind: "account", Message: "activity 2", CreatedAt: time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)},
					{UserID: 1, Kind: "account", Message: "activity 1", CreatedAt: time.Date(2006, 1, 1, 3, 4, 5, 0, time.UTC)},
				},
			}
			assert.Equal(t, want, user)

			user, err = u.Get(t.Context(), 2)
			require.NoError(t, err)

			want = &UserDetail{
				User: User{
					ID:   2,
					Name: "user2",
				},
				TaskCount:  0,
				DoneCount:  0,
				Roles:      []string{},
				Activities: []Activity{},
			}
			assert.Equal(t, want, user)
		})

		t.Run("not found", func(t *testing.T) {
			u := NewUserUsecase(db)
			user, err := u.Get(t.Context(), 0)
			require.ErrorIs(t, err, ErrNotFound)
			assert.Nil(t, user)
		})
	})
}

func TestGrantRole(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', ?),
			(2, 'user2', ?)`,
			string(getdigest()),
			string(getdigest()),
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(2, 'token', 10000000000)`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO roles (name) VALUES
			('test')`,
		)
		require.NoError(t, err)
	}

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
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', ?),
			(2, 'user2', ?)`,
			string(getdigest()),
			string(getdigest()),
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO sessions (user_id, token, expires_at) VALUES
			(2, 'token', 10000000000)`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO roles (name) VALUES
			('test')`,
		)
		require.NoError(t, err)
	}

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
