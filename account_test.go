package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteAccount(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		err := CreateUser(t.Context(), db, "user1", "password")
		require.NoError(t, err)
		err = CreateUser(t.Context(), db, "user2", "password")
		require.NoError(t, err)

		_, err = Login(t.Context(), db, "user1", "password")
		require.NoError(t, err)
		_, err = Login(t.Context(), db, "user2", "password")
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), `INSERT INTO tasks (user_id, title, status) VALUES
			(1, 'test1', 'created'),
			(1, 'test2', 'done'),
			(2, 'test3', 'created')`,
		)
		require.NoError(t, err)
	}

	t.Run("user not found", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		err := DeleteAccount(t.Context(), db, 3, "password")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		var count int
		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 2, count)
	})

	t.Run("password not match", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		err := DeleteAccount(t.Context(), db, 1, "foo")
		require.ErrorIs(t, err, ErrInvalidCredentials)

		var count int
		row := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM users")
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 2, count)
	})

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)
		fixture(t, db)

		err := DeleteAccount(t.Context(), db, 1, "password")
		require.NoError(t, err)

		users := make([]string, 0)
		rows, err := db.QueryContext(t.Context(), "SELECT name FROM users")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				require.NoError(t, err)
			}
			users = append(users, name)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"user2"}, users)

		sessions := make([]int, 0)
		rows, err = db.QueryContext(t.Context(), "SELECT user_id FROM sessions")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var userID int
			if err := rows.Scan(&userID); err != nil {
				require.NoError(t, err)
			}
			sessions = append(sessions, userID)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []int{2}, sessions)

		tasks := make([]string, 0)
		rows, err = db.QueryContext(t.Context(), "SELECT title FROM tasks")
		require.NoError(t, err)
		defer rows.Close() // nolint:errcheck
		for rows.Next() {
			var title string
			if err := rows.Scan(&title); err != nil {
				require.NoError(t, err)
			}
			tasks = append(tasks, title)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []string{"test3"}, tasks)
	})
}

func TestGetMeHandler(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		err := CreateUser(t.Context(), db, "user1", "password")
		require.NoError(t, err)
	}

	db := NewTestDB(t)
	fixture(t, db)

	ctx := SetUserIntoContext(t.Context(), User{ID: 1, Name: "user1"})
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h := NewGetMeHandler(db)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
}

func TestGetAccountDeleteHandler(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h := NewGetAccountDeleteHandler()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
}

func TestPostAccountDeleteHandler(t *testing.T) {
	var fixture = func(t *testing.T, db *sql.DB) {
		err := CreateUser(t.Context(), db, "user1", "password")
		require.NoError(t, err)
	}

	db := NewTestDB(t)
	fixture(t, db)

	for _, test := range []struct {
		Name       string
		Body       url.Values
		ErrMessage string
	}{
		{Name: "password empty", Body: url.Values{"confirmation": []string{"delete account"}}, ErrMessage: "password is required"},
		{Name: "password too long", Body: url.Values{"password": []string{strings.Repeat("p", 73)}, "confirmation": []string{"delete account"}}, ErrMessage: "password too long"},
		{Name: "password not match", Body: url.Values{"password": []string{"foo"}, "confirmation": []string{"delete account"}}, ErrMessage: ErrInvalidCredentials.Error()},
		{Name: "confirmation not match", Body: url.Values{"password": []string{"password"}, "confirmation": []string{"foo"}}, ErrMessage: "confirmation is required"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			db := NewTestDB(t)

			fixture(t, db)

			ctx := SetUserIntoContext(t.Context(), User{ID: 1, Name: "user1"})
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/account/delete?password=foo&confirmation=bar", strings.NewReader(test.Body.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rec := httptest.NewRecorder()

			h := NewPostAccountDeleteHandler(db)
			h.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))

			body := rec.Body.String()
			assert.Contains(t, body, test.ErrMessage)
		})
	}

	t.Run("success", func(t *testing.T) {
		db := NewTestDB(t)

		fixture(t, db)

		values := url.Values{}
		values.Set("password", "password")
		values.Set("confirmation", "delete account")

		ctx := SetUserIntoContext(t.Context(), User{ID: 1, Name: "user1"})
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/account/delete?password=foo&confirmation=bar", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		h := NewPostAccountDeleteHandler(db)
		h.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
	})
}

func TestGetAccountDeleteSuccessHandler(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h := NewGetAccountDeleteSuccessHandler()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/html; charset=utf-8", rec.Result().Header.Get("Content-Type"))
}
