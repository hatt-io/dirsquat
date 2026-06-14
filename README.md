# dirsquat

`dirsquat` is a shell-startup reminder for old files. Run it from `.zshrc`, `.bashrc`, or another shell startup file and it reports files older than your cutoff.

By default, it checks `~/Downloads` for files older than 7 days:

```sh
dirsquat
```

Ages are based on file modification time.

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

Check `~/Downloads`:

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

Quote paths with spaces:

```sh
dirsquat --days 14 --names "$HOME/Project Notes"
```

## Setup Wizard

Use `--setup` to build a shell startup command interactively:

```sh
dirsquat --setup
```

Setup prompts for:

- directories to scan
- day cutoff
- count mode or names mode
- whether to follow symlinked directories
- whether to use plain output for scripts or agents

It prints a command you can add to `.zshrc` or `.bashrc`:

```sh
dirsquat --days 7 --count "$HOME/Downloads" "$HOME/Desktop"
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

Clear output:

```text
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃ D I R S Q U A T                                       CLEAR ┃
┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
┃ No files need attention.                                    ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

## Automation And Agents

Use `--plain` when another program or AI agent needs exact output. Plain mode is tab-separated and never shortens paths.

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

When no older files are found, plain mode writes no stdout. Warnings go to stderr:

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

Missing or unreadable directories print `WARN` and scanning continues. Argument errors print `ERROR` and exit with code `2`.

## Scanning Rules

`dirsquat` scans recursively by default.

Files and directories beginning with `.` are skipped.

Symlinked directories are not entered by default:

```sh
dirsquat ~/Downloads
```

Use `--follow-symlinks` to include symlinked directories:

```sh
dirsquat --follow-symlinks ~/Downloads
```

Symlink loops are skipped.

## Options

```text
--days N             report files older than N days
--count              show counts and oldest age by directory
--names              show file paths and ages
--plain              show tab-separated output for automation
--setup              build a shell startup command
--follow-symlinks    enter symlinked directories
--help               show help
--version            show version
```

If neither `--count` nor `--names` is passed, count mode is used.

Passing any directory argument replaces the default `~/Downloads` target.

## Development

Run checks:

```sh
gofmt -w .
go test ./...
go vet ./...
go build -o dirsquat .
```
