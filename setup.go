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

var errSetupCancelled = errors.New("setup cancelled")

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
	config.roots = session.askDirectories(config.roots, 1, 5)
	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}
	config.days = session.askDays(config.days, 2, 5)
	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}
	config.mode = session.askMode(config.mode, 3, 5)
	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}
	config.followSymlinks = session.askBool(4, 5, "Follow symlinked directories", config.followSymlinks)
	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}
	config.plain = session.askBool(5, 5, "Use plain output for scripts or agents", config.plain)

	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}

	command := buildSetupCommand(config)
	writeSetupReady(stdout, command)

	if !session.askBoolPrompt("Run automatically in new terminals", false) {
		if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
			return code
		}
		return 0
	}

	target := detectSetupShellTarget(os.Getenv("SHELL"))
	writeSetupShellTarget(stdout, target)
	startupFile := session.askStartupFile(target.Path)

	if code, stopped := finishSetupIfStopped(session, stdout, stderr, defaults.plain); stopped {
		return code
	}

	result, err := installSetupCommand(startupFile, command)
	if err != nil {
		writeSetupError(stderr, err, defaults.plain)
		return 1
	}

	writeSetupInstalled(stdout, result)
	if err := sourceStartupFile(target, result.Path, stdout, stderr); err != nil {
		writeSetupSourceFailed(stderr, result.DisplayPath, err)
		return 1
	}
	writeSetupSourced(stdout, result.DisplayPath)
	return 0
}

func writeSetupIntro(w io.Writer) {
	writeCard(w, "SETUP", "command builder", [][]string{
		{
			"Create a reusable dirsquat command.",
			"Run it anytime, or run it automatically in new terminals.",
			"Press Enter to keep a default.",
			"Type q to quit.",
		},
	})
}

func (s *setupSession) askDirectories(defaults []string, step int, total int) []string {
	value := s.prompt(setupPromptLabel(step, total, "Directories"), strings.Join(defaults, ", "))
	if s.err != nil {
		return defaults
	}
	return parseSetupDirectories(value, defaults)
}

func (s *setupSession) askDays(defaultDays int, step int, total int) int {
	for {
		value := s.prompt(setupPromptLabel(step, total, "Days"), strconv.Itoa(defaultDays))
		if s.err != nil {
			return defaultDays
		}
		days, err := parseDays(value)
		if err == nil {
			return days
		}
		fmt.Fprintln(s.out, "  Use a positive whole number.")
	}
}

func (s *setupSession) askMode(defaultMode outputMode, step int, total int) outputMode {
	fmt.Fprintln(s.out, "  1  Count files by directory")
	fmt.Fprintln(s.out, "  2  List each file path")
	for {
		value := strings.ToLower(s.prompt(setupPromptLabel(step, total, "Mode"), setupModeLabel(defaultMode)))
		if s.err != nil {
			return defaultMode
		}
		switch value {
		case "1", "count", "counts", "c":
			return modeCount
		case "2", "names", "name", "files", "n":
			return modeNames
		default:
			fmt.Fprintln(s.out, "  Choose 1/count or 2/names.")
		}
	}
}

func (s *setupSession) askBool(step int, total int, label string, defaultValue bool) bool {
	return s.askBoolPrompt(setupPromptLabel(step, total, label), defaultValue)
}

func (s *setupSession) askBoolPrompt(label string, defaultValue bool) bool {
	for {
		value := strings.ToLower(s.prompt(label, setupBoolLabel(defaultValue)))
		if s.err != nil {
			return defaultValue
		}
		switch value {
		case "y", "yes", "true", "1":
			return true
		case "n", "no", "false", "0":
			return false
		default:
			fmt.Fprintln(s.out, "  Choose yes or no.")
		}
	}
}

func (s *setupSession) askStartupFile(defaultPath string) string {
	return s.prompt("File to update", defaultPath)
}

func (s *setupSession) prompt(label string, defaultValue string) string {
	fmt.Fprintf(s.out, "%s [%s]> ", label, defaultValue)
	if s.err != nil {
		fmt.Fprintln(s.out)
		return defaultValue
	}
	if s.closed {
		fmt.Fprintln(s.out)
		return defaultValue
	}

	line, err := s.in.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.closed = true
			if line == "" {
				s.err = errSetupCancelled
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
	if isSetupQuit(value) {
		s.err = errSetupCancelled
		return defaultValue
	}
	if value == "" {
		return defaultValue
	}
	return value
}

func finishSetupIfStopped(session setupSession, stdout, stderr io.Writer, plain bool) (int, bool) {
	if session.err == nil {
		return 0, false
	}
	if errors.Is(session.err, errSetupCancelled) {
		writeSetupCancelled(stdout)
		return 0, true
	}
	writeSetupError(stderr, session.err, plain)
	return 1, true
}

func writeSetupReady(w io.Writer, command string) {
	writeCard(w, "READY", "copy or run this command", [][]string{
		{
			"COMMAND",
			"  Run it now, or save it for new terminals.",
		},
	})
	fmt.Fprintf(w, "\n%s\n\n", command)
}

func writeSetupShellTarget(w io.Writer, target setupShellTarget) {
	writeCard(w, "STARTUP", "automatic runs", [][]string{
		{
			"DETECTED SHELL",
			"  " + target.Name,
		},
		{
			"FILE TO UPDATE",
			"  " + target.Path,
		},
	})
}

func writeSetupInstalled(w io.Writer, result setupInstallResult) {
	status := "Startup file updated."
	if result.Unchanged {
		status = "Startup file already had this command."
	}

	writeCard(w, "DONE", "one dirsquat command is active", [][]string{
		{
			"FILE",
			"  " + result.DisplayPath,
		},
		{
			"RESULT",
			"  " + status,
		},
	})
}

func writeSetupSourced(w io.Writer, displayPath string) {
	writeCard(w, "RAN NOW", "startup command ran once", [][]string{
		{
			"FILE",
			"  " + displayPath,
		},
	})
}

func writeSetupCancelled(w io.Writer) {
	writeCard(w, "CANCELLED", "no changes made", [][]string{
		{
			"Setup stopped.",
		},
	})
}

func writeSetupSourceFailed(w io.Writer, displayPath string, err error) {
	writeCard(w, "WARN", "could not run the startup file now", [][]string{
		{
			"FILE",
			"  " + displayPath,
		},
		setupMessageSection("ISSUE", err.Error()),
		{
			"NEXT",
			"  Open a new terminal, or check the file above.",
		},
	})
}

func writeSetupError(w io.Writer, err error, plain bool) {
	if plain {
		writePlainError(w, err)
		return
	}
	writeCard(w, "ERROR", "setup stopped", [][]string{
		setupMessageSection("ISSUE", err.Error()),
	})
}

func setupPromptLabel(step int, total int, label string) string {
	return fmt.Sprintf("%d/%d %s", step, total, label)
}

func setupMessageSection(title string, message string) []string {
	lines := []string{title}
	for _, line := range wrapSetupMessage(message, 58) {
		lines = append(lines, "  "+line)
	}
	return lines
}

func wrapSetupMessage(message string, width int) []string {
	words := strings.Fields(message)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, 1)
	current := words[0]
	for _, word := range words[1:] {
		if runeLen(current)+1+runeLen(word) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}
	lines = append(lines, current)
	return lines
}

func isSetupQuit(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "q", "quit", "exit":
		return true
	default:
		return false
	}
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
	if path == "$HOME" || strings.HasPrefix(path, "$HOME/") {
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
