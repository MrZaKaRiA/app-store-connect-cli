package apps

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func captureAppsCreateOutput(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe error: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe error: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	stdoutCh := make(chan struct{}, 1)
	stderrCh := make(chan string, 1)
	go func() {
		_, _ = io.Copy(io.Discard, stdoutReader)
		stdoutCh <- struct{}{}
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrReader)
		stderrCh <- buf.String()
	}()

	closeWriters := func() {
		if stdoutWriter != nil {
			_ = stdoutWriter.Close()
			stdoutWriter = nil
		}
		if stderrWriter != nil {
			_ = stderrWriter.Close()
			stderrWriter = nil
		}
	}

	defer func() {
		closeWriters()
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	fn()

	closeWriters()
	os.Stdout = origStdout
	os.Stderr = origStderr

	<-stdoutCh
	return <-stderrCh
}
