package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetupBuildsDefaultShellCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runWithInput([]string{"--setup"}, strings.NewReader("\n\n\n\n\n"), &stdout, &stderr, testNow)

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
		"Answer a few prompts, then copy the command.",
		`dirsquat --days 7 --count "$HOME/Downloads"`,
	)
}

func TestSetupBuildsCustomShellCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	input := strings.Join([]string{
		"~/Downloads, $HOME/Desktop, /tmp/Project Notes",
		"14",
		"2",
		"yes",
		"yes",
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
		"1/5 DIRECTORIES",
		"2/5 AGE",
		"3/5 OUTPUT",
		"4/5 SYMLINKS",
		"5/5 AUTOMATION",
		`dirsquat --days 14 --names --plain --follow-symlinks "$HOME/Downloads" "$HOME/Desktop" '/tmp/Project Notes'`,
	)
}
