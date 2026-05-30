// SPDX-FileCopyrightText: 2026 AnmiTaliDev <anmitalidev@nuros.org>
// SPDX-FileCopyrightText: 2026 CosmoBlade <ilovesmetana777@gmail.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"warproot/src/mount"
	"warproot/src/user"
)

const version = "1.0.0"

// devNull implements io.Writer that discards everything. Used when logging is disabled.
type devNull struct{}

func (d *devNull) Write(p []byte) (int, error) { return len(p), nil }

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTION] NEWROOT [COMMAND [ARG]...]
Run COMMAND with root directory set to NEWROOT.

Options:
  --userspec=USER[:GROUP]   specify user and group (ID or name)
  --groups=G_LIST           supplementary groups (comma-separated)
  --skip-chdir              do not change working directory to '/'
  --mount-proc              mount proc filesystem inside NEWROOT
  --preserve-environment    do not clear environment variables
  --loglevel=LEVEL          log verbosity: null|err|warning|info|debug (default: info)
  --help                    display this help and exit
  --version                 output version information and exit

If COMMAND is not specified, run '/bin/sh'.
`, os.Args[0])
}

func main() {
	var (
		userspec    string
		groups      string
		skipChdir   bool
		mountProc   bool
		preserveEnv bool
		help        bool
		showVersion bool
		target      string
		syncSrc     string
		archive     string
		enter       bool
		cmdStr      string
		logLevel    string
	)

	flag.StringVar(&userspec, "userspec", "", "specify user and group (ID or name)")
	flag.StringVar(&groups, "groups", "", "supplementary groups (comma-separated)")
	flag.BoolVar(&skipChdir, "skip-chdir", false, "do not change working directory to '/'")
	flag.BoolVar(&mountProc, "mount-proc", false, "mount proc filesystem inside NEWROOT")
	flag.BoolVar(&preserveEnv, "preserve-environment", false, "do not clear environment variables")
	flag.BoolVar(&help, "help", false, "display this help and exit")
	flag.BoolVar(&help, "h", false, "display this help and exit (alias)")
	flag.BoolVar(&showVersion, "version", false, "output version information and exit")
	flag.StringVar(&target, "target", "", "new root directory")
	flag.StringVar(&syncSrc, "sync", "", "source directory to sync (placeholder)")
	flag.StringVar(&archive, "archive", "", "archive file (placeholder)")
	flag.BoolVar(&enter, "enter", false, "enter chroot after setup")
	flag.StringVar(&cmdStr, "cmd", "", "command to run inside chroot")
	flag.StringVar(&logLevel, "loglevel", "info", "logging level: null|err|warning|info|debug")

	flag.Parse()

	// Logging levels
	const (
		levelNull = iota
		levelError
		levelWarning
		levelInfo
		levelDebug
	)
	var level int
	switch strings.ToLower(logLevel) {
	case "", "info":
		level = levelInfo
	case "warning", "warn":
		level = levelWarning
	case "err", "error", "fatal", "panic":
		level = levelError
	case "debug":
		level = levelDebug
	case "null", "none":
		level = levelNull
	default:
		level = levelInfo
	}

	// Open log file only if not null
	var logFile *os.File
	if level != levelNull {
		lf, err := os.OpenFile("latest.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			os.Exit(1)
		}
		logFile = lf
		log.SetOutput(logFile)
		defer logFile.Close()
	} else {
		// discard logs (do not create latest.log)
		log.SetOutput(new(devNull))
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// helper logging functions
	logDebug := func(format string, args ...interface{}) {
		if level >= levelDebug {
			log.Printf("[Debug] "+format, args...)
		}
	}
	logInfo := func(format string, args ...interface{}) {
		if level >= levelInfo {
			log.Printf("[Info] "+format, args...)
		}
	}
	logWarn := func(format string, args ...interface{}) {
		if level >= levelWarning {
			log.Printf("[Warning] "+format, args...)
		}
	}
	logErr := func(format string, args ...interface{}) {
		if level >= levelError {
			log.Printf("[Error] "+format, args...)
		}
	}

	// Program start
	logWarn("Program started")

	if help {
		usage()
		logInfo("Displayed help and exiting")
		os.Exit(0)
	}
	if showVersion {
		fmt.Printf("%s (go-chroot) %s\n", os.Args[0], version)
		logInfo("Displayed version and exiting")
		os.Exit(0)
	}

	var newRoot string
	var cmdArgs []string
	if target != "" {
		newRoot = target
		logDebug("Using target flag as NEWROOT: %s", newRoot)
		if flag.NArg() > 0 {
			cmdArgs = flag.Args()
		}
	} else {
		if flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "missing NEWROOT argument\n")
			usage()
			logWarn("Missing NEWROOT argument, exiting")
			os.Exit(1)
		}
		newRoot = flag.Arg(0)
		logDebug("Using positional NEWROOT: %s", newRoot)
		if flag.NArg() > 1 {
			cmdArgs = flag.Args()[1:]
		}
	}

	if enter && cmdStr != "" {
		cmdArgs = []string{"/bin/sh", "-c", cmdStr}
		logDebug("Custom command set via -cmd: %s", cmdStr)
	}
	if len(cmdArgs) == 0 {
		cmdArgs = []string{"/bin/sh"}
	}
	logDebug("Command to execute: %v", cmdArgs)

	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "chroot: you must be root")
		logWarn("Not running as root, exiting")
		os.Exit(1)
	}

	absRoot, err := filepath.Abs(newRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
		os.Exit(1)
	}

	if err := syscall.Chroot(absRoot); err != nil {
		fmt.Fprintf(os.Stderr, "chroot: cannot change root to %s: %v\n", absRoot, err)
		logErr("Failed to chroot to %s: %v", absRoot, err)
		os.Exit(1)
	}
	logInfo("Changed root to %s", absRoot)

	if !skipChdir {
		if err := syscall.Chdir("/"); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot chdir to /: %v\n", err)
			logWarn("Failed to chdir to /: %v", err)
			os.Exit(1)
		}
		logInfo("Changed directory to /")
	}

	if mountProc {
		if err := mount.MountProc(); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
			logWarn("Failed to mount proc: %v", err)
			os.Exit(1)
		}
		logInfo("Mounted proc filesystem")
	}

	var uid, gid int
	var gids []int
	if userspec != "" {
		var userPart, groupPart string
		if idx := strings.Index(userspec, ":"); idx >= 0 {
			userPart = userspec[:idx]
			groupPart = userspec[idx+1:]
		} else {
			userPart = userspec
		}
		uid, gid, err = user.LookupUserGroup(userPart, groupPart)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
			logWarn("User/group lookup failed: %v", err)
			os.Exit(1)
		}
		logInfo("Resolved user %s to uid:%d gid:%d", userPart, uid, gid)
	}
	if groups != "" {
		if userspec == "" {
			fmt.Fprintln(os.Stderr, "chroot: --groups specified without --userspec, ignoring groups")
			logWarn("--groups provided without --userspec, ignored")
		} else {
			gids, err = user.LookupGroups(groups)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
				logWarn("Group lookup failed: %v", err)
				os.Exit(1)
			}
			logInfo("Supplementary groups resolved: %v", gids)
		}
	}

	if userspec != "" {
		if gids != nil {
			if err := syscall.Setgroups(gids); err != nil {
				fmt.Fprintf(os.Stderr, "chroot: cannot set groups: %v\n", err)
				logWarn("Failed to set groups: %v", err)
				os.Exit(1)
			}
			logInfo("Set supplementary groups: %v", gids)
		} else {
			if err := syscall.Setgroups([]int{}); err != nil {
				fmt.Fprintf(os.Stderr, "chroot: cannot clear groups: %v\n", err)
				logWarn("Failed to clear groups: %v", err)
				os.Exit(1)
			}
			logInfo("Cleared supplementary groups")
		}
		if err := syscall.Setgid(gid); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot set gid: %v\n", err)
			logWarn("Failed to set gid: %v", err)
			os.Exit(1)
		}
		logInfo("Set gid to %d", gid)
		if err := syscall.Setuid(uid); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot set uid: %v\n", err)
			logWarn("Failed to set uid: %v", err)
			os.Exit(1)
		}
		logInfo("Set uid to %d", uid)
	}

	if !preserveEnv {
		os.Clearenv()
		logInfo("Cleared environment variables")
	}

	cmdPath := cmdArgs[0]
	if !strings.Contains(cmdPath, "/") {
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			pathEnv = "/bin:/usr/bin"
		}
		found := false
		for _, dir := range strings.Split(pathEnv, ":") {
			full := filepath.Join(dir, cmdPath)
			if _, err := os.Stat(full); err == nil {
				cmdPath = full
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "chroot: %s: command not found\n", cmdArgs[0])
			os.Exit(127)
		}
	}
	argv := make([]string, len(cmdArgs))
	copy(argv, cmdArgs)
	argv[0] = cmdPath

	logInfo("Executing command: %s %v", cmdPath, argv[1:])
	if err := syscall.Exec(cmdPath, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "chroot: failed to exec %s: %v\n", cmdPath, err)
		logErr("Exec failed: %v", err)
		os.Exit(126)
	}

	_ = syncSrc
	_ = archive
}
