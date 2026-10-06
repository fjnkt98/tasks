package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed schema.sql
var schema []byte

func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	file := filepath.Join(t.TempDir(), "test.db")
	cmd := exec.CommandContext(t.Context(), "sqlite3def", file, "--apply")
	cmd.Stdin = bytes.NewReader(schema)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3def failed: %v\n%s", err, out)
	}

	dsn, err := enforceForeignKeys(fmt.Sprintf("file:%s", file))
	require.NoError(t, err)

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	return db
}

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
			got, err := enforceForeignKeys(test.Value)
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
			_, err := enforceForeignKeys(test.Value)
			require.Error(t, err)
		})
	}
}
