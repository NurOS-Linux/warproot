//go:build !windows
// +build !windows

package mount

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMountProcCreatesDirAndCallsRunCommand(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()

	procPath := filepath.Join(tmp, "proc")
	called := false
	var gotName string
	var gotArgs []string
	runCmd := func(name string, args ...string) error {
		called = true
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := MountProcWith(procPath, runCmd); err != nil {
		t.Fatalf("MountProcWith failed: %v", err)
	}

	if _, err := os.Stat(procPath); err != nil {
		t.Fatalf("expected procPath to exist, stat failed: %v", err)
	}

	if !called {
		t.Fatalf("expected runCmd to be called")
	}

	expectedArgs := []string{"-t", "proc", "proc", procPath}
	if gotName != "mount" || !reflect.DeepEqual(gotArgs, expectedArgs) {
		t.Fatalf("unexpected runCmd call: name=%s args=%v, want mount %v", gotName, gotArgs, expectedArgs)
	}
}
