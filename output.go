package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	cardMinWidth = 60
	cardMaxWidth = 76
	pathMaxWidth = 54
)

func writeCLIError(w io.Writer, err error, plain bool) {
	if plain {
		writePlainError(w, err)
		return
	}
	writeError(w, err)
}

func writeScanWarnings(w io.Writer, warnings []ScanWarning, plain bool) {
	if len(warnings) == 0 {
		return
	}
	if plain {
		writePlainWarnings(w, warnings)
		return
	}
	writeWarnings(w, warnings)
}

func writeScanResult(w io.Writer, result ScanResult, opts cliOptions, now time.Time) {
	if totalOlderFiles(result.Roots) == 0 {
		if !opts.plain {
			writeClear(w)
		}
		return
	}

	if opts.mode == modeNames {
		if opts.plain {
			writePlainNames(w, result.Files, now)
		} else {
			writeNames(w, result.Files, now)
		}
		return
	}

	if opts.plain {
		writePlainCounts(w, result.Roots, now)
	} else {
		writeCounts(w, result.Roots, now)
	}
}

func printUsage(w io.Writer) {
	writeCard(w, "HELP", "file age report", [][]string{
		{
			"USAGE",
			"  dirsquat [--days N] [--count|--names] [--plain]",
			"           [--follow-symlinks] [DIR...]",
			"  dirsquat --setup",
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
			"  --plain              show tab-separated output for automation",
			"  --setup              build a shell startup command",
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

func writePlainCounts(w io.Writer, roots []RootResult, now time.Time) {
	for _, root := range roots {
		if root.Count == 0 {
			continue
		}
		fmt.Fprintf(w, "%d\t%d\t%s\n", root.Count, ageDaysSince(now, root.OldestModTime), root.Path)
	}
}

func writePlainNames(w io.Writer, files []FileMatch, now time.Time) {
	for _, file := range files {
		fmt.Fprintf(w, "%d\t%s\n", ageDaysSince(now, file.ModTime), file.Path)
	}
}

func writePlainWarnings(w io.Writer, warnings []ScanWarning) {
	for _, warning := range warnings {
		fmt.Fprintf(w, "WARN\t%s\t%s\n", warning.Path, warning.Err)
	}
}

func writePlainError(w io.Writer, err error) {
	fmt.Fprintf(w, "ERROR\t%s\n", err)
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

	days := ageDaysSince(now, modTime)
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

func ageDaysSince(now time.Time, modTime time.Time) int {
	if modTime.IsZero() {
		return 0
	}

	days := int(now.Sub(modTime).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
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
