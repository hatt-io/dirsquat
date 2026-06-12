package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Scanner struct {
	FollowSymlinks bool
	Now            time.Time
}

type ScanResult struct {
	Roots    []RootResult
	Files    []FileMatch
	Warnings []ScanWarning
}

type RootResult struct {
	Path  string
	Count int
}

type FileMatch struct {
	Root    string
	Path    string
	ModTime time.Time
}

type ScanWarning struct {
	Path string
	Err  error
}

func (s Scanner) Scan(roots []string, days int) ScanResult {
	now := s.Now
	if now.IsZero() {
		now = time.Now()
	}

	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	result := ScanResult{
		Roots: make([]RootResult, 0, len(roots)),
	}

	for _, root := range roots {
		rootResult := RootResult{Path: root}
		visited := make(map[string]struct{})
		s.scanRoot(root, cutoff, &rootResult, &result, visited)
		result.Roots = append(result.Roots, rootResult)
	}

	return result
}

func (s Scanner) scanRoot(root string, cutoff time.Time, rootResult *RootResult, result *ScanResult, visited map[string]struct{}) {
	info, err := os.Lstat(root)
	if err != nil {
		result.addWarning(root, err)
		return
	}

	if isHiddenName(filepath.Base(root)) {
		return
	}

	if info.Mode()&os.ModeSymlink != 0 {
		if !s.FollowSymlinks {
			return
		}

		info, err = os.Stat(root)
		if err != nil {
			result.addWarning(root, err)
			return
		}
	}

	if !info.IsDir() {
		result.addWarning(root, errors.New("not a directory"))
		return
	}

	s.scanDir(root, cutoff, rootResult, result, visited)
}

func (s Scanner) scanDir(dir string, cutoff time.Time, rootResult *RootResult, result *ScanResult, visited map[string]struct{}) {
	if isHiddenName(filepath.Base(dir)) {
		return
	}

	realPath, err := filepath.EvalSymlinks(dir)
	if err != nil {
		result.addWarning(dir, err)
		return
	}

	realPath, err = filepath.Abs(realPath)
	if err != nil {
		result.addWarning(dir, err)
		return
	}

	if _, ok := visited[realPath]; ok {
		return
	}
	visited[realPath] = struct{}{}

	entries, err := os.ReadDir(dir)
	if err != nil {
		result.addWarning(dir, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		if isHiddenName(name) {
			continue
		}

		child := filepath.Join(dir, name)
		if entry.Type()&os.ModeSymlink != 0 {
			s.scanSymlink(child, cutoff, rootResult, result, visited)
			continue
		}

		if entry.IsDir() {
			s.scanDir(child, cutoff, rootResult, result, visited)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			result.addWarning(child, err)
			continue
		}

		if !info.Mode().IsRegular() {
			continue
		}

		if info.ModTime().Before(cutoff) {
			rootResult.Count++
			result.Files = append(result.Files, FileMatch{
				Root:    rootResult.Path,
				Path:    child,
				ModTime: info.ModTime(),
			})
		}
	}
}

func (s Scanner) scanSymlink(path string, cutoff time.Time, rootResult *RootResult, result *ScanResult, visited map[string]struct{}) {
	if !s.FollowSymlinks {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		result.addWarning(path, err)
		return
	}

	if info.IsDir() {
		s.scanDir(path, cutoff, rootResult, result, visited)
	}
}

func (r *ScanResult) addWarning(path string, err error) {
	r.Warnings = append(r.Warnings, ScanWarning{
		Path: path,
		Err:  err,
	})
}
