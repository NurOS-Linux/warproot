//go:build !windows
// +build !windows

package user

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestLookupUserGroupNumeric(t *testing.T) {
	// safe to run in parallel because it does not modify package state
	t.Parallel()
	uid, gid, err := LookupUserGroup("1000", "1001")
	if err != nil {
		t.Fatalf("LookupUserGroup numeric failed: %v", err)
	}
	if uid != 1000 || gid != 1001 {
		t.Fatalf("expected uid=1000 gid=1001 got uid=%d gid=%d", uid, gid)
	}
}

func TestLookupUserGroupByName(t *testing.T) {
	// safe to run in parallel because we pass explicit file paths
	t.Parallel()
	tmp := t.TempDir()
	pw := filepath.Join(tmp, "passwd")
	grp := filepath.Join(tmp, "group")
	writeFile(t, pw, "alice:x:1500:1600:Alice:/home/alice:/bin/sh\n")
	writeFile(t, grp, "developers:x:1600:\n")

	uid, gid, err := LookupUserGroupWith(pw, grp, "alice", "")
	if err != nil {
		t.Fatalf("LookupUserGroupWith by name failed: %v", err)
	}
	if uid != 1500 || gid != 1600 {
		t.Fatalf("expected uid=1500 gid=1600 got uid=%d gid=%d", uid, gid)
	}
}

func TestLookupGroups(t *testing.T) {
	// safe to run in parallel because we pass explicit file paths
	t.Parallel()
	tmp := t.TempDir()
	grp := filepath.Join(tmp, "group")
	writeFile(t, grp, "dev:x:3001:\nstaff:x:1000:\n")

	gids, err := LookupGroupsWith(grp, "3001,staff")
	if err != nil {
		t.Fatalf("LookupGroupsWith failed: %v", err)
	}
	if !reflect.DeepEqual(gids, []int{3001, 1000}) {
		t.Fatalf("expected gids [3001 1000], got %v", gids)
	}
}
