package repository

import (
	"database/sql"
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		t.Helper()

		_, err := db.ExecContext(t.Context(), `INSERT INTO users (id, name, digest) VALUES
			(1, 'user1', 'digest'),
			(2, 'user2', 'digest')`,
		)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO roles (name) VALUES ('admin')`)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO user_role_relations (user_id, role) VALUES (1, 'admin')`)
		require.NoError(t, err)
	}

	t.Run("create user", func(t *testing.T) {
		db := testutil.NewDB(t)

		repo := NewUserRepository()
		user, err := repo.Create(t.Context(), db, "test", []byte("digest"))

		require.NoError(t, err)
		assert.Equal(t, 1, user.ID)
		assert.Equal(t, "test", user.Name)
		assert.Equal(t, "digest", user.Digest)
	})

	t.Run("get existing user by id", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		for _, test := range []struct {
			Name string
			Key  int
			Want *User
		}{
			{Name: "1", Key: 1, Want: &User{ID: 1, Name: "user1", Digest: "digest"}},
			{Name: "2", Key: 2, Want: &User{ID: 2, Name: "user2", Digest: "digest"}},
		} {
			t.Run(test.Name, func(t *testing.T) {
				user, err := repo.GetByID(t.Context(), db, test.Key)

				require.NoError(t, err)
				assert.Equal(t, test.Want, user)
			})
		}
	})

	t.Run("get non-existent user by id", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()
		user, err := repo.GetByID(t.Context(), db, 0)

		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, user)
	})

	t.Run("get existing user by name", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		for _, test := range []struct {
			Name string
			Key  string
			Want *User
		}{
			{Name: "1", Key: "user1", Want: &User{ID: 1, Name: "user1", Digest: "digest"}},
			{Name: "2", Key: "user2", Want: &User{ID: 2, Name: "user2", Digest: "digest"}},
		} {
			t.Run(test.Name, func(t *testing.T) {
				user, err := repo.GetByName(t.Context(), db, test.Key)

				require.NoError(t, err)
				assert.Equal(t, test.Want, user)
			})
		}
	})

	t.Run("get non-existent user by name", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()
		user, err := repo.GetByName(t.Context(), db, "foo")

		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, user)
	})

	t.Run("select users", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		want := []User{
			{ID: 1, Name: "user1", Digest: "digest"},
			{ID: 2, Name: "user2", Digest: "digest"},
		}
		assert.Equal(t, want, users)
	})

	t.Run("select no users", func(t *testing.T) {
		db := testutil.NewDB(t)

		repo := NewUserRepository()

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		assert.Equal(t, []User{}, users)
	})

	t.Run("update existing user's digest", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.UpdateDigest(t.Context(), db, 1, []byte("new digest"))
		require.NoError(t, err)
		assert.True(t, ok)

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		want := []User{
			{ID: 1, Name: "user1", Digest: "new digest"},
			{ID: 2, Name: "user2", Digest: "digest"},
		}
		assert.Equal(t, want, users)
	})

	t.Run("update non-existent user's digest", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.UpdateDigest(t.Context(), db, 0, []byte("new digest"))
		require.NoError(t, err)
		assert.False(t, ok)

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		want := []User{
			{ID: 1, Name: "user1", Digest: "digest"},
			{ID: 2, Name: "user2", Digest: "digest"},
		}
		assert.Equal(t, want, users)
	})

	t.Run("delete existing user", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.Delete(t.Context(), db, 1)
		require.NoError(t, err)
		assert.True(t, ok)

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		want := []User{
			{ID: 2, Name: "user2", Digest: "digest"},
		}
		assert.Equal(t, want, users)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.Delete(t.Context(), db, 0)
		require.NoError(t, err)
		assert.False(t, ok)

		users, err := repo.SelectUsers(t.Context(), db, 100, 0)
		require.NoError(t, err)

		want := []User{
			{ID: 1, Name: "user1", Digest: "digest"},
			{ID: 2, Name: "user2", Digest: "digest"},
		}
		assert.Equal(t, want, users)
	})

	t.Run("select roles", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		roles, err := repo.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []Role{{Name: "admin"}}, roles)

		roles, err = repo.SelectRoles(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Equal(t, []Role{}, roles)
	})

	t.Run("add role", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		err := repo.AddRole(t.Context(), db, 2, Role{Name: "admin"})
		require.NoError(t, err)

		roles, err := repo.SelectRoles(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Equal(t, []Role{{Name: "admin"}}, roles)
	})

	t.Run("delete role", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.DeleteRole(t.Context(), db, 1, Role{Name: "admin"})
		require.NoError(t, err)
		assert.True(t, ok)

		roles, err := repo.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []Role{}, roles)
	})

	t.Run("delete non-existent role", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.DeleteRole(t.Context(), db, 2, Role{Name: "admin"})
		require.NoError(t, err)
		assert.False(t, ok)

		roles, err := repo.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []Role{{Name: "admin"}}, roles)

		roles, err = repo.SelectRoles(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Equal(t, []Role{}, roles)
	})

	t.Run("delete non-existent user's role", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		repo := NewUserRepository()

		ok, err := repo.DeleteRole(t.Context(), db, 0, Role{Name: "admin"})
		require.NoError(t, err)
		assert.False(t, ok)

		roles, err := repo.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []Role{{Name: "admin"}}, roles)

		roles, err = repo.SelectRoles(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Equal(t, []Role{}, roles)
	})
}
