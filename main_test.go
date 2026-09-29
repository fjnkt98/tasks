package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/slogtest"
)

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
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatal(err)
			}
			ms = append(ms, m)
		}
		return ms
	}

	if err := slogtest.TestHandler(h, results); err != nil {
		t.Error(err)
	}
}

func TestSetup(t *testing.T) {
	shutdown, err := setup(t.Context(), "localhost:4137", "test")
	defer shutdown() // nolint:errcheck
	if err != nil {
		t.Fatal(err)
	}
}
