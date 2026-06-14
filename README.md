# dirsquat

`dirsquat` reports visible files that are older than a chosen number of days. It is a small Go CLI for macOS and Linux, designed to run from `.zshrc`, `.bashrc`, or another shell startup file.

Default behavior:

```sh
dirsquat
```

That is equivalent to:

```sh
dirsquat --days 7 ~/Downloads
```

Ages are based on file modification time, which is available on both macOS and Linux.

## Safety

`dirsquat` only reports. It never moves, deletes, renames, archives, modifies, opens, or touches files.

It also does not run as a daemon, watch the filesystem, send desktop notifications, use a database, store state, read a config file, print JSON, or use color output.

## Install

Build locally:

```sh
go build -o dirsquat .
```

Install the binary somewhere on your `PATH`:

```sh
install -m 0755 dirsquat /usr/local/bin/dirsquat
```

Or install directly with Go:

```sh
go install github.com/hatt-io/dirsquat@latest
```

Build with an explicit version string:

```sh
go build -ldflags="-X main.version=0.1.0" -o dirsquat .
```

## Quick Usage

Scan the default directory with the default 7-day cutoff:

```sh
dirsquat
```

Scan specific directories:

```sh
dirsquat ~/Downloads ~/Desktop
```

Use a different day cutoff:

```sh
dirsquat --days 14 ~/Downloads ~/Desktop
```

Paths with spaces work as normal shell arguments:

```sh
dirsquat --days 14 --names "$HOME/Project Notes"
```

## Human Output

Count mode is the default. It shows one row per directory with older files:

```sh
dirsquat --count ~/Downloads ~/Desktop
```

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       FOUND ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ FILES  OLDEST FILE AGE   DIRECTORY                         ┃
┃ 12     3 months 2 days   /path/to/Downloads                ┃
┃ 2      18 days           /path/to/Desktop                  ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

Names mode shows each older file:

```sh
dirsquat --names ~/Downloads ~/Desktop
```

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       FOUND ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ FILE AGE        FILE                                        ┃
┃ 18 days         /path/to/Downloads/report.pdf               ┃
┃ 3 months 2 days /path/to/Desktop/archive.zip                ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

When no older files are found, human output confirms that clearly:

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       CLEAR ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ No files need attention.                                    ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

## Automation And Agents

Use `--plain` when another program or AI agent needs to consume the output. Plain mode is tab-separated, uses exact paths, and never shortens paths with an ellipsis.

Plain count mode:

```sh
dirsquat --plain --count ~/Downloads ~/Desktop
```

```text
12	92	/path/to/Downloads
2	18	/path/to/Desktop
```

Columns:

```text
file_count	oldest_file_age_days	directory
```

Plain names mode:

```sh
dirsquat --plain --names ~/Downloads ~/Desktop
```

```text
18	/path/to/Downloads/report.pdf
92	/path/to/Desktop/archive.zip
```

Columns:

```text
file_age_days	file_path
```

Plain mode writes no stdout when no older files are found. Scan warnings still go to stderr:

```text
WARN	/path/to/missing	lstat /path/to/missing: no such file or directory
```

## Shell Startup

Add this to `.zshrc` or `.bashrc`:

```sh
dirsquat
```

Or pass explicit directories:

```sh
dirsquat --days 7 ~/Downloads ~/Desktop
```

Missing or unreadable directories print a `WARN` panel and scanning continues. Argument errors print an `ERROR` panel and exit with code `2`.

## Scanning Rules

`dirsquat` scans recursively by default.

Files and directories with names beginning with `.` are hidden. Hidden files are not reported, and hidden directories are not entered.

Symlinked directories are not entered by default:

```sh
dirsquat ~/Downloads
```

Use `--follow-symlinks` to include symlinked directories:

```sh
dirsquat --follow-symlinks ~/Downloads
```

Directory loops reached through symlinks are skipped.

## Options

```text
--days N             report files older than N days
--count              show counts and oldest age by directory
--names              show file paths and ages
--plain              show tab-separated output for automation
--follow-symlinks    enter symlinked directories
--help               show help
--version            show version
```

If neither `--count` nor `--names` is passed, count mode is used.

Passing any directory argument replaces the default `~/Downloads` target.

## Development

Run the checks used by CI:

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o dirsquat .
```

CI runs those checks on Linux and macOS and cross-builds `linux/amd64`, `linux/arm64`, `darwin/amd64`, and `darwin/arm64`.
