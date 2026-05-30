// SPDX-FileCopyrightText: 2026 AnmiTaliDev <anmitalidev@nuros.org>
// SPDX-FileCopyrightText: 2026 CosmoBlade <ilovesmetana777@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package user

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LookupUserGroupWith resolves a user and optional group using the provided
// passwd and group file paths. This is testable without touching the real
// /etc/passwd and /etc/group files.
func LookupUserGroupWith(passwdPath, groupPath, userSpec, groupSpec string) (uid, gid int, err error) {
	if u, err := strconv.Atoi(userSpec); err == nil {
		uid = u
	} else {
		uid, err = findUIDByNameWith(passwdPath, userSpec)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid user %q: %v", userSpec, err)
		}
	}

	if groupSpec == "" {
		gid, err = getPrimaryGIDWith(passwdPath, uid)
		if err != nil {
			return 0, 0, fmt.Errorf("cannot determine primary group for uid %d: %v", uid, err)
		}
	} else {
		if g, err := strconv.Atoi(groupSpec); err == nil {
			gid = g
		} else {
			gid, err = findGIDByNameWith(groupPath, groupSpec)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid group %q: %v", groupSpec, err)
			}
		}
	}

	return uid, gid, nil
}

// LookupGroupsWith resolves a comma-separated list of group names or IDs
// using the provided group file path.
func LookupGroupsWith(groupPath, groupList string) ([]int, error) {
	var gids []int
	for _, grp := range strings.Split(groupList, ",") {
		if g, err := strconv.Atoi(grp); err == nil {
			gids = append(gids, g)
		} else {
			g, err := findGIDByNameWith(groupPath, grp)
			if err != nil {
				return nil, fmt.Errorf("group %q not found", grp)
			}
			gids = append(gids, g)
		}
	}
	return gids, nil
}

// Convenience wrappers that use the system files.
func LookupUserGroup(userSpec, groupSpec string) (int, int, error) {
	return LookupUserGroupWith("/etc/passwd", "/etc/group", userSpec, groupSpec)
}

func LookupGroups(groupList string) ([]int, error) {
	return LookupGroupsWith("/etc/group", groupList)
}

// Internal helpers that operate on provided file paths.
func findUIDByNameWith(passwdPath, name string) (int, error) {
	f, err := os.Open(passwdPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == name {
			return strconv.Atoi(fields[2])
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("user not found")
}

func findGIDByNameWith(groupPath, name string) (int, error) {
	f, err := os.Open(groupPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[0] == name {
			return strconv.Atoi(fields[2])
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("group not found")
}

func getPrimaryGIDWith(passwdPath string, uid int) (int, error) {
	f, err := os.Open(passwdPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) >= 4 {
			if u, err := strconv.Atoi(fields[2]); err == nil && u == uid {
				return strconv.Atoi(fields[3])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("no passwd entry for uid %d", uid)
}
