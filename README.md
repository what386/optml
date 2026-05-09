# optml

**OPTML (OPT Manager for Linux)** is a lightweight `/opt` manager for installing, tracking, and exposing manually installed programs.

`optml` manages package metadata in `/opt/optml/metadata.json`, executable symlinks in `/opt/optml/bin`, and shell PATH integration through `/opt/optml/paths.sh`.

## Features

- Add programs from raw files, directories, or archives into `/opt/<name>`
- Detect executables and expose them via symlinks in `/opt/optml/bin`
- Fallback PATH directory registration (`bin`, then `sbin`) when no executables are detected
- Refresh metadata from existing `/opt/*` entries
- Inspect package metadata and run integrity diagnostics
- Manage shell hooks (`hooks init`, `hooks clean`)

## Requirements

- Linux
- Root privileges for system-changing commands (`add`, `remove`, `refresh`, `doctor --fix`)

## Build

```bash
go build ./...
```

## Quick Start

```bash
# add shell hooks to your profiles
sudo optml hooks init

# install a program into /opt/mytool
sudo optml add mytool ./mytool.tar.gz

# list tracked entries
optml list

# show metadata for one entry
optml info mytool
```

## Commands

### `optml add <name> <source>`

Install a program into `/opt/<name>` and update integration artifacts.

### `optml remove <item>`

Remove `/opt/<item>`, metadata entry, and related integration references.

### `optml refresh`

Auto-discover entries already present in `/opt` and rewrite metadata.

### `optml list`

List tracked programs.

### `optml info <item>`

Print JSON metadata for a tracked program.

### `optml doctor [--fix]`

Run integrity checks across metadata, install roots, checksums, symlinks, and shell integration.

- `--fix`: refresh metadata and rebuild integration artifacts.

### `optml hooks init`

Add shell hook lines (bash/zsh/fish) to source `/opt/optml/paths.sh`.

### `optml hooks clean`

Remove shell hook lines added by `optml hooks init`.

## Paths Used

- Metadata: `/opt/optml/metadata.json`
- Shell PATH file: `/opt/optml/paths.sh`
- Managed symlink directory: `/opt/optml/bin`
- Program installs: `/opt/<name>`

## Notes

- `paths.sh` always includes `/opt/optml/bin`.
- Per-package PATH additions are only used as fallback when no symlinkable executables are found.
