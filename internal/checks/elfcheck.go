package checks

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/rishisinu/Cascade-CI/internal/report"
)

// ELFHygiene inspects a compiled Linux binary for missing exploit mitigations: PIE,
// a non-executable stack, RELRO, stack canaries, and unsafe library search paths.
// Every finding fails the run, so each check only fires where the fix is reasonable
// for the language the binary was built from.
func ELFHygiene(binary string) ([]report.Finding, error) {
	f, err := elf.Open(binary)
	if err != nil {
		var formatErr *elf.FormatError
		if errors.As(err, &formatErr) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			// e.g. a Windows .exe or macOS binary from a cross-compile. Not a failure,
			// just nothing this check can inspect.
			fmt.Fprintf(os.Stderr, "elf-hygiene: skipping %s: not an ELF binary\n", binary)
			return nil, nil
		}
		return nil, fmt.Errorf("opening binary: %w", err)
	}
	defer f.Close()

	lang := binaryLanguage(f)
	var findings []report.Finding
	add := func(rule string, severity report.Severity, message string) {
		findings = append(findings, report.Finding{
			ID:         rule + ":" + binary,
			Tier:       "0",
			SourceTool: report.ToolElfHygiene,
			Severity:   severity,
			Category:   rule,
			Location:   report.Location{File: binary},
			RawMessage: message,
		})
	}

	// PIE: ET_DYN means the code can be loaded at a random address (ASLR).
	if f.Type != elf.ET_DYN {
		add("elf-not-pie", report.SeverityHigh, pieMessage(lang))
	}

	// NX: the GNU_STACK header must exist and must not be executable. With no header,
	// the kernel falls back to an executable stack for compatibility.
	stack := progHeader(f, elf.PT_GNU_STACK)
	if stack == nil || stack.Flags&elf.PF_X != 0 {
		add("elf-executable-stack", report.SeverityHigh,
			"The stack is executable, so a buffer overflow can run injected code directly. "+
				"Remove `-z execstack` from the link flags, and check for assembly files "+
				"missing a `.note.GNU-stack` section.")
	}

	// RELRO only matters for dynamically linked binaries, which have a GOT that can be
	// overwritten. Go is skipped: its PIE builds get partial RELRO, and pure Go code
	// doesn't call through the lazily bound GOT that full RELRO protects.
	if progHeader(f, elf.PT_DYNAMIC) != nil && lang != "go" {
		switch {
		case progHeader(f, elf.PT_GNU_RELRO) == nil:
			add("elf-no-relro", report.SeverityMedium,
				"No RELRO: the GOT stays writable, so a memory-write bug can redirect "+
					"function calls. Link with `-Wl,-z,relro,-z,now`.")
		case !bindNow(f):
			add("elf-partial-relro", report.SeverityLow,
				"Partial RELRO: symbols are bound lazily, so part of the GOT stays "+
					"writable. Add `-Wl,-z,now` to the link flags for full RELRO.")
		}
	}

	// Stack canaries are a C/C++ compiler feature. Go and Rust don't use them, so their
	// absence there isn't a problem.
	if lang == "c" {
		if has, known := hasSymbol(f, "__stack_chk_fail", "__stack_chk_guard"); known && !has {
			add("elf-no-stack-canary", report.SeverityMedium,
				"No stack canaries found, so stack overflows aren't detected at runtime. "+
					"Compile with `-fstack-protector-strong`. (A very small program may "+
					"legitimately have no functions that need one.)")
		}
	}

	// RPATH/RUNPATH entries that are empty or relative resolve against the current
	// directory, letting whoever controls it supply a malicious shared library.
	for _, tag := range []elf.DynTag{elf.DT_RPATH, elf.DT_RUNPATH} {
		paths, _ := f.DynString(tag)
		for _, p := range paths {
			for entry := range strings.SplitSeq(p, ":") {
				if !strings.HasPrefix(entry, "/") && !strings.HasPrefix(entry, "$ORIGIN") && !strings.HasPrefix(entry, "${ORIGIN}") {
					add("elf-relative-rpath", report.SeverityMedium, fmt.Sprintf(
						"%v contains %q, which resolves relative to the working directory. "+
							"Use absolute paths or $ORIGIN.", tag, entry))
				}
			}
		}
	}

	return findings, nil
}

// binaryLanguage guesses which toolchain produced the binary, so we only apply checks
// that make sense for it.
func binaryLanguage(f *elf.File) string {
	if f.Section(".go.buildinfo") != nil {
		return "go"
	}
	// Rust embeds paths to its standard library (/rustc/<hash>/library/...) in panic
	// messages, and those survive stripping.
	if rodata := f.Section(".rodata"); rodata != nil {
		if data, err := rodata.Data(); err == nil && bytes.Contains(data, []byte("/rustc/")) {
			return "rust"
		}
	}
	return "c"
}

func pieMessage(lang string) string {
	switch lang {
	case "go":
		return "elf check failed: this Go binary was built without PIE, so ASLR can't " +
			"randomize where its code is loaded. Add -buildmode=pie to the go build step " +
			"in your GitHub Actions workflow, e.g. `go build -buildmode=pie ./...`, or " +
			"set `GOFLAGS: -buildmode=pie` in that step's env."
	case "rust":
		return "elf check failed: this Rust binary isn't PIE, although rustc builds PIE " +
			"by default. Look for `-C relocation-model=static` or a custom target or " +
			"linker in your workflow or .cargo/config.toml."
	default:
		return "elf check failed: this binary isn't PIE, so ASLR can't randomize where " +
			"its code is loaded. Compile with -fPIE and link with -pie (add them to " +
			"CFLAGS/LDFLAGS in your GitHub Actions build step)."
	}
}

func progHeader(f *elf.File, t elf.ProgType) *elf.Prog {
	for _, p := range f.Progs {
		if p.Type == t {
			return p
		}
	}
	return nil
}

// bindNow reports whether the dynamic linker resolves every symbol at startup, which
// lets the GOT be made read-only afterwards (full RELRO).
func bindNow(f *elf.File) bool {
	if v, _ := f.DynValue(elf.DT_BIND_NOW); len(v) > 0 {
		return true
	}
	if v, _ := f.DynValue(elf.DT_FLAGS); len(v) > 0 && v[0]&uint64(elf.DF_BIND_NOW) != 0 {
		return true
	}
	if v, _ := f.DynValue(elf.DT_FLAGS_1); len(v) > 0 && v[0]&uint64(elf.DF_1_NOW) != 0 {
		return true
	}
	return false
}

// hasSymbol looks for any of names in the static and dynamic symbol tables. known is
// false when the binary has no symbols at all (fully stripped), so the answer can't
// be trusted either way.
func hasSymbol(f *elf.File, names ...string) (has bool, known bool) {
	static, _ := f.Symbols()
	dynamic, _ := f.DynamicSymbols()
	for _, s := range append(static, dynamic...) {
		if slices.Contains(names, s.Name) {
			return true, true
		}
	}
	return false, len(static)+len(dynamic) > 0
}
