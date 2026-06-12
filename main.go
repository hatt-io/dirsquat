package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	appName     = "dirsquat"
	defaultDays = 7
)

var version = "0.1.0"

type outputMode int

const (
	modeCount outputMode = iota
	modeNames
)

type cliOptions struct {
	days           int
	mode           outputMode
	followSymlinks bool
	help           bool
	version        bool
	roots          []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, time.Now()))
}

func run(args []string, stdout, stderr io.Writer, now time.Time) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", appName, err)
		fmt.Fprintf(stderr, "Try '%s --help' for usage.\n", appName)
		return 2
	}

	if opts.help {
		printUsage(stdout)
		return 0
	}

	if opts.version {
		fmt.Fprintf(stdout, "%s %s\n", appName, version)
		return 0
	}

	scanner := Scanner{
		FollowSymlinks: opts.followSymlinks,
		Now:            now,
	}
	result := scanner.Scan(opts.roots, opts.days)

	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "%s: warning: %s: %v\n", appName, warning.Path, warning.Err)
	}

	if totalMatches(result.Roots) == 0 {
		writeNoMatches(stdout, opts.days)
		return 0
	}

	switch opts.mode {
	case modeNames:
		for _, file := range result.Files {
			fmt.Fprintln(stdout, file.Path)
		}
	default:
		writeCounts(stdout, result.Roots, opts.days)
	}

	return 0
}

func parseArgs(args []string) (cliOptions, error) {
	opts := cliOptions{
		days: defaultDays,
		mode: modeCount,
	}
	daysValue := strconv.Itoa(defaultDays)
	var count bool
	var names bool

	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&daysValue, "days", daysValue, "number of days")
	fs.BoolVar(&count, "count", false, "print counts")
	fs.BoolVar(&names, "names", false, "print file names")
	fs.BoolVar(&opts.followSymlinks, "follow-symlinks", false, "follow symlinked directories")
	fs.BoolVar(&opts.help, "help", false, "show help")
	fs.BoolVar(&opts.version, "version", false, "show version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			opts.help = true
			return opts, nil
		}
		return opts, err
	}

	if opts.help || opts.version {
		return opts, nil
	}

	if count && names {
		return opts, errors.New("use only one of --count or --names")
	}

	days, err := parseDays(daysValue)
	if err != nil {
		return opts, err
	}
	opts.days = days

	if names {
		opts.mode = modeNames
	}

	opts.roots = fs.Args()
	if len(opts.roots) == 0 {
		defaultRoot, err := defaultDownloadsDir()
		if err != nil {
			return opts, err
		}
		opts.roots = []string{defaultRoot}
	}

	return opts, nil
}

func parseDays(value string) (int, error) {
	days, err := strconv.Atoi(value)
	if err != nil || days <= 0 {
		return 0, errors.New("--days must be a positive integer")
	}

	const maxDurationDays = int64(1<<63-1) / int64(24*time.Hour)
	if int64(days) > maxDurationDays {
		return 0, errors.New("--days is too large")
	}

	return days, nil
}

func defaultDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("could not determine home directory for default ~/Downloads")
	}

	return filepath.Join(home, "Downloads"), nil
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `%s reports visible files older than a configured number of days.

Usage:
  %s [--days N] [--count|--names] [--follow-symlinks] [DIR...]

Defaults:
  --days %d
  DIR: ~/Downloads

Output:
  Prints a confirmation line when no files match.

Options:
  --days N             report files older than N days
  --count              print one count line per directory with matches
  --names              print matching file paths, one per line
  --follow-symlinks    enter symlinked directories
  --help               show this help
  --version            show version
`, appName, appName, defaultDays)
}

func writeCounts(w io.Writer, roots []RootResult, days int) {
	for _, root := range roots {
		if root.Count == 0 {
			continue
		}

		fileWord := "files"
		if root.Count == 1 {
			fileWord = "file"
		}

		dayWord := "days"
		if days == 1 {
			dayWord = "day"
		}

		fmt.Fprintf(w, "%s: %d %s older than %d %s\n", root.Path, root.Count, fileWord, days, dayWord)
	}
}

func writeNoMatches(w io.Writer, days int) {
	dayWord := "days"
	if days == 1 {
		dayWord = "day"
	}

	fmt.Fprintf(w, "No files older than %d %s found.\n", days, dayWord)
}

func totalMatches(roots []RootResult) int {
	total := 0
	for _, root := range roots {
		total += root.Count
	}
	return total
}

func isHiddenName(name string) bool {
	return name != "." && name != ".." && strings.HasPrefix(name, ".")
}
