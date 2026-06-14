package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	setupBlockStart = "# dirsquat setup start"
	setupBlockEnd   = "# dirsquat setup end"
)

type setupShellTarget struct {
	Name       string
	Path       string
	Executable string
}

type setupInstallResult struct {
	Path        string
	DisplayPath string
	Unchanged   bool
}

var sourceStartupFile = runSourceStartupFile

func detectSetupShellTarget(shellPath string) setupShellTarget {
	executable := shellPath
	name := filepath.Base(shellPath)
	if name == "" {
		name = "unknown"
	}

	switch name {
	case "zsh":
		return setupShellTarget{Name: "zsh", Path: "$HOME/.zshrc", Executable: executable}
	case "bash":
		return setupShellTarget{Name: "bash", Path: "$HOME/.bashrc", Executable: executable}
	case "fish":
		return setupShellTarget{Name: "fish", Path: "$HOME/.config/fish/config.fish", Executable: executable}
	default:
		return setupShellTarget{Name: "unknown", Path: "$HOME/.profile", Executable: "sh"}
	}
}

func installSetupCommand(displayPath string, command string) (setupInstallResult, error) {
	displayPath = strings.TrimSpace(displayPath)
	result := setupInstallResult{DisplayPath: displayPath}

	path, err := expandSetupFilePath(displayPath)
	if err != nil {
		return result, err
	}
	result.Path = path
	result.DisplayPath = normalizeSetupPath(path)
	writePath, err := setupWritePath(path)
	if err != nil {
		return result, err
	}

	originalBytes, readErr := os.ReadFile(writePath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return result, fmt.Errorf("could not read %s: %w", result.DisplayPath, readErr)
	}
	mode := os.FileMode(0644)
	if readErr == nil {
		info, err := os.Stat(writePath)
		if err != nil {
			return result, fmt.Errorf("could not check %s: %w", result.DisplayPath, err)
		}
		mode = info.Mode().Perm()
	}

	original := string(originalBytes)
	updated, err := rewriteSetupFileContent(original, command)
	if err != nil {
		return result, err
	}
	if updated == original {
		result.Unchanged = true
		return result, nil
	}

	if err := os.MkdirAll(filepath.Dir(writePath), 0755); err != nil {
		return result, fmt.Errorf("could not create %s: %w", normalizeSetupPath(filepath.Dir(writePath)), err)
	}
	if err := writeSetupFileAtomic(writePath, []byte(updated), mode); err != nil {
		return result, fmt.Errorf("could not write %s: %w", result.DisplayPath, err)
	}

	return result, nil
}

func setupWritePath(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return path, nil
	}
	return filepath.EvalSymlinks(path)
}

func writeSetupFileAtomic(path string, content []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".dirsquat-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()

	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	return os.Rename(tempName, path)
}

func runSourceStartupFile(target setupShellTarget, path string, stdout, stderr io.Writer) error {
	if target.Executable == "" {
		target.Executable = target.Name
	}
	if target.Executable == "" || target.Name == "unknown" {
		target.Executable = "sh"
	}

	sourceCommand := "source " + shellQuoteLiteral(path)
	if target.Name == "unknown" {
		sourceCommand = ". " + shellQuoteLiteral(path)
	}

	cmd := exec.Command(target.Executable, "-c", sourceCommand)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func expandSetupFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("startup file is required")
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("could not determine home directory")
	}

	switch {
	case path == "~", path == "$HOME":
		return home, nil
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[2:]), nil
	case strings.HasPrefix(path, "$HOME/"):
		return filepath.Join(home, strings.TrimPrefix(path, "$HOME/")), nil
	default:
		return path, nil
	}
}

func rewriteSetupFileContent(content string, command string) (string, error) {
	newline := setupNewline(content)
	normalizedContent := strings.ReplaceAll(content, "\r\n", "\n")
	lines := splitSetupLines(normalizedContent)
	filtered := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == setupBlockStart {
			end := findSetupBlockEnd(lines, i+1)
			if end == -1 {
				continue
			}
			i = end
			continue
		}
		if strings.TrimSpace(line) == setupBlockEnd {
			continue
		}
		action := classifyStartupLine(line)
		switch action {
		case setupLineRemove:
			continue
		case setupLineConflict:
			return "", fmt.Errorf("not changed: line %d has dirsquat with other commands. Put dirsquat on its own line or remove that old dirsquat command. Then run setup again", i+1)
		}
		filtered = append(filtered, line)
	}

	body := strings.Join(filtered, "")
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if body != "" && !strings.HasSuffix(body, "\n\n") {
		body += "\n"
	}

	body += setupBlockStart + "\n" + command + "\n" + setupBlockEnd + "\n"
	if newline == "\r\n" {
		body = strings.ReplaceAll(body, "\n", "\r\n")
	}
	return body, nil
}

func setupNewline(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func splitSetupLines(content string) []string {
	if content == "" {
		return nil
	}
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}

func findSetupBlockEnd(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == setupBlockEnd {
			return i
		}
	}
	return -1
}

type setupLineAction int

const (
	setupLineKeep setupLineAction = iota
	setupLineRemove
	setupLineConflict
)

func classifyStartupLine(line string) setupLineAction {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return setupLineKeep
	}
	if strings.HasPrefix(trimmed, "alias ") || strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "dirsquat()") {
		return setupLineKeep
	}

	command, rest := firstShellWord(trimmed)
	if command == "command" || command == "exec" {
		command, rest = firstShellWord(rest)
	}
	if commandName(command) == "dirsquat" {
		if hasCommandJoiner(rest) {
			return setupLineConflict
		}
		return setupLineRemove
	}

	if strings.Contains(trimmed, "dirsquat") {
		return setupLineConflict
	}
	return setupLineKeep
}

func firstShellWord(line string) (string, string) {
	line = strings.TrimSpace(line)
	var quote rune
	escaped := false

	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == ' ' || r == '\t' || r == ';' || r == '|' || r == '&' {
			return line[:i], line[i:]
		}
	}

	return line, ""
}

func hasCommandJoiner(text string) bool {
	var quote rune
	escaped := false
	var previous rune

	for _, r := range text {
		if escaped {
			escaped = false
			previous = r
			continue
		}
		if r == '\\' {
			escaped = true
			previous = r
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			previous = r
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			previous = r
			continue
		}
		if r == ';' || r == '|' || (r == '&' && previous != '>' && previous != '<') {
			return true
		}
		previous = r
	}
	return false
}

func commandName(field string) string {
	field = strings.Trim(field, `"'`)
	if field == "" {
		return ""
	}
	return filepath.Base(field)
}

func shellQuoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
