package main

import (
	"bytes"
	"encoding/json/v2"
	"io"
	"log/slog"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	m.Run()
}

func TestTraceHandler(t *testing.T) {
	var buf bytes.Buffer
	h := &traceHandler{
		Handler:        slog.NewJSONHandler(&buf, nil),
		gcpProjectName: "test",
	}

	results := func() []map[string]any {
		var ms []map[string]any
		for line := range bytes.SplitSeq(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			require.NoError(t, json.Unmarshal(line, &m))
			ms = append(ms, m)
		}
		return ms
	}

	assert.NoError(t, slogtest.TestHandler(h, results))
}

func TestSetup(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	shutdown, err := setup(t.Context(), "localhost:4137", "test")
	require.NoError(t, err)
	defer shutdown() // nolint:errcheck
}
