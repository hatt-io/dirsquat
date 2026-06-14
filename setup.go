package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type setupConfig struct {
	days           int
	mode           outputMode
	plain          bool
	followSymlinks bool
	roots          []string
}

type setupSession struct {
	in     *bufio.Reader
	out    io.Writer
	err    error
	closed bool
}

func runSetup(stdin io.Reader, stdout, stderr io.Writer, defaults cliOptions) int {
	session := setupSession{
		in:  bufio.NewReader(stdin),
		out: stdout,
	}

	config := setupConfig{
		days:           defaults.days,
		mode:           defaults.mode,
		plain:          defaults.plain,
		followSymlinks: defaults.followSymlinks,
		roots:          setupDefaultRoots(defaults.roots),
	}

	writeSetupIntro(stdout)
	config.roots = session.askDirectories(config.roots)
	config.days = session.askDays(config.days)
	config.mode = session.askMode(config.mode)
	config.followSymlinks = session.askBool("4/5 SYMLINKS", "Follow symlinked directories", config.followSymlinks)
	config.plain = session.askBool("5/5 AUTOMATION", "Use plain output for scripts or agents", config.plain)

	if session.err != nil {
		writeCLIError(stderr, session.err, defaults.plain)
		return 1
	}

	writeSetupReady(stdout, buildSetupCommand(config))
	return 0
}

func writeSetupIntro(w io.Writer) {
	writeCard(w, "SETUP", "shell startup command builder", [][]string{
		{
			"WHAT THIS DOES",
			"  Builds a dirsquat command for .zshrc or .bashrc.",
			"  Answer a few prompts, then copy the command.",
		},
		{
			"TIP",
			"  Press Enter to accept a default.",
		},
	})
}

func (s *setupSession) askDirectories(defaults []string) []string {
	writeSetupStep(s.out, "1/5 DIRECTORIES", []string{
		"Enter directories separated by commas.",
		"Paths with spaces are fine.",
	})

	value := s.prompt("Directories", strings.Join(defaults, ", "))
	return parseSetupDirectories(value, defaults)
}

func (s *setupSession) askDays(defaultDays int) int {
	writeSetupStep(s.out, "2/5 AGE", []string{
		"Report files older than this many days.",
	})

	for {
		value := s.prompt("Days", strconv.Itoa(defaultDays))
		days, err := parseDays(value)
		if err == nil {
			return days
		}
		writeSetupNote(s.out, "Use a positive whole number.")
	}
}

func (s *setupSession) askMode(defaultMode outputMode) outputMode {
	writeSetupStep(s.out, "3/5 OUTPUT", []string{
		"1  Count files by directory",
		"2  List each file path",
	})

	for {
		value := strings.ToLower(s.prompt("Mode", setupModeLabel(defaultMode)))
		switch value {
		case "1", "count", "counts", "c":
			return modeCount
		case "2", "names", "name", "files", "n":
			return modeNames
		default:
			writeSetupNote(s.out, "Choose 1/count or 2/names.")
		}
	}
}

func (s *setupSession) askBool(title string, label string, defaultValue bool) bool {
	writeSetupStep(s.out, title, []string{
		label,
	})

	for {
		value := strings.ToLower(s.prompt(label, setupBoolLabel(defaultValue)))
		switch value {
		case "y", "yes", "true", "1":
			return true
		case "n", "no", "false", "0":
			return false
		default:
			writeSetupNote(s.out, "Choose yes or no.")
		}
	}
}

func (s *setupSession) prompt(label string, defaultValue string) string {
	fmt.Fprintf(s.out, "%s [%s]> ", label, defaultValue)
	if s.closed {
		fmt.Fprintln(s.out)
		return defaultValue
	}

	line, err := s.in.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.closed = true
			if line == "" {
				fmt.Fprintln(s.out)
				return defaultValue
			}
			fmt.Fprintln(s.out)
		} else {
			s.err = err
			fmt.Fprintln(s.out)
			return defaultValue
		}
	}
	if err == nil {
		fmt.Fprintln(s.out)
	}

	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue
	}
	return value
}

func writeSetupStep(w io.Writer, title string, lines []string) {
	section := append([]string{title}, indentSetupLines(lines)...)
	writeCard(w, "SETUP", "", [][]string{section})
}

func writeSetupNote(w io.Writer, message string) {
	writeCard(w, "SETUP", "check input", [][]string{
		{
			"NOTE",
			"  " + message,
		},
	})
}

func writeSetupReady(w io.Writer, command string) {
	writeCard(w, "READY", "copy into your shell startup file", [][]string{
		{
			"COMMAND",
			"  Add this line to .zshrc or .bashrc.",
		},
	})
	fmt.Fprintf(w, "\n%s\n\n", command)
}

func indentSetupLines(lines []string) []string {
	indented := make([]string, 0, len(lines))
	for _, line := range lines {
		indented = append(indented, "  "+line)
	}
	return indented
}

func setupDefaultRoots(roots []string) []string {
	if len(roots) == 0 {
		return []string{"$HOME/Downloads"}
	}

	defaults := make([]string, 0, len(roots))
	for _, root := range roots {
		defaults = append(defaults, normalizeSetupPath(root))
	}
	return defaults
}

func parseSetupDirectories(value string, defaults []string) []string {
	fields := strings.Split(value, ",")
	roots := make([]string, 0, len(fields))
	for _, field := range fields {
		root := normalizeSetupPath(field)
		if root != "" {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return defaults
	}
	return roots
}

func normalizeSetupPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "~" {
		return "$HOME"
	}
	if strings.HasPrefix(path, "~/") {
		return "$HOME/" + strings.TrimPrefix(path, "~/")
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		cleanHome := filepath.Clean(home)
		cleanPath := filepath.Clean(path)
		if cleanPath == cleanHome {
			return "$HOME"
		}
		prefix := cleanHome + string(os.PathSeparator)
		if strings.HasPrefix(cleanPath, prefix) {
			rel := strings.TrimPrefix(cleanPath, prefix)
			return "$HOME/" + filepath.ToSlash(rel)
		}
	}

	return path
}

func buildSetupCommand(config setupConfig) string {
	args := []string{
		"dirsquat",
		"--days",
		strconv.Itoa(config.days),
	}

	if config.mode == modeNames {
		args = append(args, "--names")
	} else {
		args = append(args, "--count")
	}
	if config.plain {
		args = append(args, "--plain")
	}
	if config.followSymlinks {
		args = append(args, "--follow-symlinks")
	}
	for _, root := range config.roots {
		args = append(args, shellQuotePath(root))
	}

	return strings.Join(args, " ")
}

func shellQuotePath(path string) string {
	if strings.HasPrefix(path, "$HOME") {
		return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
	}
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}

func setupModeLabel(mode outputMode) string {
	if mode == modeNames {
		return "names"
	}
	return "count"
}

func setupBoolLabel(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
