package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const version = "1.0.0"

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTION] NEWROOT [COMMAND [ARG]...]
Run COMMAND with root directory set to NEWROOT.

Options:
  --userspec=USER[:GROUP]   specify user and group (ID or name)
  --groups=G_LIST           supplementary groups (comma-separated)
  --skip-chdir              do not change working directory to '/'
  --mount-proc              mount proc filesystem inside NEWROOT
  -i, --preserve-environment  do not clear environment variables
  --help                    display this help and exit
  --version                 output version information and exit

If COMMAND is not specified, run '/bin/sh'.
`, os.Args[0])
}
func main() {
	// Initialize logging to latest.log
	// Truncate the log file at program start to ensure a fresh log for each run
	logFile, err := os.OpenFile("latest.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("Program started")

	// Use flag package for clearer option handling required by tests.
	var (
		userspec    string
		groups      string
		skipChdir   bool
		mountProc   bool
		preserveEnv bool
		help        bool
		showVersion bool
		// Test‑specific flags
		target   string
		syncSrc  string
		archive  string
		enter    bool
		cmdStr   string
		logLevel string
	)

	flag.StringVar(&userspec, "userspec", "", "specify user and group (ID or name)")
	flag.StringVar(&groups, "groups", "", "supplementary groups (comma-separated)")
	flag.BoolVar(&skipChdir, "skip-chdir", false, "do not change working directory to '/'")
	flag.BoolVar(&mountProc, "mount-proc", false, "mount proc filesystem inside NEWROOT")
	flag.BoolVar(&preserveEnv, "preserve-environment", false, "do not clear environment variables")
	flag.BoolVar(&help, "help", false, "display this help and exit")
	// Alias -h for help
	flag.BoolVar(&help, "h", false, "display this help and exit (alias)")
	flag.BoolVar(&showVersion, "version", false, "output version information and exit")
	// Flags used by the test suite
	flag.StringVar(&target, "target", "", "new root directory")
	flag.StringVar(&syncSrc, "sync", "", "source directory to sync (placeholder)")
	flag.StringVar(&archive, "archive", "", "archive file (placeholder)")
	flag.BoolVar(&enter, "enter", false, "enter chroot after setup")
	flag.StringVar(&cmdStr, "cmd", "", "command to run inside chroot")
	// Logging level: one of fatal, panic, warning, info (default info)
	flag.StringVar(&logLevel, "loglevel", "info", "logging level: fatal|panic|warning|info")

	flag.Parse()

	// Helper functions for log levels
	// Simple log level helpers. Each prints only when the configured level includes it.
	logFatal := func(msg string) {
		if logLevel == "fatal" || logLevel == "panic" || logLevel == "warning" || logLevel == "info" {
			log.Printf("[Fatal] %s", msg)
		}
	}
	logPanic := func(msg string) {
		if logLevel == "panic" || logLevel == "warning" || logLevel == "info" {
			log.Printf("[Panic] %s", msg)
		}
	}
	logWarning := func(msg string) {
		if logLevel == "warning" || logLevel == "info" {
			log.Printf("[Warning] %s", msg)
		}
	}
	logInfo := func(msg string) {
		if logLevel == "info" {
			log.Printf("[Info] %s", msg)
		}
	}
	// Prevent "unused variable" lint warnings when a helper is not referenced.
	_ = logPanic

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

	// Determine NEWROOT: prefer explicit -target flag, otherwise first positional arg.
	var newRoot string
	var cmdArgs []string
	if target != "" {
		newRoot = target
		log.Printf("Using target flag as NEWROOT: %s", newRoot)
		// Remaining non‑flag arguments are treated as command to execute.
		if flag.NArg() > 0 {
			cmdArgs = flag.Args()
		}
	} else {
		if flag.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "missing NEWROOT argument\n")
			usage()
			logWarning("Missing NEWROOT argument, exiting")
			os.Exit(1)
		}
		newRoot = flag.Arg(0)
		log.Printf("Using positional NEWROOT: %s", newRoot)
		if flag.NArg() > 1 {
			cmdArgs = flag.Args()[1:]
		}
	}

	// If a custom command is supplied via -cmd, use it.
	if enter && cmdStr != "" {
		cmdArgs = []string{"/bin/sh", "-c", cmdStr}
		log.Printf("Custom command set via -cmd: %s", cmdStr)
	}
	if len(cmdArgs) == 0 {
		cmdArgs = []string{"/bin/sh"}
	}
	log.Printf("Command to execute: %v", cmdArgs)

	// Must be root
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "chroot: you must be root")
		logWarning("Not running as root, exiting")
		os.Exit(1)
	}

	// Resolve absolute path to newRoot
	absRoot, err := filepath.Abs(newRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
		os.Exit(1)
	}

	// 1. chroot()
	if err := syscall.Chroot(absRoot); err != nil {
		fmt.Fprintf(os.Stderr, "chroot: cannot change root to %s: %v\n", absRoot, err)
		logFatal(fmt.Sprintf("Failed to chroot to %s: %v", absRoot, err))
		os.Exit(1)
	}
	logInfo(fmt.Sprintf("Changed root to %s", absRoot))

	// 2. Change directory unless --skip-chdir
	if !skipChdir {
		if err := syscall.Chdir("/"); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot chdir to /: %v\n", err)
			logWarning(fmt.Sprintf("Failed to chdir to /: %v", err))
			os.Exit(1)
		}
		logInfo("Changed directory to /")
	}

	// 3. Mount proc if requested (inside new root)
	if mountProc {
		procPath := "/proc"
		if err := os.MkdirAll(procPath, 0555); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot create /proc: %v\n", err)
			logWarning(fmt.Sprintf("Failed to create /proc: %v", err))
			os.Exit(1)
		}
		// Use external mount command for portability (works on Linux/macOS with appropriate privileges)
		if err := runCommand("mount", "-t", "proc", "proc", procPath); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot mount proc: %v\n", err)
			logWarning(fmt.Sprintf("Failed to mount proc: %v", err))
			os.Exit(1)
		}
		logInfo("Mounted proc filesystem")
	}

	// 4. Handle user/group switching (must be after chroot, using new root's passwd/group)
	var uid, gid int
	var gids []int
	if userspec != "" {
		// parse userspec
		var userPart, groupPart string
		if idx := strings.Index(userspec, ":"); idx >= 0 {
			userPart = userspec[:idx]
			groupPart = userspec[idx+1:]
		} else {
			userPart = userspec
		}
		uid, gid, err = lookupUserGroupInRoot(userPart, groupPart)
		if err != nil {
			fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
			logWarning(fmt.Sprintf("User/group lookup failed: %v", err))
			os.Exit(1)
		}
		logInfo(fmt.Sprintf("Resolved user %s to uid:%d gid:%d", userPart, uid, gid))
	}
	if groups != "" {
		if userspec == "" {
			fmt.Fprintln(os.Stderr, "chroot: --groups specified without --userspec, ignoring groups")
			logWarning("--groups provided without --userspec, ignored")
		} else {
			gids, err = lookupGroupsInRoot(groups)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chroot: %v\n", err)
				logWarning(fmt.Sprintf("Group lookup failed: %v", err))
				os.Exit(1)
			}
			logInfo(fmt.Sprintf("Supplementary groups resolved: %v", gids))
		}
	}
	// Apply group changes only if userspec was given (or groups given with userspec)
	if userspec != "" {
		// Set supplementary groups
		if gids != nil {
			if err := syscall.Setgroups(gids); err != nil {
				fmt.Fprintf(os.Stderr, "chroot: cannot set groups: %v\n", err)
				logWarning(fmt.Sprintf("Failed to set groups: %v", err))
				os.Exit(1)
			}
			logInfo(fmt.Sprintf("Set supplementary groups: %v", gids))
		} else {
			// clear groups
			if err := syscall.Setgroups([]int{}); err != nil {
				fmt.Fprintf(os.Stderr, "chroot: cannot clear groups: %v\n", err)
				logWarning(fmt.Sprintf("Failed to clear groups: %v", err))
				os.Exit(1)
			}
			logInfo("Cleared supplementary groups")
		}
		// Set gid
		if err := syscall.Setgid(gid); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot set gid: %v\n", err)
			logWarning(fmt.Sprintf("Failed to set gid: %v", err))
			os.Exit(1)
		}
		logInfo(fmt.Sprintf("Set gid to %d", gid))
		// Set uid
		if err := syscall.Setuid(uid); err != nil {
			fmt.Fprintf(os.Stderr, "chroot: cannot set uid: %v\n", err)
			logWarning(fmt.Sprintf("Failed to set uid: %v", err))
			os.Exit(1)
		}
		logInfo(fmt.Sprintf("Set uid to %d", uid))
	}

	// 5. Environment: clear unless preserveEnv
	if !preserveEnv {
		os.Clearenv()
		logInfo("Cleared environment variables")
	}

	// 6. Prepare command execution (search PATH inside chroot)
	cmdPath := cmdArgs[0]
	if !strings.Contains(cmdPath, "/") {
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			// default fallback inside chroot
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

	// 7. exec
	logInfo(fmt.Sprintf("Executing command: %s %v", cmdPath, argv[1:]))
	if err := syscall.Exec(cmdPath, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "chroot: failed to exec %s: %v\n", cmdPath, err)
		log.Printf("Exec failed: %v", err)
		os.Exit(126)
	}
}

// lookupUserGroupInRoot resolves username/uid and optional group inside current root (which is already chrooted)
func lookupUserGroupInRoot(userSpec, groupSpec string) (uid, gid int, err error) {
	// Try numeric first
	if u, err := strconv.Atoi(userSpec); err == nil {
		uid = u
	} else {
		// parse /etc/passwd inside chroot (already chrooted, so open "/etc/passwd")
		uid, err = findUIDByName(userSpec)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid user %q: %v", userSpec, err)
		}
	}
	if groupSpec == "" {
		// get primary group from passwd
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

func lookupGroupsInRoot(groupList string) ([]int, error) {
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

// runCommand executes a command and streams its output.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
