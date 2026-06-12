package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoArgsUseDefaultDaysAndDownloads(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	downloads := filepath.Join(home, "Downloads")
	writeTestFile(t, downloads, "old.txt", testNow.AddDate(0, 0, -8))
	writeTestFile(t, downloads, "new.txt", testNow.AddDate(0, 0, -2))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	want := fmt.Sprintf("%s: 1 file older than 7 days\n", downloads)
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestExplicitDaysWithDefaultDownloads(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	downloads := filepath.Join(home, "Downloads")
	writeTestFile(t, downloads, "old.txt", testNow.AddDate(0, 0, -8))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "10"}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestExplicitDirectoryUsesDefaultDays(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "old.txt", testNow.AddDate(0, 0, -8))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	want := fmt.Sprintf("%s: 1 file older than 7 days\n", dir)
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestHelpWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--help"}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	for _, want := range []string{"Usage:", "Defaults:", "--days 7", "DIR: ~/Downloads", "--count", "--names", "--follow-symlinks", "--help", "--version"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestVersionWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stdout.String() != appName+" "+version+"\n" {
		t.Fatalf("unexpected version output %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestArgumentErrorsFailClearly(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid days",
			args: []string{"--days", "zero"},
			want: "--days must be a positive integer",
		},
		{
			name: "zero days",
			args: []string{"--days", "0"},
			want: "--days must be a positive integer",
		},
		{
			name: "conflicting output modes",
			args: []string{"--count", "--names"},
			want: "use only one of --count or --names",
		},
		{
			name: "unknown flag",
			args: []string{"--json"},
			want: "flag provided but not defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := run(tt.args, &stdout, &stderr, testNow)

			if code != 2 {
				t.Fatalf("expected exit code 2, got %d", code)
			}
			if stdout.Len() != 0 {
				t.Fatalf("expected empty stdout, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr missing %q:\n%s", tt.want, stderr.String())
			}
			if !strings.Contains(stderr.String(), "Try 'dirsquat --help' for usage.") {
				t.Fatalf("stderr missing help hint:\n%s", stderr.String())
			}
		})
	}
}
