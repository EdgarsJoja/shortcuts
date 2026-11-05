package shim

import (
	"fmt"
	"os"
	"path/filepath"
)

// LookPathOutside finds an executable named name in pathEnv, skipping the
// shim directory. It returns ok=false when no such command exists.
func LookPathOutside(name, shimDir, pathEnv string) (string, bool) {
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		if SamePath(dir, shimDir) {
			continue
		}
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0o111 == 0 {
			continue
		}
		return p, true
	}
	return "", false
}

// PathContains reports whether dir is present in pathEnv.
func PathContains(dir, pathEnv string) bool {
	for _, d := range filepath.SplitList(pathEnv) {
		if d == "" {
			d = "."
		}
		if SamePath(d, dir) {
			return true
		}
	}
	return false
}

// CheckPrecedence explains whether the shim directory is reachable and whether
// another command earlier in PATH shadows name. An empty string means clean.
func CheckPrecedence(name, shimDir, pathEnv string) string {
	dirs := filepath.SplitList(pathEnv)
	shimIdx := -1
	for i, dir := range dirs {
		if dir == "" {
			dir = "."
		}
		if SamePath(dir, shimDir) {
			shimIdx = i
			break
		}
	}

	if shimIdx < 0 {
		if other, ok := LookPathOutside(name, shimDir, pathEnv); ok {
			return fmt.Sprintf("shim directory %q is not in PATH; %q will run instead", shimDir, other)
		}
		return fmt.Sprintf("shim directory %q is not in PATH", shimDir)
	}

	for i := 0; i < shimIdx; i++ {
		dir := dirs[i]
		if dir == "" {
			dir = "."
		}
		if SamePath(dir, shimDir) {
			continue
		}
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return fmt.Sprintf("%q precedes the shim directory in PATH and shadows this shortcut", p)
		}
	}
	return ""
}

// SamePath compares two filesystem paths, resolving symlinks when both exist.
func SamePath(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err == nil {
		a = aa
	}
	bb, err := filepath.Abs(b)
	if err == nil {
		b = bb
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ra, erra := filepath.EvalSymlinks(a)
	rb, errb := filepath.EvalSymlinks(b)
	if erra == nil && errb == nil {
		return ra == rb
	}
	return false
}
