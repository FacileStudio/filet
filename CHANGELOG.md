# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Every entry below was reconstructed from git history on 2026-08-24, so they
record what shipped rather than what was written down at the time.

## [Unreleased]
### Added
- go.err.nilerr rule: detects functions that handle an error but return nil in its place (the nilerr bug)
- Corresponding unit tests in rules_test.go
- Updated frozen rules list to include go.err.nilerr
- Added roast punchlines for the new rule

## [0.3.0] — 2026-08-10

### Changed

- `filet.yml` and `filet.yaml` come first in the discovery list, and are what
  `filet init` writes. A guidelines file is meant to be read, edited and argued
  with, so it should not be hidden. `.filet.yml` and `.filet.yaml` are still
  read, so an unrenamed repository keeps working with no coordinated upgrade.

## [0.2.0] — 2026-08-10

### Added

- `-format github` writes workflow commands that annotate the pull request
  diff, and `-format sarif` feeds GitHub code scanning.
- The JSON output carries the severity counts and the grade filet already
  computed, so a pipeline reads it instead of re-tallying in shell.
- `action.yml` installs filet, annotates, writes a graded job summary, and can
  post the run to an Antenne webhook signed with HMAC-SHA256. filet itself
  never opens a socket, so the linter stays runnable offline.

### Fixed

- `gen.nesting` applied every closing brace on a line before every opening one,
  and the clamp at zero hid the underflow, so the table-driven test idiom left
  the depth counter raised for the rest of the file. Braces now apply in order,
  and a pair opened and closed on one line scores nothing. Across five suite
  repositories, 83 findings became 31.
- `Grade()` divided by actual line count, so one info finding on a 37-line
  Dockerfile graded F. A floor of 1000 lines fixes it and removes the
  divide-by-zero.
- `go.global.mutable` no longer flags `var ErrX = errors.New(...)`, which is how
  the standard library declares `io.EOF`, and a wrapped package-doc line
  beginning with the word "package" no longer reads as a commented-out package
  clause.

## [0.1.0] — 2026-08-09

### Added

- First release. filet is a style checker, roaster and test runner.
- Per-directory layout contracts, checked against the paths they actually
  match.
- Output built around a parseable contract.
- MIT license and a CI workflow that runs filet against filet.

### Changed

- Config discovery stops at the repository root, and the contract is frozen.
- Moved to the FacileStudio organisation, with sharper comment rules and limits
  tuned against real codebases.

[Unreleased]: https://github.com/FacileStudio/filet/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/FacileStudio/filet/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/FacileStudio/filet/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/FacileStudio/filet/releases/tag/v0.1.0
