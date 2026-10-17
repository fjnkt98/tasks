package usecases

import (
	"database/sql"
	"sync"
	"testing"

	"github.com/fjnkt98/tasks/internal/repository"
	"github.com/fjnkt98/tasks/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var getdigest = sync.OnceValue(func() []byte {
	digest, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return digest
})

func fixture(t *testing.T, db *sql.DB) {
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

func TestSignup(t *testing.T) {
	t.Run("user already exists", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Signup(t.Context(), "user1", "password")
		require.ErrorIs(t, err, ErrUserAlreadyExists)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)

		u := NewAccountUsecase(db)
		err := u.Signup(t.Context(), "user1", "password")
		require.NoError(t, err)

		userRepo := repository.NewUserRepository()
		user, err := userRepo.GetByName(t.Context(), db, "user1")
		require.NoError(t, err)
		assert.Equal(t, 1, user.ID)
	})
}

func TestSignin(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		_, err := u.Signin(t.Context(), "foo", "password")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("password not match", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		_, err := u.Signin(t.Context(), "user1", "foo")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		token, err := u.Signin(t.Context(), "user1", "password")
		require.NoError(t, err)

		sessionRepo := repository.NewSessionRepository()
		session, err := sessionRepo.GetByToken(t.Context(), db, token)
		require.NoError(t, err)
		assert.Equal(t, 1, session.UserID)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}

func TestChangePassword(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.ChangePassword(t.Context(), 0, "foo", "bar")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("password not match", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.ChangePassword(t.Context(), 1, "foo", "bar")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 0)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.ChangePassword(t.Context(), 2, "password", "new_password")
		require.NoError(t, err)

		userRepo := repository.NewUserRepository()
		user, err := userRepo.Get(t.Context(), db, 2)
		require.NoError(t, err)
		assert.NotEqual(t, string(getdigest()), user.Digest)

		sessionRepo := repository.NewSessionRepository()
		session, err := sessionRepo.GetByToken(t.Context(), db, "token")
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, session)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}

func TestSignout(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Signout(t.Context(), "foo")
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Signout(t.Context(), "token")
		require.NoError(t, err)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})

	t.Run("signout again", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Signout(t.Context(), "token")
		require.NoError(t, err)

		a := repository.NewActivityRepository()
		activities, err := a.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		err = u.Signout(t.Context(), "token")
		require.NoError(t, err)

		activities, err = a.Select(t.Context(), db, 2)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}

func TestGrantRole(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.GrantRole(t.Context(), 0, "test")
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)

		err := u.GrantRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err := u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []repository.Role{{Name: "test"}}, roles)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		// grant again (no effect)
		err = u.GrantRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err = u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []repository.Role{{Name: "test"}}, roles)

		activities, err = u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 2)
	})
}

func TestRevokeRole(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.RevokeRole(t.Context(), 0, "test")
		require.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		_, err := db.ExecContext(t.Context(), `INSERT INTO user_role_relations (user_id, role) VALUES (1, 'test')`)
		require.NoError(t, err)

		u := NewAccountUsecase(db)
		err = u.RevokeRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err := u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []repository.Role{}, roles)

		activities, err := u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)

		// revoke again
		err = u.RevokeRole(t.Context(), 1, "test")
		require.NoError(t, err)

		roles, err = u.users.SelectRoles(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Equal(t, []repository.Role{}, roles)

		activities, err = u.activities.Select(t.Context(), db, 1)
		require.NoError(t, err)
		assert.Len(t, activities, 1)
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Delete(t.Context(), 0, "password")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("password not match", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Delete(t.Context(), 2, "foo")
		require.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("success", func(t *testing.T) {
		db := testutil.NewDB(t)
		fixture(t, db)

		u := NewAccountUsecase(db)
		err := u.Delete(t.Context(), 2, "password")
		require.NoError(t, err)

		user, err := u.users.Get(t.Context(), db, 2)
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, user)

		session, err := u.sessions.GetByToken(t.Context(), db, "token")
		require.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, session)
	})
}
