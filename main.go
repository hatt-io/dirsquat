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
	"unicode/utf8"
)

const (
	appName      = "dirsquat"
	defaultDays  = 7
	cardMinWidth = 60
	cardMaxWidth = 76
	pathMaxWidth = 54
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
		writeError(stderr, err)
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

	scanner := Scanner{
		FollowSymlinks: opts.followSymlinks,
		Now:            now,
	}
	result := scanner.Scan(opts.roots, opts.days)

	if len(result.Warnings) > 0 {
		writeWarnings(stderr, result.Warnings)
	}

	if totalOlderFiles(result.Roots) == 0 {
		writeClear(stdout)
		return 0
	}

	switch opts.mode {
	case modeNames:
		writeNames(stdout, result.Files, now)
	default:
		writeCounts(stdout, result.Roots, now)
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
	writeCard(w, "HELP", "visible-file age report · report only", [][]string{
		{
			"USAGE",
			"  dirsquat [--days N] [--count|--names] [--follow-symlinks] [DIR...]",
		},
		{
			"DEFAULTS",
			fmt.Sprintf("  days       %d", defaultDays),
			"  directory  ~/Downloads",
			"  mode       count",
		},
		{
			"STATES",
			"  CLEAR   no older files found",
			"  FOUND   older files found",
			"  WARN    scan problem; command continues",
			"  ERROR   argument problem; command exits",
		},
		{
			"OPTIONS",
			"  --days N             report files older than N days",
			"  --count              show counts and oldest age by directory",
			"  --names              show file paths and ages",
			"  --follow-symlinks    enter symlinked directories",
			"  --help               show this help",
			"  --version            show version",
		},
	})
}

func writeCounts(w io.Writer, roots []RootResult, now time.Time) {
	countWidth := 0
	ageWidth := 0
	for _, root := range roots {
		if root.Count > 0 {
			countWidth = maxInt(countWidth, runeLen(strconv.Itoa(root.Count)))
			ageWidth = maxInt(ageWidth, runeLen(ageSince(now, root.OldestModTime)))
		}
	}
	countWidth = maxInt(countWidth, runeLen("FILES"))
	ageWidth = maxInt(ageWidth, runeLen("OLDEST FILE AGE"))

	rows := make([]string, 0, len(roots)+1)
	rows = append(rows, fmt.Sprintf("%s  %s  DIRECTORY",
		padRight("FILES", countWidth),
		padRight("OLDEST FILE AGE", ageWidth),
	))
	for _, root := range roots {
		if root.Count == 0 {
			continue
		}
		rows = append(rows, fmt.Sprintf("%s  %s  %s",
			padRight(strconv.Itoa(root.Count), countWidth),
			padRight(ageSince(now, root.OldestModTime), ageWidth),
			displayPath(root.Path),
		))
	}

	writeCard(w, "FOUND", "", [][]string{rows})
}

func writeNames(w io.Writer, files []FileMatch, now time.Time) {
	ageWidth := 0
	for _, file := range files {
		ageWidth = maxInt(ageWidth, runeLen(ageSince(now, file.ModTime)))
	}
	ageWidth = maxInt(ageWidth, runeLen("FILE AGE"))

	rows := make([]string, 0, len(files)+1)
	rows = append(rows, fmt.Sprintf("%s  FILE", padRight("FILE AGE", ageWidth)))
	for _, file := range files {
		rows = append(rows, fmt.Sprintf("%s  %s",
			padRight(ageSince(now, file.ModTime), ageWidth),
			displayPath(file.Path),
		))
	}

	writeCard(w, "FOUND", "", [][]string{rows})
}

func writeClear(w io.Writer) {
	writeCard(w, "CLEAR", "", [][]string{
		{
			"No files need attention.",
		},
	})
}

func writeWarnings(w io.Writer, warnings []ScanWarning) {
	summary := []string{
		"ISSUES",
		fmt.Sprintf("    %d", len(warnings)),
		"Scan continues after each issue.",
	}

	rows := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		rows = append(rows, displayPath(warning.Path))
		rows = append(rows, "  "+warning.Err.Error())
	}

	writeCard(w, "WARN", "scan completed with warnings", [][]string{summary, append([]string{"DETAILS"}, rows...)})
}

func writeError(w io.Writer, err error) {
	writeCard(w, "ERROR", "argument problem", [][]string{
		{
			"ISSUE",
			"  " + err.Error(),
		},
		{
			"NEXT",
			"  dirsquat --help",
		},
	})
}

func writeVersion(w io.Writer) {
	writeCard(w, "VERSION", "release information", [][]string{
		{
			"VERSION",
			"  " + version,
		},
	})
}

func writeCard(w io.Writer, state string, subtitle string, sections [][]string) {
	brand := "D I R S Q U A T"
	width := runeLen(brand) + runeLen(state) + 4
	if subtitle != "" {
		width = maxInt(width, runeLen(subtitle))
	}
	for _, section := range sections {
		for _, line := range section {
			width = maxInt(width, runeLen(line))
		}
	}
	width = maxInt(width, cardMinWidth)
	width = minInt(width, cardMaxWidth)

	header := brand + strings.Repeat(" ", width-runeLen(brand)-runeLen(state)) + state

	fmt.Fprintf(w, "┏%s┓\n", strings.Repeat("━", width+2))
	writeCardLine(w, header, width)
	if subtitle != "" {
		writeCardLine(w, subtitle, width)
	}
	fmt.Fprintf(w, "┣%s┫\n", strings.Repeat("━", width+2))
	for i, section := range sections {
		if i > 0 {
			fmt.Fprintf(w, "┣%s┫\n", strings.Repeat("━", width+2))
		}
		for _, line := range section {
			writeCardLine(w, line, width)
		}
	}
	fmt.Fprintf(w, "┗%s┛\n", strings.Repeat("━", width+2))
}

func writeCardLine(w io.Writer, line string, width int) {
	line = fitText(line, width)
	fmt.Fprintf(w, "┃ %s ┃\n", padRight(line, width))
}

func ageSince(now time.Time, modTime time.Time) string {
	if modTime.IsZero() {
		return "unknown"
	}

	days := int(now.Sub(modTime).Hours() / 24)
	if days < 0 {
		days = 0
	}
	if days == 0 {
		return "less than 1 day"
	}
	if days == 1 {
		return "1 day"
	}
	if days < 60 {
		return fmt.Sprintf("%d days", days)
	}

	months := days / 30
	remainingDays := days % 30
	if days < 365 {
		if remainingDays == 0 {
			return fmt.Sprintf("%d months", months)
		}
		return fmt.Sprintf("%d months %d days", months, remainingDays)
	}

	years := days / 365
	remainingDays = days % 365
	if remainingDays == 0 {
		return fmt.Sprintf("%d years", years)
	}
	return fmt.Sprintf("%d years %d days", years, remainingDays)
}

func displayPath(path string) string {
	return fitText(path, pathMaxWidth)
}

func fitText(value string, width int) string {
	if runeLen(value) <= width {
		return value
	}
	if width <= 1 {
		return strings.Repeat("…", maxInt(width, 0))
	}

	runes := []rune(value)
	keep := width - 1
	left := keep / 2
	right := keep - left
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func padRight(value string, width int) string {
	padding := width - runeLen(value)
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func totalOlderFiles(roots []RootResult) int {
	total := 0
	for _, root := range roots {
		total += root.Count
	}
	return total
}

func isHiddenName(name string) bool {
	return name != "." && name != ".." && strings.HasPrefix(name, ".")
}
