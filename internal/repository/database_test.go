package repository

import (
	"testing"

	"github.com/fjnkt98/tasks/internal/testutil"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	dsn := "file::memory:"
	db, err := NewDB(t.Context(), dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})
}

func TestEnforceForeignKeys(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Value string
		Want  string
	}{
		{Name: "memory", Value: "file::memory:", Want: "file::memory:?_fk=1"},
		{Name: "file", Value: "file:app.db", Want: "file:app.db?_fk=1"},
		{Name: "disabled", Value: "file:app.db?_fk=0&_foreign_keys=0", Want: "file:app.db?_fk=1"},
		{Name: "other options", Value: "file:app.db?mode=rwc", Want: "file:app.db?_fk=1&mode=rwc"},
		{Name: "duplicated", Value: "file:app.db?_fk=0&_fk=0", Want: "file:app.db?_fk=1"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			got, err := EnforceForeignKeys(test.Value)
			require.NoError(t, err)
			assert.Equal(t, test.Want, got)
		})
	}

	for _, test := range []struct {
		Name  string
		Value string
	}{
		{Name: "invalid query", Value: "file:app.db?mode=%zz"},
		{Name: "invalid url", Value: "file:///app%zz.d"},
	} {
		t.Run(test.Name, func(t *testing.T) {
			_, err := EnforceForeignKeys(test.Value)
			require.Error(t, err)
		})
	}
}

func TestIsUniqueConstraintError(t *testing.T) {
	db := testutil.NewDB(t)

	_, err := db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('test', 'digest')")
	require.NoError(t, err)
	_, err = db.ExecContext(t.Context(), "INSERT INTO users (name, digest) VALUES ('test', 'digest')")
	assert.Error(t, err)
	assert.True(t, IsUniqueConstraintError(err))

	_, err = db.ExecContext(t.Context(), "INSERT INTO users (name) VALUES ('foo')")
	assert.Error(t, err)
	assert.False(t, IsUniqueConstraintError(err))
}
