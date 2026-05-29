# warproot

`warproot` is a small Go utility that provides a convenient wrapper around the
`chroot` system call. It allows you to change the root directory of a process
and optionally run a command inside that new root. The tool adds a number of
use‑friendly command‑line flags and logging capabilities that make it easier to
use in scripts and tests.

## Features

* **Root switching** – Performs a `chroot` to the specified directory.
* **User / group handling** – Resolve usernames and groups inside the new
  root and drop privileges accordingly (`--userspec` and `--groups`).
* **Mount `/proc`** – Optionally mount the proc filesystem inside the chroot
  (`--mount-proc`).
* **Environment control** – Clear the environment by default, with an option
  to preserve it (`-i` / `--preserve-environment`).
* **Custom command execution** – Run any command inside the chroot, either by
  passing it as positional arguments or via the `--cmd` flag.
* **Logging** – Adjustable log level (`--loglevel`) with structured output.
* **Help & version** – Built‑in `--help`/`-h` and `--version` flags.

The binary can be placed anywhere in your `$PATH`.

## Usage

```text
warproot [OPTIONS] NEWROOT [COMMAND [ARG]...]
```

### Options

| Flag | Description |
|------|-------------|
| `--userspec=USER[:GROUP]` | Specify the user (and optional group) to run the command as after the chroot. |
| `--groups=G_LIST` | Comma‑separated list of supplementary groups (requires `--userspec`). |
| `--skip-chdir` | Do not `chdir` to `/` after the chroot. |
| `--mount-proc` | Mount the proc filesystem inside the new root. |
| `-i`, `--preserve-environment` | Keep the current environment variables instead of clearing them. |
| `--loglevel=LEVEL` | Logging level – one of `fatal`, `panic`, `warning`, `info` (default `info`). |
| `--cmd=STRING` | Command string to execute inside the chroot (used with `--enter`). |
| `--enter` | After setting up the chroot, drop privileges and run the command. |
| `--target=DIR` | Alias for the positional `NEWROOT` argument. |
| `--help`, `-h` | Show help and exit. |
| `--version` | Show version information and exit. |

### Example

```bash
sudo ./warproot --userspec=www-data --mount-proc --loglevel=info /var/chroot /bin/bash
```

This will:

1. Change root to `/var/chroot`.
2. Mount `/proc` inside the new root.
3. Drop privileges to the `www-data` user.
4. Start an interactive Bash shell.

## License

This project is licensed under the GNU GPLv3 – see the `LICENSE` file for
details.

---
