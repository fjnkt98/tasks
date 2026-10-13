package testutil

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fjnkt98/tasks/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NewDBFile(t *testing.T) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "test.db")
	cmd := exec.CommandContext(t.Context(), "sqlite3def", file, "--apply")
	cmd.Stdin = bytes.NewReader(repository.Schema)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3def failed: %v\n%s", err, out)
	}

	return file
}

func NewDB(t *testing.T) *sql.DB {
	t.Helper()

	file := NewDBFile(t)

	dsn, err := repository.EnforceForeignKeys(fmt.Sprintf("file:%s", file))
	require.NoError(t, err)

	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	return db
}
