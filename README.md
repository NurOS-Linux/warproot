# warproot

> [!NOTE]
> **Repository Mirrors**
>
> - **git.nuros.org** ([utils/warproot](https://git.nuros.org/utils/warproot)): primary, self-hosted Forgejo instance, accounts restricted to the core team.
> - **GitHub** ([NurOS-Linux/warproot](https://github.com/NurOS-Linux/warproot)): mirror for external contributors. Issues and Pull Requests opened here are welcome and are reviewed and processed by the core team.

A chroot utility.

## Usage

```
warproot [OPTIONS] NEWROOT [COMMAND [ARG]...]
```

If `COMMAND` is omitted, `/bin/sh` is started inside the new root.

## Options

| Flag | Description |
|---|---|
| `--userspec=USER[:GROUP]` | Run as the given user and optional group (name or numeric ID) |
| `--groups=G_LIST` | Set supplementary groups (comma-separated names or IDs) |
| `--skip-chdir` | Do not change working directory to `/` after chroot |
| `--mount-proc` | Mount the proc filesystem at `/proc` inside NEWROOT |
| `--preserve-environment` | Keep current environment variables instead of clearing them |
| `--loglevel=LEVEL` | Log verbosity: `fatal`, `warning`, or `info` (default: `info`) |
| `--version` | Print version and exit |
| `--help`, `-h` | Print help and exit |

## Logging

Each run truncates and rewrites `latest.log` in the working directory. Verbosity is controlled with `--loglevel`.

## Building

```sh
go build -o warproot .
```

Requires root privileges at runtime.

## License

GPL-3.0-only
