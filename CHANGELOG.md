# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.16.0] — 2026-09-11

### Changed
- `filet check` now runs the file and Go-package rules across a bounded worker
  pool (one goroutine per logical core, capped at one per unit) instead of
  strictly sequentially. Each file's content hash is computed once at scan time
  and reused by every tier. Two latent races the parallel paths would otherwise
  expose are fixed: the disk-cache store is now mutex-guarded, and the shared Go
  toolchain root (`build.Default.GOROOT`) is set exactly once. Results stay
  deterministic: files keep their scan order and Go directories are sorted.

## [0.15.0] — 2026-09-11

### Added
- `filet check` and `filet roast` now cache findings on disk keyed by content
  hash, config and tool version, reusing them on later runs instead of
  re-parsing, re-typechecking and re-spawning language servers. The cache
  lives under `$XDG_CACHE_HOME/filet` (else `~/.cache/filet`) and
  self-invalidates when the config or a rule changes; `-no-cache`,
  `-cache-dir` and `cache: {enabled: false}` control it.

## [0.14.0] — 2026-09-10

### Added
- `filet clean` now removes comments inside a function body (`go.comment.inbody`),
  so it self-fixes the in-body finding instead of only reporting it. A comment
  trailing code on the same line is stripped without deleting the line; standalone
  prose, file headers, generated files and tool directives (`//nolint:`, `//go:`)
  are still preserved.

## [0.13.1] — 2026-09-10

### Changed
- filet now holds to its own defaults. `check.go`'s render dispatch moved to
  its own file, the nilerr scanner split out of `go_err_nilerr.go`, and the
  lipgloss/plain renderers share one column-width pass, so every file sits
  under `funcsPerFile` and `fileLines`.
- `goRoot()` derives GOROOT from `go env GOROOT` instead of the deprecated
  `runtime.GOROOT()`, which stays correct for a copied or `-trimpath` binary.
- Language-server teardown no longer swallows errors: pipe-closing failures
  are joined into the returned error, and a failed post-initialize
  notification makes `lspStart` close the server and fail instead of silently
  proceeding.
- Applied the `gopls` modernization passes surfaced by the LSP tier
  (`maps.Copy`, `strings.Cut`, `max`, range-over-int and `Named.Methods`).

### Fixed
- `go.err.nilerr` scan helpers were split with no behaviour change; the
  sentinel-gate canary suite is unchanged.

## [0.13.0] — 2026-09-10

### Changed
- `filet check` now fails on any finding, not just errors: the default gate is
  `info`, so the report must be clean for the build to pass. Set `failOn` in
  `filet.yml` to keep a looser gate.
- `filet clean` now also reformats Go source through the language's own
  formatter (the stdlib `go/format` engine, the same one `gofmt` drives), so a
  cleaned file satisfies the project's formatter gate. A file the formatter
  cannot parse keeps its line-level edits and is not failed. Gated on the new
  `format` config toggle (default on); only `.go` files are affected.

### Fixed
- `go.err.nilerr` no longer fires on a return that puts nil in the error slot
  when it sits under a sentinel gate — `if err == ErrNoRows` or
  `if errors.Is(err, ErrNoRows)`. Returning a default for a known "not found"
  sentinel is a deliberate result, not a swallowed error. An `else` branch of a
  sentinel gate is still flagged.

## [0.12.0] — 2026-09-10

### Added
- `filet clean [path]` applies the auto-fixable checks in place: trailing
  whitespace, trailing line comments, and whole lines of commented-out code.
  Each fix is gated on the same config toggle that gates its check; file
  headers, standalone prose comments and tool directives are left alone. Pass
  `-dry-run` to preview the edits without writing. Covers the mechanical subset
  that `balai` used to handle, so comment cleanup now lives in the checker that
  reports it.

## [0.11.2] — 2026-09-10

### Fixed
- The mason LSP-server directory is resolved through XDG (honours
  `$XDG_DATA_HOME` when set) instead of hardcoding `~/.local/share`, so a
  Neovim/mason fleet living under a non-default data dir is still found.
  Mirrors the existing global-config resolution for `$XDG_CONFIG_HOME`.

## [0.11.1] — 2026-09-10

### Changed
- LSP findings render without decorative brackets, and the docs link is now a
  muted `docs: <url>` tail in the terminal instead of `[docs: <url>]` buried in
  the message. The link stays a structured `docs` field in JSON, so the line,
  SARIF and GitHub formats still carry it.

## [0.11.0] — 2026-09-10

### Added
- `lsp.*` findings now carry the exact column of each diagnostic, so `file:line:col`
  appears in every output format instead of line-only.
- LSP messages fold in two more fields servers already sent but filet dropped:
  the `codeDescription` docs link (TypeScript, clangd, Pyright set it), and the
  `deprecated` / `unnecessary` tags. Both appear only when a server provides them,
  so plain diagnostics read exactly as before.

## [0.10.1] — 2026-09-09

### Fixed
- `go.leak.resource` now detects leaks in `-trimpath` release builds. The
  flag strips `runtime.GOROOT()`, so the standard-library importer could not
  resolve `os` and the shipped binary silently missed every leak. filet now
  falls back to `$GOROOT` and `go env GOROOT`, and reports a visible
  "could not run" finding only when no GOROOT can be found.

## [0.10.0] — 2026-09-09

### Added
- The language-server (LSP) tier, on by default. For every file whose language
  has a server recipe (Go, Rust, TypeScript, Svelte), filet spawns the server,
  collects its `publishDiagnostics` and folds them into `lsp.error` / `lsp.warn`
  / `lsp.info` findings. A missing, unsupported or unresponsive server degrades
  to a single `lsp.unavailable` info finding — never a crash, never silence.
  `lsp.fail: true` promotes every `lsp.*` finding to error so the tier gates a
  build. Servers are overridden per extension in `lsp.servers`; a global
  `~/.config/filet/filet.yml` can add or replace recipes. One server is spawned
  per distinct recipe, so a single TypeScript server serves `.ts/.tsx/.js/.jsx`.
- A tree-sitter analysis tier, now on by default. Files in a language with a
  vendored grammar (`.rs` today) get their shape rules — `ts.nesting`,
  `ts.func.long`, `ts.func.params`, `ts.func.statements`, `ts.func.complexity` —
  measured on a real parse tree instead of brace-counting. Each grammar's
  generated C is vendored and bound with cgo, so the tier is deterministic and
  offline. Set `treesitter.enabled: false` to fall back to brace-counting.
- `-format lipgloss` is now reachable: the lipgloss renderer is wired into the
  format flag and documented in `--help`.
- `go.leak.resource` now actually runs: the leak rule is type-checked per
  package (one shared `types.Info` per package, so a file resolves symbols its
  siblings define) and keeps the partial type info `go/types` resolves even when
  a third-party import is unavailable. It previously no-op'd on real code.

### Fixed
- `go.leak.resource` now hands out trustworthy findings: closed detection is no
  longer defer-only — an explicit `errors.Join(werr, f.Close())` or a receiver
  chain like `resp.Body.Close()` counts — and borrowed handles (`os.Stdout`, a
  type assertion, a field read) are no longer flagged as acquisitions.
- The LSP server conversation is fully bounded. A server that stalls on
  `initialize` or never publishes diagnostics is killed, with its whole process
  group, after a timeout instead of hanging the run; partial diagnostics are
  kept when only some files report.
- A malformed global `~/.config/filet/filet.yml` no longer fails every check
  run; its `lsp.servers` layer is ignored instead.
- `go.err.nilerr` no longer fires on a value guard: it reports only when the
  compared identifier is the error slot of an error-returning call, so
  `if item != nil { return item, nil }` stops being flagged as a swallowed
  error.
- Tree-sitter shape metrics (`ts.func.*`) are no longer computed twice per
  function, and multi-line block comments no longer count their continuation
  lines as code.
- `-format` is validated after the path, so `filet check . -format bogus` is
  rejected like the flag-before-path form.
- The GitHub action's default install version and the README pins are back in
  step with the newest tag.

## [0.9.0] — 2026-09-09
### Added
- Styled help output (`filet --help`): cyan section headers, bold commands, aligned descriptions, media-aware (plain off a terminal).
- Golden tests for the CI (JSON) and SARIF renderers.
- Opt-in lipgloss report renderer, not yet wired into the default path.

### Changed
- Extracted Rust raw-string scanning out of the line scanner into `rust.go`.
- Moved TODO-marker detection from `text.go` into `rules_generic.go`.

### Fixed
- `.gitignore` rule that silently matched the `internal/filet/` directory, which hid new files there from version control. Anchored the pattern to the repo root.

## [0.7.1] — 2026-09-08
### Changed
- Release v0.7.1: no functional changes; updates CI and local quality gate.

## [0.7.0] — 2026-09-08
### Added
- go.err.nilerr rule: detects functions that handle an error but return nil in its place (the nilerr bug)
- Corresponding unit tests in nilerr_test.go and rules_test.go
- Updated frozen rules list to include go.err.nilerr
- Added roast punchlines for the new rule

### Changed
- Refactored go leak detection: moved `go_leaks.go`, `deferred.go`, `go_findings.go`, `types.go`, `receivers.go`, `nilerr_test.go`, and `resource_leak_test.go` into separate files for better maintainability
- `go.leak.resource` now correctly detects Close() calls inside deferred anonymous functions, fixing false positives on idiomatic patterns like `defer func() { f.Close() }()`

## [0.6.0] — 2026-09-08
### Added
- go.err.nilerr rule: detects functions that handle an error but return nil in its place (the nilerr bug)
- Corresponding unit tests in rules_test.go
- Updated frozen rules list to include go.err.nilerr
- Added roast punchlines for the new rule

### Changed
- `go.leak.resource` now detects Close() calls inside deferred anonymous functions, fixing false positives on idiomatic patterns like `defer func() { f.Close() }()`

## [0.5.0] — 2026-09-08
### Changed
- `filet.yml` and `filet.yaml` come first in the discovery list, and are what
  `filet init` writes. A guidelines file is meant to be read, edited and argued
  with, so it should not be hidden. `.filet.yml` and `.filet.yaml` are still
  read, so an unrenamed repository keeps working with no coordinated upgrade.

## [0.4.0] — 2026-09-08
### Changed
- docs: add changelog entry for go.leak.resource deferred func fix

## [0.3.1] — 2026-08-24
### Added
- 📝 docs: add a changelog, backfilled from the tag history

## [0.3.0] — 2026-08-10
### Added
- First release. filet is a style checker, code roaster and test runner.
- Per-directory layout contracts, checked against the paths they actually
  match.
- Output built around a parseable contract.
- MIT license and a CI workflow that runs filet against filet.

## [0.2.0] — 2026-08-10
### Added
- `-format github` writes workflow commands that annotate the pull request
  diff, and `-format sarif` feeds GitHub code scanning.
- The JSON output carries the severity counts and the grade filet already
  computed, so a pipeline reads it instead of re-tallying in shell.
- `action.yml` installs filet, annotates, writes a graded job summary, and can
  post the run to an Antenne webhook signed with HMAC-SHA256. filet itself
  never opens a socket, so the linter stays runnable offline.

## [0.1.0] — 2026-08-09
### Added
- First release. filet is a style checker, code roaster and test runner.
- Per-directory layout contracts, checked against the paths they actually
  match.
- Output built around a parseable contract.
- MIT license and a CI workflow that runs filet against filet.

[Unreleased]: https://github.com/FacileStudio/filet/compare/v0.14.0...HEAD
[0.14.0]: https://github.com/FacileStudio/filet/compare/v0.13.1...v0.14.0
[0.13.1]: https://github.com/FacileStudio/filet/compare/v0.13.0...v0.13.1
[0.13.0]: https://github.com/FacileStudio/filet/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/FacileStudio/filet/compare/v0.11.2...v0.12.0
[0.11.2]: https://github.com/FacileStudio/filet/compare/v0.11.1...v0.11.2
[0.11.1]: https://github.com/FacileStudio/filet/compare/v0.11.0...v0.11.1
[0.11.0]: https://github.com/FacileStudio/filet/compare/v0.10.1...v0.11.0
[0.10.1]: https://github.com/FacileStudio/filet/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/FacileStudio/filet/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/FacileStudio/filet/compare/v0.7.1...v0.9.0
[0.7.1]: https://github.com/FacileStudio/filet/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/FacileStudio/filet/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/FacileStudio/filet/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/FacileStudio/filet/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/FacileStudio/filet/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/FacileStudio/filet/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/FacileStudio/filet/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/FacileStudio/filet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FacileStudio/filet/releases/tag/v0.1.0