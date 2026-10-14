package main

import (
	"bytes"
	"testing"
)

func TestRunWritesStartupMessage(t *testing.T) {
	var output bytes.Buffer

	if err := run(&output); err != nil {
		t.Fatalf("run() returned an error: %v", err)
	}

	const want = "NetArena Go foundation is ready.\n"
	if got := output.String(); got != want {
		t.Fatalf("run() output = %q, want %q", got, want)
	}
}
