package agent

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolvePath(repo, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("Can only use a relative file path to repo, not %s", repo)
	}
	root, err := repoRoot(repo) //confirms repo is an abs path and evals symlinks

	if err != nil {
		return "", err
	}
	pat, err := filepath.EvalSymlinks(filepath.Join(root, path)) //Evals sym link with our rel path added nto abs
	if err != nil {
		return "", err
	}
	r, err := filepath.Rel(root, pat)
	// If r = .., then pat is the parent to root, or if it starts with .. then it is a sib directory
	if err != nil || r == ".." || strings.HasPrefix(r, "..") {
		return "", fmt.Errorf("path is outside repo", path)
	}

	return pat, nil
}

func repoRoot(repo string) (string, error) {
	abs, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)

}
