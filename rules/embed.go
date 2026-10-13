// Package rules embeds Cascade's semgrep rulesets into the binary, so they are
// available when cascade runs inside a repo that doesn't contain this folder.
package rules

import "embed"

// FS holds every ruleset, keyed by path: "go/security.yaml", "agentic/python.yaml", ...
//
//go:embed */*.yaml
var FS embed.FS
