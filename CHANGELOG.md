# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `-format lipgloss` is now reachable: the lipgloss renderer is wired into the format flag and documented in `--help`.

### Fixed
- `go.err.nilerr` no longer fires on a value guard: it now reports only when the compared identifier is the error slot of an error-returning call, so `if item != nil { return item, nil }` stops being flagged as a swallowed error.
- `go.leak.resource` no longer fails silent: when type information is unavailable (real cross-module files), it reports an info finding instead of quietly doing nothing.
- `-format` is validated after the path, so `filet check . -format bogus` is rejected like the flag-before-path form.
- The GitHub action's default install version and the README pins are back in step with the newest tag (v0.8.0).

## [0.8.0] — 2026-09-09
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

[Unreleased]: https://github.com/FacileStudio/filet/compare/v0.7.1...HEAD
[0.7.1]: https://github.com/FacileStudio/filet/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/FacileStudio/filet/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/FacileStudio/filet/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/FacileStudio/filet/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/FacileStudio/filet/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/FacileStudio/filet/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/FacileStudio/filet/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/FacileStudio/filet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FacileStudio/filet/releases/tag/v0.1.0