package checks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/rishisinu/Cascade-CI/internal/report"
	"github.com/rishisinu/Cascade-CI/rules"
)

// rulesets maps a file kind (see fileKind) to the semgrep configs that apply to it.
// Plain names are folders in the embedded rules/ directory; "p/..." and "r/..." are
// fetched from the Semgrep registry at scan time (needs network, not pinned).
var rulesets = map[string][]string{
	"go":             {"go", "agentic"},
	"python":         {"python", "agentic"},
	"javascript":     {"javascript", "agentic"},
	"bash":           {"bash"},
	"dockerfile":     {"dockerfile"},
	"github-actions": {"github-actions"},
	"rust":           {"p/rust"},
}

var extKinds = map[string]string{
	".go":  "go",
	".py":  "python",
	".js":  "javascript",
	".jsx": "javascript",
	".mjs": "javascript",
	".cjs": "javascript",
	".ts":  "javascript",
	".tsx": "javascript",
	".sh":  "bash",
	".rs":  "rust",
}

type semgrepOutput struct {
	Results []semgrepResult `json:"results"`
}

type semgrepResult struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line int `json:"line"`
	} `json:"start"`
	Extra struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
	} `json:"extra"`
}

func Semgrep(repo string, base string) ([]report.Finding, error) {
	files, err := changedFiles(repo, base) // Calls changed files passing in base to extract all files
	if err != nil {
		return nil, err
	}
	configs := configsFor(files) // Pass in all files to configsFor to get the configs for the files in the diff
	if len(configs) == 0 {
		return nil, nil // Nothing in this diff that we have rules for
	}

	rulesDir, err := extractRules() //Creates temp dir and extracts the rules from the embedded rules dir into that temp dir
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(rulesDir)

	args := []string{"scan", "--json", "--metrics=off", "--disable-version-check"}
	if base != "" {
		// Only report findings this PR introduced, not ones already on base.
		//Turn whatever base was passed into a commit sha using rev-parse, and pass that to semgrep as the baseline commit. This will make semgrep only report findings that are new in this PR,
		// not ones that were already present in base.
		sha, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", base+"^{commit}").Output()
		if err != nil {
			return nil, fmt.Errorf("base ref %q not found in %s (shallow clone?): %w", base, repo, err)
		}
		args = append(args, "--baseline-commit", strings.TrimSpace(string(sha)))
	}
	for _, c := range configs {
		if !strings.HasPrefix(c, "p/") && !strings.HasPrefix(c, "r/") {
			c = filepath.Join(rulesDir, c)
		}
		args = append(args, "--config", c)
	}
	args = append(args, ".")

	cmd := exec.Command("semgrep", args...)
	cmd.Dir = repo // --baseline-commit uses the git repo in the working directory
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("semgrep failed: %w: %s", err, stderr.String())
	}
	var completed semgrepOutput
	if err := json.Unmarshal(output, &completed); err != nil {
		return nil, fmt.Errorf("parsing semgrep output: %w", err)
	}
	// Process the semgrep results and convert them to our report.Finding
	findings := make([]report.Finding, 0, len(completed.Results))
	for _, s := range completed.Results {
		// check_id comes back prefixed with the config's path, e.g. "tmp.go.go-weak-hash"
		ruleID := s.CheckID[strings.LastIndex(s.CheckID, ".")+1:]
		findings = append(findings, report.Finding{
			ID:         fmt.Sprintf("%s:%s:%d", ruleID, filepath.ToSlash(s.Path), s.Start.Line),
			Tier:       0,
			SourceTool: report.ToolSemgrep,
			Severity:   semgrepSeverity(s.Extra.Severity),
			Category:   ruleID,
			Location:   report.Location{File: filepath.ToSlash(s.Path), Line: s.Start.Line},
			RawMessage: s.Extra.Message,
		})
	}

	return findings, nil
}

// changedFiles lists the files this PR added or modified, relative to the repo root.
// With no base, every tracked file counts as changed.
func changedFiles(repo string, base string) ([]string, error) {
	args := []string{"-C", repo, "ls-files", "-z"}
	if base != "" {
		// Three dots diffs from where the branch split off base, so changes merged
		// into base since then aren't counted. --diff-filter=d drops deleted files.
		args = []string{"-C", repo, "diff", "--name-only", "-z", "--diff-filter=d", base + "...HEAD"}
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("listing changed files: %w", err)
	}
	// -z separates names with NUL bytes, so file names containing spaces survive
	return strings.FieldsFunc(string(out), func(r rune) bool { return r == 0 }), nil
}

// configsFor basically determines which semgrep configs to run, without duplicates by looping thru all files in the diff
func configsFor(files []string) []string {
	seen := map[string]bool{}
	var configs []string
	for _, f := range files {
		for _, c := range rulesets[fileKind(f)] {
			if !seen[c] {
				seen[c] = true
				configs = append(configs, c)
			}
		}
	}
	return configs
}

// fileKind decides which rulesets apply to a path. Most files go by extension, but
// Dockerfiles and workflow files are recognised by name and location.
func fileKind(file string) string {
	file = filepath.ToSlash(file)
	name := strings.ToLower(path.Base(file))
	ext := path.Ext(name)
	// Here we basically have a special checl to see if the file is a workflow file or a dockerfile, otherwise we just return the extension
	switch {
	case strings.HasPrefix(file, ".github/workflows/") && (ext == ".yml" || ext == ".yaml"):
		return "github-actions"
	case name == "dockerfile" || strings.HasPrefix(name, "dockerfile.") || ext == ".dockerfile":
		return "dockerfile"
	}
	//append onto extKinds map to return the kind of file based on the extension, if it is not found it will return an empty string
	return extKinds[ext]
}

// extractRules writes the embedded rulesets to a temp directory, since semgrep can only
// read rules from disk. The caller removes the directory when done.
func extractRules() (string, error) {
	dir, err := os.MkdirTemp("", "cascade-rules-")
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(rules.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := rules.FS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("extracting rules: %w", err)
	}
	return dir, nil
}

// semgrepSeverity maps semgrep's levels onto ours. and we return the severity obj depending on what is returned
func semgrepSeverity(s string) report.Severity {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return report.SeverityCritical
	case "HIGH", "ERROR":
		return report.SeverityHigh
	case "MEDIUM", "WARNING":
		return report.SeverityMedium
	default:
		return report.SeverityLow
	}
}
