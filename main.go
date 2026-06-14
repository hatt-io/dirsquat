package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	plain          bool
	setup          bool
	followSymlinks bool
	help           bool
	version        bool
	roots          []string
}

func main() {
	os.Exit(runWithInput(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now()))
}

func run(args []string, stdout, stderr io.Writer, now time.Time) int {
	return runWithInput(args, os.Stdin, stdout, stderr, now)
}

func runWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	opts, err := parseArgs(args)
	if err != nil {
		writeCLIError(stderr, err, opts.plain)
		return 2
	}

	if opts.help {
		printUsage(stdout)
		return 0
	}

	if opts.version {
		writeVersion(stdout)
		return 0
	}

	if opts.setup {
		return runSetup(stdin, stdout, stderr, opts)
	}

	scanner := Scanner{
		FollowSymlinks: opts.followSymlinks,
		Now:            now,
	}
	result := scanner.Scan(opts.roots, opts.days)

	writeScanWarnings(stderr, result.Warnings, opts.plain)
	writeScanResult(stdout, result, opts, now)

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
	fs.BoolVar(&opts.plain, "plain", false, "print tab-separated output")
	fs.BoolVar(&opts.setup, "setup", false, "run interactive setup")
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
