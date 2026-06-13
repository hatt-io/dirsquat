# dirsquat

`dirsquat` is a small Go command-line tool for macOS and Linux. It is designed to run from `.zshrc`, `.bashrc`, or another shell startup file and report visible files that are older than a configured number of days.

By default, it scans `~/Downloads` and reports files older than 7 days. It scans recursively. Output uses a heavy terminal banner with `CLEAR`, `FOUND`, `WARN`, and `ERROR` states so it is easy to spot in a new shell window. Ages are based on each file's modification time, which is available on both macOS and Linux.

## What It Never Does

`dirsquat` only reports. It never moves, deletes, renames, archives, modifies, opens, or touches files. It does not run as a daemon, watch the filesystem, send desktop notifications, use a database, store state, or read a config file.

## Build And Install

`dirsquat` builds with Go 1.25 or newer.

Build locally:

```sh
go build -o dirsquat .
```

Build with an explicit version string:

```sh
go build -ldflags="-X main.version=0.1.0" -o dirsquat .
```

Install the built executable somewhere on your `PATH`:

```sh
install -m 0755 dirsquat /usr/local/bin/dirsquat
```

Or install directly with Go:

```sh
go install github.com/hatt-io/dirsquat@latest
```

## Usage

Run with defaults:

```sh
dirsquat
```

That is equivalent to:

```sh
dirsquat --days 7 ~/Downloads
```

Scan specific directories with the default 7-day cutoff:

```sh
dirsquat ~/Downloads ~/Desktop
```

Use a different day cutoff:

```sh
dirsquat --days 14 ~/Downloads ~/Desktop
```

By default, `dirsquat` uses count mode. Count mode shows the number of older files in each directory and the age of the oldest file found there:

```sh
dirsquat --count
dirsquat --days 7 --count ~/Downloads ~/Desktop
```

Example output:

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       FOUND ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ FILES  OLDEST FILE AGE   DIRECTORY                         ┃
┃ 12     3 months 2 days   /path/to/Downloads                ┃
┃ 2      18 days           /path/to/Desktop                  ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

If no files match:

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       CLEAR ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ No files need attention.                                    ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

Use names mode to print each matching file path with that file's age:

```sh
dirsquat --names
dirsquat --days 7 --names ~/Downloads ~/Desktop
```

Example output:

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       FOUND ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ FILE AGE        FILE                                        ┃
┃ 18 days         /path/to/Downloads/report.pdf               ┃
┃ 3 months 2 days /path/to/Desktop/archive.zip                ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

Paths with spaces work when passed as normal shell arguments:

```sh
dirsquat --days 14 --names "$HOME/Project Notes"
```

## Shell Startup

Add a command like this to `.zshrc`:

```sh
dirsquat
```

Or pass explicit directories:

```sh
dirsquat --days 7 ~/Downloads ~/Desktop
```

Add a command like this to `.bashrc`:

```sh
dirsquat
```

Missing or unreadable directories print a `WARN` panel and scanning continues. Argument errors, such as an invalid `--days` value, print an `ERROR` panel and fail clearly.

## Defaults

`dirsquat` defaults to:

```text
--days 7
DIR: ~/Downloads
```

Passing any directory argument replaces the default `~/Downloads` target.

## Modes

Count mode prints one line per directory that has matching files. Each line includes the age of the oldest matching file in that directory:

```sh
dirsquat --count
dirsquat --days 7 --count ~/Downloads ~/Desktop
```

If neither `--count` nor `--names` is passed, count mode is used.

Names mode prints each matching file path with that file's age:

```sh
dirsquat --names
dirsquat --days 7 --names ~/Downloads ~/Desktop
```

## Symlinked Directories

By default, `dirsquat` does not enter symlinked directories:

```sh
dirsquat ~/Downloads
```

Use `--follow-symlinks` to include symlinked directories:

```sh
dirsquat --follow-symlinks ~/Downloads
```

Directory loops reached through symlinks are skipped.

## Hidden Files And Directories

Files and directories with names beginning with `.` are hidden. `dirsquat` does not report hidden files and does not enter hidden directories.

## Options

```text
--days N             report files older than N days
--count              show counts and oldest age by directory
--names              show file paths and ages
--follow-symlinks    enter symlinked directories
--help               show help
--version            show version
```

## Development

Run the checks used by CI:

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o dirsquat .
```

CI runs those checks on Linux and macOS and cross-builds `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`.
