# Cascade design notes: tier scope and Tier 1 planning

Reference notes from the 2026-10-14 planning discussion. Nothing in here is implemented yet unless it says so.

---

## 1. Tier scope

- **Two tiers only.** The old decompilation tier is dropped. Cascade has the source, so there is nothing to gain from decompiling a binary built from it.
- **Tier 0** (implemented): runs on every PR. It's deterministic, uses no agent and needs no secrets, so it also works on fork PRs, which don't get repo secrets.
  - gitleaks: secrets in the PR's commits (`base..HEAD`)
  - semgrep: changed files only, reporting only findings the PR introduced (`--baseline-commit`)
  - elf: exploit mitigation checks on `--binary` (PIE, NX, RELRO, canaries, RPATH)
- **Tier 1** (planned): the agentic tier. It runs on a cron schedule plus `workflow_dispatch` and reads `CASCADE_LLM_API_KEY` from a repo secret.
  - **v1 is static only** (decided). The agent reasons over source code with deterministic tools.
  - Dynamic analysis comes later (v1.5+), as a `verify(finding)` tool rather than open-ended pentesting.

### Why static first, and why dynamic still has a place later
An agent reading the source finds almost everything it would find by attacking the running app, and does it more cheaply. Running the binary adds two things:
1. **Proof.** A crashing input or a working proof of concept turns "maybe" into "confirmed". This matters because LLM static findings have high false-positive rates.
2. **Bug classes that only show up at runtime in C/C++,** such as memory corruption, integer overflow and use-after-free, where ASan plus fuzzing beats reading the code.

---

## 2. Responsibilities and config split

- **The user owns the build and the environment. Cascade owns the analysis.**
- Cascade never builds anything. It takes a built binary (`--binary`) and, for dynamic runs later, a prebuilt image reference (`--image`). It never takes a Dockerfile.
- The **workflow YAML** decides *when* things run (triggers, cron).
- **Cascade's own config file** (`configs/`, not created yet) decides *what* runs: which checks are enabled and what severity fails the build. This keeps Cascade usable outside GitHub.
- The LLM key comes in as the `CASCADE_LLM_API_KEY` env var. There's no GitHub-specific code for it, so the provider is the dev's choice.

---

## 3. Runtime assumptions (for dynamic analysis later; record them now)

- **Exactly one target under test** (`--image`). That's the container that gets resource limits, hooks and monitoring.
- **Monolithic repo, no orchestration by Cascade.** External dependencies like a DB or cache are the user's job. They must already be running and reachable over a network the sandbox can join. In GitHub Actions, the documented way is the workflow's `services:` key.
- **Dependencies must be disposable test instances.** Never production data, credentials or live external APIs: dynamic analysis hammers the app with generated input (junk rows, rate-limit bans, real charges or emails). A later nicety: refuse to run on config that looks like production.
- **Networking gotcha:** `localhost` inside the target container is the container itself. A no-network sandbox (the safe default) means the app can't reach its DB.
- `--compose-file` for supporting services is **out of scope for v1** (networking, startup order, health checks).
- **The binary and the image must match.** Tier 0 checks `--binary` while dynamic Tier 1 runs whatever is inside `--image`. Either hash-compare them or document that both must come from the same build. Keep both as explicit, separate CLI inputs.

---

## 4. Debug symbols (`-g`): tell users

The docs must tell users to build the binary they pass to `--binary` with debug symbols:

| Language | How |
|---|---|
| C/C++ | `CFLAGS="-g"` (e.g. `make CFLAGS="-g"`) |
| Rust | `debug = true` in the Cargo profile, or `-C debuginfo=2` |
| Go | Debug info is included by default. Don't strip it with `-ldflags="-s -w"` |

Where to put this:
- README, in a Setup / GitHub Actions section next to the workflow YAML
- the `--binary` flag help text (`cmd/root.go`, `cmd/cli.go`)
- **a warning at runtime:** the ELF check already opens the binary, so it can warn on stderr when there's no `.debug_info` section. Make it a warning, not a finding, so it doesn't fail Tier 0.

Debug symbols also help the existing canary check (`hasSymbol` can't tell anything when a binary is fully stripped). Later, they give file:line in sanitizer and debugger traces.

---

## 5. GitHub Actions usage (draft YAML)

Flag names are **`--base` and `--binary`** (decided). Earlier drafts used `--diff-base` / `--binary-path`; those are dropped.

### Tier 0: every PR
```yaml
# .github/workflows/cascade-tier0.yml
name: Cascade Tier 0
on:
  pull_request:

permissions:
  contents: read

jobs:
  tier0:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0          # Cascade needs history to diff against the base branch

      # The user's own build. Build with debug symbols (-g).
      - name: Build
        run: make CFLAGS="-g"

      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - name: Install Cascade
        run: go install github.com/rishisinu/Cascade-CI/cascade@latest
      # gitleaks and semgrep also need installing here (see section 8)

      - name: Cascade Tier 0
        run: cascade tier0 run --base origin/${{ github.base_ref }} --binary ./build/app --output cascade-report.json

      - uses: actions/upload-artifact@v4
        if: always()              # keep the report even when the check fails
        with:
          name: cascade-tier0-${{ github.event.pull_request.head.sha }}
          path: cascade-report.json
```
- Blocking: `cascade` exits non-zero on findings. The repo owner marks the check as required in branch protection, which is a GitHub setting, not part of the YAML.

### Tier 1: scheduled, plus manual trigger
```yaml
# .github/workflows/cascade-tier1.yml
name: Cascade Tier 1
on:
  schedule:
    - cron: "0 6 * * 1"          # Mondays 06:00 UTC, dev picks this
  workflow_dispatch:              # "Run workflow" button for on-demand runs

permissions:
  contents: read

jobs:
  tier1:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: make CFLAGS="-g"
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - name: Install Cascade
        run: go install github.com/rishisinu/Cascade-CI/cascade@latest

      - name: Cascade Tier 1
        env:
          CASCADE_LLM_API_KEY: ${{ secrets.CASCADE_LLM_API_KEY }}
        run: cascade tier1 run --binary ./build/app --output cascade-report.json

      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: cascade-tier1-${{ github.sha }}
          path: cascade-report.json
```

### Gotchas
- Scheduled workflows only run from the default branch. GitHub disables them after 60 days of repo inactivity on public repos.
- Fork PRs get no secrets, which is fine for Tier 0 (agent-free) but rules out anything that needs the LLM key.
- On `pull_request` events, `github.sha` is GitHub's temporary merge commit, not the PR's head commit. Use `github.event.pull_request.head.sha` when you need the PR's commit.
- `go install ...@latest` is fine during development. Later, publish pinned release binaries or an image.
- A `/cascade tier1` PR-comment trigger (`issue_comment`) is possible but tricky: it runs the default branch's workflow and has to check out the PR code itself. Deferred.
- Later polish: a reusable workflow or composite action, so users add 3 lines instead of pasting YAML.
- Possible later addition: SARIF output uploaded to GitHub code scanning (`security-events: write`) for inline annotations. Works for both tiers.

---

## 6. Gaps in the current Tier 0 code

| # | Gap | Decision / note |
|---|---|---|
| 1 | Flag names in the YAML don't match the code | **Keep `--base` and `--binary`**; fix the YAML |
| 2 | No report file: JSON only goes to stdout | **Add `--output` to `tier0 run`.** Write the file in the CLI layer (`cmd/cli.go`), so `tier0.Run` keeps returning `report.Run` and doesn't handle files. Write it *before* returning the "N findings" error, so failing runs still produce the artifact. Without `--output`, keep printing to stdout. |
| 3 | Install path in the docs (`cmd/cascade`; the actual main package is `cascade/`) | Not important; leave it |
| 4 | `Run.CommitSHA` and `Run.ToolVersions` never filled in | Fill them in. ToolVersions = versions of the external programs (see section 8) |
| 5 | No severity threshold: any finding fails | Belongs in the future `configs/` file |
| 6 | `Finding.Tier` is a string (`"0"`), `Run.Tier` is an int | Make them consistent |

Also planned: `cascade doctor` (or `tier0 doctor`) to check that the required tools are installed.

---

## 7. Tier 1 design (static only, v1)

### Flow
```
1. Recon (Go, no LLM)
   tier0 seeds: reuse a saved Tier 0 run, or run tier0.Run(repo, "", binary) (full repo)
   detect languages (reuse fileKind from internal/checks/semgrep.go)
   run that language's analyzers deterministically -> more seeds
   map the attack surface: where untrusted input enters (HTTP handlers, argv, stdin,
   sockets, file parsing) and where it can do damage (exec, SQL, memcpy, deserialization)

2. Triage loop (LLM)
   for each seed or hotspot, within a turn/token budget:
     agent investigates with read-only, repo-scoped tools
     agent ends by calling report_finding{verdict, confidence, trace[], remediation}

3. Validate (Go)
   check that every cited file:line exists and matches, dedupe by ID
   -> report.Run{Tier: 1, Findings: ...}
```

### What keeps it close to deterministic
- temperature 0
- a fixed, small tool set
- budgets per seed and in total (`--max-turns`, `--max-tokens`, maybe `--max-cost`)
- schema-only output: the agent can only report through the `report_finding` tool
- Go code checks evidence before a finding is accepted
- dedupe by finding ID

### Tools: who runs what
- **Running analyzers is Cascade's job**, done in recon. Cascade already knows each file's language, so the model shouldn't have to notice "this is C, run cppcheck". That costs turns and is error-prone.
  - candidates: cppcheck / flawfinder (C/C++), gosec / govulncheck (Go), bandit (Python), osv-scanner (dependencies), full-repo semgrep with broader taint rules
- **The agent gets navigation tools that are aware of language:** `read_file(path, lines)`, `grep`, `find_symbol`, `find_callers`, `run_semgrep_rule(path, rule)`, `git_log(path)`. Backends can differ per language behind one tool name.
- **Register tools at startup based on the repo's languages.** If the repo has no Python, no Python tools appear: fewer tools, fewer wrong calls, less context.

### Tool definitions in an MCP style
Use MCP's tool definition format (name, description, JSON input schema). It's the same shape every LLM provider's tool-use API takes. **Don't run an actual MCP server and client in v1**; a Go interface is enough:

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() json.RawMessage
    Languages() []string // nil = always available
    Run(ctx context.Context, input json.RawMessage) (string, error)
}
```
Because the definitions already match MCP, a `cascade mcp` command could expose the same tools to Claude Code and similar clients later.

### Threat model for the agent itself
Repo content is **untrusted**: comments, strings and docs can carry prompt injection, the same threat model as `rules/agentic/`. Therefore:
- tools are read-only
- every path is resolved against the repo root; reject `..` and symlinks pointing outside
- cap the size of tool output
- the agent gets no shell
- the only network access is the LLM API
- tool subprocesses never see `CASCADE_LLM_API_KEY`
- treat seeds and earlier reports as leads, not as trusted facts

### Report changes
- Extend `report.Finding` with optional Tier 1 fields: `verdict` (confirmed / likely / false-positive), `confidence`, `trace[]` (source to sink, with file:line), `remediation`.
- Tier 1 is scheduled, so there's no PR to block. Output a JSON artifact for now, SARIF later.

### LLM provider
A small `Provider` interface with one tool-use loop; implement one provider first. Selection via `CASCADE_LLM_PROVIDER` / `CASCADE_LLM_MODEL` plus `CASCADE_LLM_API_KEY`.

---

## 8. External tool versions and dependencies

There are two kinds of dependency:

| Kind | Examples | Pinned by |
|---|---|---|
| Go libraries compiled into `cascade` | cobra, pflag | `go.mod` / `go.sum` |
| Separate programs Cascade runs with `exec.Command` | gitleaks, semgrep (Python), later cppcheck, gosec, bandit... | **nothing yet** |

- `go.mod` can't pin the second kind. Today the user installs them however they like, so two runs can use different versions and give different results.
- `Run.ToolVersions` **records** what actually ran (`gitleaks version`, `semgrep --version`), so a report can be explained later. Users don't configure it.
- The semgrep registry config `p/rust` is fetched at scan time and isn't pinned either.

How to ship them:
- **Now:** document the install steps in the workflow YAML, and add `cascade doctor` to check the tools are on PATH and warn when a version is outside the tested range.
- **Later (recommended):** publish a container image, e.g. `ghcr.io/rishisinu/cascade:vX.Y`, with Cascade and every analyzer at pinned versions. This matters more for Tier 1, which adds an analyzer per language.
- **Avoid:** letting users configure tool versions. Cascade decides what it was tested against.

---

## 9. Seeding Tier 1 from Tier 0

### Decision (final)
**No saved runs, no hash, no cache.** Tier 1's first step is a full-scope Tier 0 pass, run in-process: `tier0.Run(repo, "", binary)`.
- Tier 0 is cheap and deterministic. Tier 1 is weekly and the code has changed since any PR run, so the checks would need to be recomputed anyway.
- `--output` stays on both commands so users can read the findings (and upload them as artifacts). Nothing reads those files back.
- No GitHub-specific code, no `GITHUB_TOKEN` / `actions: read`, and local runs work the same way.
- How the full-scope results are used:
  - **semgrep** findings are the main seeds for the agent's triage loop.
  - **gitleaks** and **ELF** findings go straight into the Tier 1 report. The agent has little to investigate in a leaked secret or a missing linker flag.
- Watch out: with `--base ""`, gitleaks scans the whole git history. Secrets that were committed and later removed show up, which is still valid (they're in the history) but noisy. Consider gitleaks `dir` mode for Tier 1, or make full history configurable.

### Options considered and rejected

### Problems found with the original "artifact named by SHA" idea
1. Tier 0 runs on PRs, while Tier 1 runs on `main` HEAD, so no Tier 0 run happened at that commit.
2. `github.sha` on PR events is a temporary merge commit, and squash/rebase merges create new SHAs on `main`.
3. PR runs are diff-scoped (changed files, new findings only), so they don't describe the whole repo.
4. Artifacts expire (90 days by default).


### Saved runs keyed by hash (rejected)
- Proposal: an optional save flag on Tier 0 writes the run with a hash in the filename, and Tier 1 reuses a matching run or falls back to running Tier 0 itself.
- Why rejected:
  - The key would need tree hash + scope + binary hash + versions.
  - Only full-scope runs could be reused, so PR runs never qualify.
  - CI runners are wiped between jobs, so it would need `actions/cache` plus an extra `push: main` Tier 0 job, because PR-branch caches can't be read from `main`.
  - All of that to save a few seconds or minutes per weekly run.

---

## 10. Open decisions
- [ ] LLM provider to implement first
- [ ] First language for Tier 1 analyzers (C/C++ fits the `-g` focus; Go has the strongest existing rulesets)
- [ ] Format of the config file in `configs/` (severity threshold, enabled checks)

## Suggested build order
1. `--output` on `tier0 run`, plus filling in `CommitSHA` / `ToolVersions` (small)
2. `tier1 run` skeleton that calls `tier0.Run(repo, "", binary)` as its first step (section 9)
3. The `Tool` registry and repo-scoped file tools
4. Provider interface and agent loop
5. `report_finding` validation and the Tier 1 report fields
