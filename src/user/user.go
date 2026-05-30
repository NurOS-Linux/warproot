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

// LookupUserGroup resolves a user and optional group inside the current root.
// Called after chroot, so /etc/passwd and /etc/group refer to the new root.
func LookupUserGroup(userSpec, groupSpec string) (uid, gid int, err error) {
	if u, err := strconv.Atoi(userSpec); err == nil {
		uid = u
	} else {
		uid, err = findUIDByName(userSpec)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid user %q: %v", userSpec, err)
		}
	}

	if groupSpec == "" {
		gid, err = getPrimaryGID(uid)
		if err != nil {
			return 0, 0, fmt.Errorf("cannot determine primary group for uid %d: %v", uid, err)
		}
	} else {
		if g, err := strconv.Atoi(groupSpec); err == nil {
			gid = g
		} else {
			gid, err = findGIDByName(groupSpec)
			if err != nil {
				return 0, 0, fmt.Errorf("invalid group %q: %v", groupSpec, err)
			}
		}
	}

	return uid, gid, nil
}

// LookupGroups resolves a comma-separated list of group names or IDs.
func LookupGroups(groupList string) ([]int, error) {
	var gids []int
	for _, grp := range strings.Split(groupList, ",") {
		if g, err := strconv.Atoi(grp); err == nil {
			gids = append(gids, g)
		} else {
			g, err := findGIDByName(grp)
			if err != nil {
				return nil, fmt.Errorf("group %q not found", grp)
			}
			gids = append(gids, g)
		}
	}
	return gids, nil
}

func findUIDByName(name string) (int, error) {
	f, err := os.Open("/etc/passwd")
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

func findGIDByName(name string) (int, error) {
	f, err := os.Open("/etc/group")
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

func getPrimaryGID(uid int) (int, error) {
	f, err := os.Open("/etc/passwd")
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
