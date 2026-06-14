package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupBuildsDefaultCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	requireOutputContains(t, stdout.String(),
		"D I R S Q U A T",
		"SETUP",
		"READY",
		"Create a reusable dirsquat command.",
		"Run automatically in new terminals",
		`dirsquat --days 7 --count "$HOME/Downloads"`,
	)
}

func TestSetupCanQuitImmediately(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("q\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	requireOutputContains(t, stdout.String(), "CANCELLED", "no changes made")
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("expected no startup file, got err %v", err)
	}
}

func TestSetupCtrlDQuitsWithoutWriting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader(""), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	requireOutputContains(t, stdout.String(), "CANCELLED", "no changes made")
	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("expected no startup file, got err %v", err)
	}
}

func TestSetupBuildsCustomCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	input := strings.Join([]string{
		"~/Downloads, $HOME/Desktop, /tmp/Project Notes",
		"14",
		"2",
		"yes",
		"yes",
		"no",
		"",
	}, "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader(input), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	requireOutputContains(t, stdout.String(),
		"1/5 Directories",
		"2/5 Days",
		"3/5 Mode",
		"4/5 Follow symlinked directories",
		"5/5 Use plain output for scripts or agents",
		`dirsquat --days 14 --names --plain --follow-symlinks "$HOME/Downloads" "$HOME/Desktop" '/tmp/Project Notes'`,
	)
}

func TestSetupOnlyExpandsHomeVariableWhenItIsAHomePath(t *testing.T) {
	command := buildSetupCommand(setupConfig{
		days:  defaultDays,
		mode:  modeCount,
		roots: []string{"$HOME/Downloads", "$HOME_BACKUP/Downloads"},
	})

	want := `dirsquat --days 7 --count "$HOME/Downloads" '$HOME_BACKUP/Downloads'`
	if command != want {
		t.Fatalf("unexpected command:\nwant %q\n got %q", want, command)
	}
}

func TestSetupInstallsZshStartupCommandOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	sourceCalls := stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	existing := strings.Join([]string{
		"export TEST_VALUE=1",
		setupBlockStart,
		`dirsquat --days 3 --count "$HOME/Downloads"`,
		setupBlockEnd,
		setupBlockStart,
		`dirsquat --days 4 --names "$HOME/Desktop"`,
		setupBlockEnd,
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(existing), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	content := string(contentBytes)
	requireOutputContains(t, stdout.String(), "STARTUP", "DETECTED SHELL", "zsh", "DONE", "RAN NOW", "$HOME/.zshrc")
	requireOutputContains(t, content, "export TEST_VALUE=1", setupBlockStart, `dirsquat --days 7 --count "$HOME/Downloads"`, setupBlockEnd)
	if count := strings.Count(content, "dirsquat --days"); count != 1 {
		t.Fatalf("expected one dirsquat command, got %d:\n%s", count, content)
	}
	if len(*sourceCalls) != 1 {
		t.Fatalf("expected one source call, got %d", len(*sourceCalls))
	}
	if (*sourceCalls)[0].target.Name != "zsh" || (*sourceCalls)[0].path != zshrc {
		t.Fatalf("unexpected source call: %+v", (*sourceCalls)[0])
	}
}

func TestSetupInstallsFishStartupCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/opt/homebrew/bin/fish")
	sourceCalls := stubSourceStartupFile(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
	contentBytes, err := os.ReadFile(fishConfig)
	if err != nil {
		t.Fatalf("read fish config: %v", err)
	}
	requireOutputContains(t, stdout.String(), "fish", "$HOME/.config/fish/config.fish")
	requireOutputContains(t, string(contentBytes), setupBlockStart, `dirsquat --days 7 --count "$HOME/Downloads"`, setupBlockEnd)
	if len(*sourceCalls) != 1 {
		t.Fatalf("expected one source call, got %d", len(*sourceCalls))
	}
	if (*sourceCalls)[0].target.Name != "fish" || (*sourceCalls)[0].path != fishConfig {
		t.Fatalf("unexpected source call: %+v", (*sourceCalls)[0])
	}
}

func TestSetupCanQuitBeforeWritingStartupFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	zshrc := filepath.Join(home, ".zshrc")
	original := "export KEEP_ME=1\n"
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\nquit\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	requireOutputContains(t, stdout.String(), "CANCELLED", "no changes made")

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if string(contentBytes) != original {
		t.Fatalf("startup file changed unexpectedly:\n%s", string(contentBytes))
	}
}

func TestSetupInstallCanUseCustomStartupFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	sourceCalls := stubSourceStartupFile(t)

	customFile := filepath.Join(home, "shell files", "startup.sh")
	input := strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"yes",
		customFile,
		"",
	}, "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader(input), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	contentBytes, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("read custom startup file: %v", err)
	}
	requireOutputContains(t, string(contentBytes), `dirsquat --days 7 --count "$HOME/Downloads"`)
	if len(*sourceCalls) != 1 {
		t.Fatalf("expected one source call, got %d", len(*sourceCalls))
	}
	if (*sourceCalls)[0].path != customFile {
		t.Fatalf("unexpected source path: %s", (*sourceCalls)[0].path)
	}
}

func TestSetupInstallPreservesStartupFilePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("export KEEP_ME=1\n"), 0600); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}
	if err := os.Chmod(zshrc, 0600); err != nil {
		t.Fatalf("chmod existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	info, err := os.Stat(zshrc)
	if err != nil {
		t.Fatalf("stat .zshrc: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected permissions 0600, got %o", got)
	}
}

func TestSetupInstallPreservesSymlinkedStartupFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	stubSourceStartupFile(t)

	target := filepath.Join(home, "dotfiles", "zshrc")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("export KEEP_ME=1\n"), 0644); err != nil {
		t.Fatalf("write target .zshrc: %v", err)
	}
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.Symlink(target, zshrc); err != nil {
		t.Fatalf("symlink .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	linkInfo, err := os.Lstat(zshrc)
	if err != nil {
		t.Fatalf("lstat .zshrc: %v", err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected .zshrc to remain a symlink, mode is %s", linkInfo.Mode())
	}
	contentBytes, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target .zshrc: %v", err)
	}
	requireOutputContains(t, string(contentBytes), "export KEEP_ME=1", `dirsquat --days 7 --count "$HOME/Downloads"`)
}

func TestSetupReplacesUnmanagedDirsquatCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	sourceCalls := stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	original := strings.Join([]string{
		"export TEST_VALUE=1",
		`dirsquat --days 4 --names "$HOME/Desktop"`,
		"export TEST_VALUE=2",
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	content := string(contentBytes)
	requireOutputContains(t, content, "export TEST_VALUE=1", "export TEST_VALUE=2", setupBlockStart, `dirsquat --days 7 --count "$HOME/Downloads"`, setupBlockEnd)
	if strings.Contains(content, `dirsquat --days 4 --names "$HOME/Desktop"`) {
		t.Fatalf("old dirsquat command was not removed:\n%s", content)
	}
	if count := strings.Count(content, "dirsquat --days"); count != 1 {
		t.Fatalf("expected one dirsquat command, got %d:\n%s", count, content)
	}
	if len(*sourceCalls) != 1 {
		t.Fatalf("expected one source call, got %d", len(*sourceCalls))
	}
}

func TestSetupReplacesStandaloneDirsquatWithRedirection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	original := strings.Join([]string{
		"export TEST_VALUE=1",
		`dirsquat --days 4 "$HOME/Desktop" >/tmp/dirsquat.log 2>&1`,
		"export TEST_VALUE=2",
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	content := string(contentBytes)
	requireOutputContains(t, content, "export TEST_VALUE=1", "export TEST_VALUE=2", `dirsquat --days 7 --count "$HOME/Downloads"`)
	if strings.Contains(content, ">/tmp/dirsquat.log") {
		t.Fatalf("old redirected dirsquat command was not removed:\n%s", content)
	}
}

func TestSetupKeepsNonDirsquatLinesFromIncompleteManagedBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	sourceCalls := stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	original := strings.Join([]string{
		"export TEST_VALUE=1",
		setupBlockStart,
		`dirsquat --days 4 --names "$HOME/Desktop"`,
		"export TEST_VALUE=2",
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	content := string(contentBytes)
	requireOutputContains(t, content, "export TEST_VALUE=1", "export TEST_VALUE=2", setupBlockStart, `dirsquat --days 7 --count "$HOME/Downloads"`, setupBlockEnd)
	if strings.Contains(content, `dirsquat --days 4 --names "$HOME/Desktop"`) {
		t.Fatalf("old dirsquat command was not removed:\n%s", content)
	}
	if count := strings.Count(content, "dirsquat --days"); count != 1 {
		t.Fatalf("expected one dirsquat command, got %d:\n%s", count, content)
	}
	if len(*sourceCalls) != 1 {
		t.Fatalf("expected one source call, got %d", len(*sourceCalls))
	}
}

func TestSetupRefusesDirsquatMixedWithOtherCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	sourceCalls := stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	original := strings.Join([]string{
		"export TEST_VALUE=1",
		`echo before; dirsquat --days 4 "$HOME/Desktop"; echo after`,
		"export TEST_VALUE=2",
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	requireOutputContains(t, stderr.String(), "ERROR", "not changed", "dirsquat with other commands", "run setup again")

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if string(contentBytes) != original {
		t.Fatalf("startup file changed unexpectedly:\n%s", string(contentBytes))
	}
	if len(*sourceCalls) != 0 {
		t.Fatalf("expected no source calls, got %d", len(*sourceCalls))
	}
}

func TestSetupRefusesDirsquatMixedWithOtherCommandsWithoutSpaces(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	sourceCalls := stubSourceStartupFile(t)

	zshrc := filepath.Join(home, ".zshrc")
	original := strings.Join([]string{
		"export TEST_VALUE=1",
		`dirsquat --days 4 "$HOME/Desktop"&&echo done`,
		"export TEST_VALUE=2",
		"",
	}, "\n")
	if err := os.WriteFile(zshrc, []byte(original), 0644); err != nil {
		t.Fatalf("write existing .zshrc: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\ny\n\n"), &stdout, &stderr, testNow)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	requireOutputContains(t, stderr.String(), "ERROR", "not changed", "dirsquat with other commands", "run setup again")

	contentBytes, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if string(contentBytes) != original {
		t.Fatalf("startup file changed unexpectedly:\n%s", string(contentBytes))
	}
	if len(*sourceCalls) != 0 {
		t.Fatalf("expected no source calls, got %d", len(*sourceCalls))
	}
}

type sourceStartupFileCall struct {
	target setupShellTarget
	path   string
}

func stubSourceStartupFile(t *testing.T) *[]sourceStartupFileCall {
	t.Helper()

	var calls []sourceStartupFileCall
	previous := sourceStartupFile
	sourceStartupFile = func(target setupShellTarget, path string, stdout, stderr io.Writer) error {
		calls = append(calls, sourceStartupFileCall{target: target, path: path})
		return nil
	}
	t.Cleanup(func() {
		sourceStartupFile = previous
	})

	return &calls
}
