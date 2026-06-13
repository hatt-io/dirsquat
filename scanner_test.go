package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var testNow = time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)

func TestFilesOlderThanThresholdAreReported(t *testing.T) {
	dir := t.TempDir()
	old := writeTestFile(t, dir, "old.txt", testNow.AddDate(0, 0, -8))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, []string{old})
}

func TestNewerFilesAreNotReported(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "new.txt", testNow.AddDate(0, 0, -2))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, nil)
}

func TestFilesExactlyAtThresholdAreNotReported(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "threshold.txt", testNow.AddDate(0, 0, -7))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, nil)
}

func TestHiddenFilesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, ".old.txt", testNow.AddDate(0, 0, -8))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, nil)
}

func TestHiddenDirectoriesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.Mkdir(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, hiddenDir, "old.txt", testNow.AddDate(0, 0, -8))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, nil)
}

func TestRecursiveScanningWorks(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "one", "two")
	old := writeTestFile(t, nested, "old.txt", testNow.AddDate(0, 0, -8))

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, []string{old})
}

func TestSymlinkedDirectoriesAreNotFollowedByDefault(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	writeTestFile(t, target, "old.txt", testNow.AddDate(0, 0, -8))
	link := filepath.Join(dir, "link")
	createSymlink(t, target, link)

	result := scanForTest([]string{dir}, false, defaultDays)

	requirePaths(t, result.Files, nil)
}

func TestFollowSymlinksWorksWithoutDirectoryLoops(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	old := writeTestFile(t, target, "old.txt", testNow.AddDate(0, 0, -8))
	link := filepath.Join(dir, "link")
	createSymlink(t, target, link)
	createSymlink(t, dir, filepath.Join(target, "back"))

	result := scanForTest([]string{dir}, true, defaultDays)

	requirePaths(t, result.Files, []string{filepath.Join(link, filepath.Base(old))})
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
}

func TestCountModeWorks(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "one.txt", testNow.AddDate(0, 0, -8))
	writeTestFile(t, dir, "two.txt", testNow.AddDate(0, 0, -9))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--count", dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "FOUND", "FILES", "OLDEST FILE AGE", "2", "9 days", filepath.Base(dir))
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES", "DIRECTORIES", "2 files")
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestNamesModeWorks(t *testing.T) {
	dir := t.TempDir()
	old := writeTestFile(t, dir, "old.txt", testNow.AddDate(0, 0, -8))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--names", dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "FOUND", "FILE AGE", "8 days", displayPath(old))
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestCountModeConfirmsWhenNoOlderFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "new.txt", testNow.AddDate(0, 0, -2))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--count", dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "CLEAR", "No files need attention.")
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestNamesModeConfirmsWhenNoOlderFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "new.txt", testNow.AddDate(0, 0, -2))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--names", dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "CLEAR", "No files need attention.")
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestMissingDirectoryHandlingWorks(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--count", missing}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "CLEAR", "No files need attention.")
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	requireOutputContains(t, stderr.String(), "D I R S Q U A T", "WARN", "ISSUES", "    1", displayPath(missing))
}

func TestUnreadableDirectoryHandlingWorks(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read directories without owner permissions")
	}

	root := t.TempDir()
	unreadable := filepath.Join(root, "unreadable")
	if err := os.Mkdir(unreadable, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(unreadable, 0o755)
	})
	if err := os.Chmod(unreadable, 0); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--count", root}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "CLEAR", "No files need attention.")
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	requireOutputContains(t, stderr.String(), "D I R S Q U A T", "WARN", "ISSUES", "    1", displayPath(unreadable))
}

func TestPathsWithSpacesWork(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "path with spaces")
	old := writeTestFile(t, dir, "old file.txt", testNow.AddDate(0, 0, -8))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--days", "7", "--names", dir}, &stdout, &stderr, testNow)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr.String())
	}
	requireOutputContains(t, stdout.String(), "D I R S Q U A T", "FOUND", "FILE AGE", "8 days", displayPath(old))
	requireOutputExcludes(t, stdout.String(), "threshold", "OLDER FILES")
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func scanForTest(roots []string, followSymlinks bool, days int) ScanResult {
	scanner := Scanner{
		FollowSymlinks: followSymlinks,
		Now:            testNow,
	}
	return scanner.Scan(roots, days)
}

func writeTestFile(t *testing.T, dir, name string, modTime time.Time) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	return path
}

func createSymlink(t *testing.T, oldname, newname string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("symlink tests target macOS and Linux")
	}

	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatalf("create symlink %s -> %s: %v", newname, oldname, err)
	}
}

func requirePaths(t *testing.T, files []FileMatch, want []string) {
	t.Helper()

	if len(files) != len(want) {
		t.Fatalf("expected %d files, got %d: %#v", len(want), len(files), files)
	}

	for i := range want {
		if files[i].Path != want[i] {
			t.Fatalf("file %d: want %q, got %q", i, want[i], files[i].Path)
		}
	}
}
