//go:build !windows
// +build !windows

// Integration tests for the warproot command-line tool.
// These tests build the project binary and run it with safe flags that do not
// require root (for example --help). They run the binary from a temporary
// working directory so that the program's `latest.log` is created in an
// isolated location and can be inspected.

package main_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	buildTimeout = 60 * time.Second
	runTimeout   = 8 * time.Second
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	d, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working dir: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		pd := filepath.Dir(d)
		if pd == d {
			t.Fatalf("could not find go.mod upward from %s", d)
		}
		d = pd
	}
}

func buildBin(t *testing.T, out string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, ".")
	cmd.Dir = findRepoRoot(t)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\noutput:\n%s", err, string(b))
	}
}

func runBin(t *testing.T, bin string, args []string, dir string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	return outb.String(), errb.String(), err
}

func TestHelpLogsInfo(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "testbin")
	buildBin(t, bin)
	stdout, stderr, err := runBin(t, bin, []string{"--help"}, tmp)
	if err != nil {
		t.Fatalf("running --help failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	logp := filepath.Join(tmp, "latest.log")
	data, err := ioutil.ReadFile(logp)
	if err != nil {
		t.Fatalf("failed to read log file %s: %v", logp, err)
	}
	s := string(data)
	if !strings.Contains(s, "Program started") {
		t.Fatalf("expected log to contain Program started, got:\n%s", s)
	}
	if !strings.Contains(s, "Displayed help and exiting") {
		t.Fatalf("expected log to contain displayed help message, got:\n%s", s)
	}
}

func TestHelpWithWarningLogLevelSuppressesInfo(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "testbin")
	buildBin(t, bin)
	_, stderr, err := runBin(t, bin, []string{"--loglevel=warning", "--help"}, tmp)
	if err != nil {
		t.Fatalf("running --help with loglevel=warning failed: %v\nstderr:\n%s", err, stderr)
	}
	logp := filepath.Join(tmp, "latest.log")
	data, err := ioutil.ReadFile(logp)
	if err != nil {
		t.Fatalf("failed to read log file %s: %v", logp, err)
	}
	s := string(data)
	if !strings.Contains(s, "Program started") {
		t.Fatalf("expected log to contain Program started, got:\n%s", s)
	}
	if strings.Contains(s, "Displayed help and exiting") {
		t.Fatalf("did not expect displayed help message to be logged at warning level, got:\n%s", s)
	}
}

func TestMissingNewrootLogsWarningAndExits(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "testbin")
	buildBin(t, bin)
	_, stderr, err := runBin(t, bin, []string{}, tmp)
	if err == nil {
		t.Fatalf("expected non-zero exit when missing NEWROOT argument; stdout/stderr:\n%s", stderr)
	}
	// Expect exit code 1
	if ee, ok := err.(*exec.ExitError); ok {
		if ee.ExitCode() != 1 {
			t.Fatalf("expected exit status 1 when missing NEWROOT, got %d", ee.ExitCode())
		}
	} else {
		t.Fatalf("expected ExitError when missing NEWROOT, got: %v", err)
	}
	logp := filepath.Join(tmp, "latest.log")
	data, err := ioutil.ReadFile(logp)
	if err != nil {
		t.Fatalf("failed to read log file %s: %v", logp, err)
	}
	if !strings.Contains(string(data), "Missing NEWROOT argument, exiting") {
		t.Fatalf("expected warning about missing NEWROOT in log, got:\n%s", string(data))
	}
}
