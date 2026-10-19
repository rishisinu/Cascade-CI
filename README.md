# Cascade-CI

Cascade runs security checks against your code in tiers, from fast and deterministic to slow and deep. It prints a JSON report and sets its exit code so it can gate a CI job.

The tiers:

- **Tier 0** runs on every PR. It's cheap and deterministic, uses no LLM and needs no secrets, so it also works on PRs from forks.
- **Tier 1** is the agentic tier. It runs on a schedule or on demand, and an LLM triages and investigates findings across the whole repo.

Cascade doesn't build your code. You build it and pass the resulting binary in with `--binary`.

## Status

| Area | Status |
|---|---|
| CLI skeleton (`cascade`, `tier0 run`) | Done |
| Tier 0: gitleaks, semgrep, ELF checks | Done |
| Report file (`--output`) and exit codes | Done |
| Tier 1 orchestrator | In progress |
| Agent tools and path sandbox | In progress |
| LLM provider and agent loop | Not started |

## What's implemented

### Tier 0 (`cascade tier0 run`)

| Check | What it does |
|---|---|
| **gitleaks** | Scans the PR's commits (`base..HEAD`) for committed secrets |
| **semgrep** | Scans only the files the PR changed, using Cascade's built-in rulesets, and reports only findings the PR introduced (`--baseline-commit`) |
| **elf** | With `--binary`, checks a Linux binary for missing exploit mitigations: PIE, non-executable stack, RELRO, stack canaries and relative RPATH. Language-aware: checks that don't apply to Go or Rust are skipped, and non-ELF files are skipped with a warning |

The semgrep rulesets are embedded in the binary (`rules/`) and chosen by file type:

| Files | Rulesets |
|---|---|
| `.go` | go, agentic |
| `.py` | python, agentic |
| `.js .jsx .mjs .cjs .ts .tsx` | javascript, agentic |
| `.sh` | bash |
| `Dockerfile`, `*.dockerfile` | dockerfile |
| `.github/workflows/*.yml` | github-actions |
| `.rs` | Semgrep registry `p/rust` (needs network) |

The `agentic` rules cover risks in LLM/agent code, such as LLM output reaching code execution or SQL, untrusted input in system prompts, unsandboxed code tools, unsafe model loading and unbounded agent loops. Each ruleset has test fixtures in `testdata/semgrep/`.

### Reports and exit codes

- The report is always printed to stdout. Pass `--output <file>` to also write it to a file.
- If a check fails partway through, the report is still written, with status `error` and whatever the earlier checks found.
- The `report.Finding` struct already has the optional Tier 1 fields (`verdict`, `confidence`, `trace`, `remediation`).

| Exit code | Meaning |
|---|---|
| 0 | Nothing found |
| 1 | Findings |
| 2 | A check couldn't run (missing tool, bad ref, unreadable binary, bad flags) |

### Tier 1 groundwork (in progress, not yet compiling)

- `internal/agent/sandbox.go`: resolves every path the agent asks for against the repo root and rejects absolute paths, `..` and symlinks that point outside the repo.
- `internal/agent/tool.go`: a provider-independent, MCP-style `Tool` interface (name, description, JSON input schema) and a registry.
- `internal/agent/tools.go`: a `read_file` tool (line-numbered, paged at 400 lines) and a spec for a `grep` tool.
- `internal/tier1/tier1orchestrator.go`: a stub for `tier1.Run`.

## Usage

Requires `git`, `gitleaks` and `semgrep` on `PATH`.

```sh
go install github.com/rishisinu/Cascade-CI/cascade@latest

cascade tier0 run                                    # diff against origin/main
cascade tier0 run --repo ../my-service --base origin/main
cascade tier0 run --binary ./bin/server --output cascade-report.json
cascade tier0 run --base ""                          # scan the whole repository
```

| Flag | Default | Meaning |
|---|---|---|
| `--repo` | `.` | Repository to scan |
| `--base` | `origin/main` | Git ref the PR is compared against. Pass `""` to scan everything |
| `--binary` | | Linux binary built from the PR. Turns on the ELF checks |
| `--output` | | Also write the JSON report to this file |

Build the binary you pass to `--binary` with debug symbols: `-g` for C/C++, `debug = true` in the Cargo profile for Rust, and for Go, don't strip with `-ldflags="-s -w"`.

## Coming soon

The design and its reasoning are in [`docs/tier1-design-notes.md`](docs/tier1-design-notes.md). Planned work, roughly in order:

1. **Finish the agent tools:** `grep`, then `find_symbol`, `find_callers`, `run_semgrep_rule` and `git_log`. All read-only, scoped to the repo and with capped output. Tools are registered based on the languages in the repo.
2. **Tier 1 orchestrator (`cascade tier1 run`):**
   - Recon (no LLM): a full-repo Tier 0 pass run in-process, per-language analyzers (cppcheck, gosec, bandit, osv-scanner, ...), and a map of where untrusted input enters the code.
   - Triage (LLM): the agent investigates each seed within a turn and token budget, and can only report through a `report_finding` tool.
   - Validate (Go): check that every file:line the agent cites exists, then dedupe.
3. **LLM provider:** a small `Provider` interface with one tool-use loop, configured with `CASCADE_LLM_PROVIDER`, `CASCADE_LLM_MODEL` and `CASCADE_LLM_API_KEY`.
4. **Tier 0 polish:**
   - Fill in `CommitSHA` and `ToolVersions` in the report.
   - Warn when `--binary` has no debug info.
   - Add `cascade doctor` to check the required tools are installed.
5. **Config file** (`configs/`): choose which checks run and which severity fails the build.
6. **Distribution:** GitHub Actions workflow examples, then a container image with every analyzer at a pinned version.

Later:

- Dynamic analysis as a `verify(finding)` tool, run against a prebuilt `--image`.
- SARIF output for GitHub code scanning.
- A `cascade mcp` command that exposes the agent tools to MCP clients.

Still to decide: which LLM provider to support first, which language to build Tier 1 analyzers for first, and the config file format.

## License

See [LICENSE](LICENSE).
