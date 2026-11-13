# filet

[![ci](https://github.com/FacileStudio/filet/actions/workflows/ci.yml/badge.svg)](https://github.com/FacileStudio/filet/actions/workflows/ci.yml)

A single Go binary that checks your coding style, roasts your code, roasts your Dockerfiles,
and runs your test suites. One dependency, `goccy/go-yaml`, so the config file can have comments.
A composite GitHub action ships with it — see [In CI](#in-ci).

```
filet check   [path]   style, quality and architecture rules
filet roast   [path]   the same rules, with commentary
filet docker  [path]   every Dockerfile it can find, roasted
filet test    [path]   detect and run the project's test suites
filet init    [path]   write a commented filet.yml (-preset relaxed|epitech)
filet rules            list every rule id
```

`droast` is a symlink to `filet` and behaves exactly like `filet docker`.

## Install

```sh
mise run install          # go install + the droast symlink
# or
go install github.com/FacileStudio/filet@latest
ln -sf "$(go env GOPATH)/bin/filet" "$(go env GOPATH)/bin/droast"
```

## Configuration

`filet init` writes a commented starter file. `filet` then walks up from the target directory
looking for `filet.yml`, `filet.yaml`, `filet.yml` or `filet.yaml`, and **stops at the repository
root** — a config living above your repository would apply on your machine and vanish in CI, where
only the repository is checked out. Everything it finds is merged over the defaults, and the
directory holding the config becomes the project root.

```yaml
preset: epitech           # optional starting point, see below

ignore: [.git, node_modules, vendor, dist, testdata]
extensions: [.go, .ts, .svelte]

limits:
  fileLines: 250
  funcsPerFile: 8         # 0 disables the budget; _test.go files are exempt
  funcLines: 30           # Go only, blank and comment lines excluded
  funcStatements: 25      # Go only
  lineLength: 120
  params: 5
  returns: 3
  nesting: 4
  complexity: 10          # cognitive, not cyclomatic
  structFields: 15
  interfaceMethods: 5

architecture:
  requiredDirs: [apps/api, apps/client]
  forbiddenDirs: [internal/utils, pkg/common]
  maxDepth: 4
  fileNamePattern: '^[a-z0-9_]+\.go$'
  requiredFiles:
    "apps/api/modules/*": [router.go]
  forbiddenImports:
    internal/domain: [net/http, database/sql]

style:
  banInlineComments: true
  banTODO: true
  requireDocComments: true
  banGlobalMutable: true
  banInit: true
  banTrailingSpace: true
  strict: false           # the go.strict.* family, see "Strict Go rules"

disabled: [gen.line.long]
failOn: error
```

### Presets

`preset:` seeds the limits before your own keys are applied, so anything you write explicitly still
wins. Five exist:

| Preset | funcsPerFile | funcLines | funcStatements | fileLines | nesting | lineLength | complexity |
|---|---|---|---|---|---|---|---|
| `default` | 8 | 30 | 25 | 250 | 4 | 120 | 10 |
| `relaxed` | off | 60 | 40 | 400 | 5 | 120 | 15 |
| `epitech` | 5 | 20 | 15 | 400 | 3 | 80 | 10 |
| `facile` | 8 | 30 | 25 | 250 | 4 | 120 | 10 |
| `facile-strict` | 8 | 30 | 25 | 250 | 4 | 120 | 10 |

`default` also turns on `banInlineComments`, `banTODO`, `banInit` and `banGlobalMutable`;
`relaxed` turns all four off.

`facile` is the Facile suite standard, the `filet.yml` Sonde, Capsule and Sablier share: the
numbers above plus `params: 5`, `returns: 3`, `structFields: 12`, `interfaceMethods: 4`, every
style ban on, `extensions: [.go, .ts, .svelte, .sh]`, the suite's `ignore` list,
`requiredDirs: [apps/api, apps/client]`, `requiredFiles: {"apps/api/modules/*": [router.go]}`,
`disabled: [gen.line.long]` and `failOn: error`. A suite repository's config reduces to
`preset: facile` plus its own exceptions. A list or map you write replaces the preset's whole value
rather than merging into it, so an exception to `requiredFiles` restates the shared glob:

```yaml
preset: facile
architecture:
  requiredFiles:
    "apps/api/modules/*": [router.go]
    "apps/api/modules/docs": []   # static OpenAPI spec, no HTTP routes
```

The same holds for `disabled:`: a repo that disables one more rule lists `gen.line.long` again.

`facile-strict` is `facile` plus `style.strict: true`, which turns on the
[strict Go rules](#strict-go-rules). It is opt-in: no repository gets those rules without naming
this preset or setting the key.

`epitech` is the real norm's numbers, and they were written for C: no closures, no `if err != nil`,
no method receivers. On Go or TypeScript they fire constantly on code that is fine. Excellent as a
deliberate team discipline, wrong as a silent default — hence the opt-in.

### Adopting it on an existing project

Three things are worth setting before you read the first report, because they are noise on any real
codebase rather than findings:

- **Extend `ignore` with whatever your build copies around.** `_app`, `.output`, `dist` and friends
  are already there, but a Capacitor project keeps a full web build under `android/` and `ios/`, and
  a code generator that emits no banner (Prisma's zod output, for one) has to be listed by path.
  Files that do announce themselves — `code generated by`, `@generated`, `do not edit` — are skipped
  automatically.

  `ignore` entries are either bare directory names, skipped anywhere in the tree, or path patterns
  relative to the config: `data/training/**` drops a whole subtree (a dataset folder, say) while
  sibling directories stay checked, and `gen/*.gen.go` skips files matching a glob.
- **Disable `gen.line.long` if you run a formatter.** Prettier and gofmt already own line width;
  what survives them is what they could not break, so flagging it twice is noise.
- **Leave `architecture` empty until you mean it.** `maxDepth` is off by default: directory depth in
  a SvelteKit or monorepo tree is decided by the framework, not by your team.

`requiredFiles` states a per-directory contract: every directory matching the glob must contain the
files it lists. It is how a convention stops being folklore —

```yaml
requiredFiles:
  "apps/api/modules/*": [router.go]     # every module declares its routes
  "apps/api/modules/docs": []           # except this one, which serves no HTTP
  "apps/*": [package.json]              # every workspace member is a package
```

A directory covered by an exact pattern uses that one, so a convention carries its own exceptions
without a second config key. Globs are `path.Match`, so `*` does not cross a `/`. Findings are
errors: you asked for the contract, so breaking it breaks the build.

Paths in `architecture` always match relative to the directory holding `filet.yml`, whatever
directory filet was run from. Reported paths are relative to your working directory instead, so
they stay clickable.

`forbiddenImports` maps a path prefix to the imports that prefix must never pull in. It is the
cheapest way to keep a layered architecture honest: the domain layer stops importing HTTP, the
frontend stops importing the database driver. Both it and `forbiddenDirs` match a path *prefix*
relative to the config, not a bare directory name anywhere in the tree.

`maxFilesPerDir` warns when a directory holds more tracked files than the limit. It takes either
an integer for every directory, or a map from glob to limit where the most specific matching glob
wins and a limit of 0 opts a path out:

```yaml
maxFilesPerDir:
  .: 20                  # the root collects meta files; loose code does not belong there
  internal/engine: 0     # many small files by design, so no limit
```

Thirty is a sensible general cap — past that, a directory is usually a dumping ground rather
than a module. It is a warning, not an error: no established tool enforces a hard per-directory
count, so treat it as a prompt to group, not a broken build.

### Keys that cannot fire

Some keys drive rules that only run on some languages. `returns`, `structFields`,
`interfaceMethods`, `requireDocComments`, `banGlobalMutable`, `banInit`, `format` and `strict` are Go only;
`funcLines`, `funcStatements`, `params` and `complexity` need Go or a tree-sitter grammar (`.rs`);
`funcsPerFile` needs Go or a language filet can spot a function declaration in. When `filet.yml`
switches one of them on and no entry in `extensions:` can fire it, `filet check` reports
`gen.config.inert` as a warning on the config file, at the key's line. A config is a statement about
the code, and `banGlobalMutable: true` in a TypeScript-only repository states a rule nothing
enforces. Only keys written in the file count, not those a preset or the defaults set, and only a
run over the config root reports them.

## Checking code

Generic rules run on every configured extension. Go files additionally go through `go/ast`, so
function length, parameter count, cyclomatic complexity, naked returns, discarded return values,
oversized interfaces and package-level mutable state are measured rather than guessed.

When a language ships a vendored grammar (`.rs` today), filet parses it with tree-sitter, so its
shape rules — nesting, function length, params, statements, cognitive complexity — are measured on
real syntax instead of the brace counting used for the other non-Go languages. The tier is on by
default and deterministic and offline (grammars are compiled in); set `treesitter.enabled: false`
to fall back to the lighter brace-counting rules.

```sh
filet check .
filet roast internal/          # same findings, plus commentary
filet check . -fail warn       # exit 1 on warnings too
```

By default `check` and `roast` cache each tier's findings on disk, keyed by the content the tier
inspected plus the config and the tool version, and reuse them on a later run instead of re-parsing,
re-typechecking and re-spawning language servers. `filet check; filet check` on an unchanged tree is
near-instant; edit one file and only its own tier re-runs. The cache lives in
`$XDG_CACHE_HOME/filet` (else `~/.cache/filet`), is one file per project, and self-invalidates when
`filet.yml`, the config or the tool changes, so a stale entry is never served. Turn it off with
`-no-cache`, move it with `-cache-dir <dir>` or `cache.enabled: false` in `filet.yml`.

### Ratchet

A baseline lets a repository with standing findings gate on new ones without fixing the old ones
first, and makes sure the old ones only ever go down.

```sh
filet check -baseline write .     # record today's findings in .filet-baseline.json
filet check .                     # fail only on what the baseline does not cover
filet check -baseline update .    # lower the recorded counts after a cleanup
```

The file records, for each `(rule, path)`, how many findings at or above `failOn` it holds. Counts,
not line numbers, so an edit elsewhere in a file does not invalidate it. When the file exists at
the config root, or `filet.yml` names one with `baseline: <path>`, every `filet check`:

- shows a finding the baseline covers as `info` with the suffix `(baseline)`, and leaves it out of
  the exit code;
- fails as usual when a `(rule, path)` count grows past its baseline, or a new `(rule, path)`
  appears. All findings of a grown path keep their severity, since a count cannot say which one is
  new;
- fails with `gen.baseline.stale` when a count dropped, until `-baseline update` rewrites the file.
  `update` only lowers counts and drops entries; it never raises one or adds one. That is the
  ratchet: the file can only shrink. `write` records the current state from scratch.

A scoped run, `filet check apps/api`, compares and rewrites only the entries under that path. A
baseline that `filet.yml` names but that does not exist is an error, not a silent pass.

The known cost of counting per path: renaming a file shows as one stale entry for the old path plus
one new entry for the new one. Rename, then run `filet check -baseline write` in the same commit.

### Output formats

`-format` takes `auto` (the default), `text`, `line`, `json`, `sarif` or `github`.

**`auto`** picks `text` when stdout is a terminal and `line` when it is not, so a pipe or a
redirect gets something parseable without anyone passing a flag. Pass `-format text` explicitly
when you want the grouped report in a CI log.

**`text`** is grouped by file, aligned, and coloured when the terminal says it can take it —
`NO_COLOR` and `TERM=dumb` are honoured, and box-drawing characters degrade to ASCII unless the
locale announces UTF-8. It is meant to be read, never parsed.

**`line`** is one finding per line in the GNU error format every editor and CI annotator already
understands. Treat it as a stable contract:

```
path:line:column: severity: message [rule]
```

```console
$ filet check tronc | head -2
tronc/apiref/apiref.go:7:1: info: commented-out code [gen.commented.code]
tronc/apiref/apiref.go:82:3: info: every return value discarded into _ [go.err.discarded]

$ filet check tronc | cut -d: -f1 | sort | uniq -c | sort -rn | head -3
  15 tronc/migrate/migrate.go
  15 tronc/errors/errors.go
  11 tronc/spa/spa.go

$ filet check tronc | grep ': warn:' | wc -l
      39
```

Vim reads it with `set errorformat=%f:%l:%c:\ %t%*[a-z]:\ %m`, and Emacs `compile-mode` parses it
out of the box. `-quiet` suppresses the lines entirely, leaving only the exit code.

**`json`** carries the same findings plus `files` and `lines` totals, a `counts` breakdown by
severity and the `grade`, so a CI step reading it with `jq` does not reimplement the tally and the
grading curve in shell.

**`sarif`** is SARIF 2.1.0, which GitHub code scanning and most review tools ingest. Paths are
emitted relative to the working directory, because an upload matches results against repository
paths and shows nothing at all for a URI it cannot resolve — run filet from the repository root
in CI.

**`github`** writes GitHub Actions workflow commands, which the runner turns into annotations on
the file and line inside the pull request diff:

```console
$ filet check . -format github | head -1
::warning file=apps/api/main.go,line=12,col=1,title=go.doc.missing::exported Run has no doc comment
```

On a private repository this is the one to use: annotations cost nothing and work everywhere,
while a SARIF upload needs code scanning, which private repositories only get with GitHub
Advanced Security.

Exit codes: `0` clean, `1` findings at or above `failOn`, `2` bad usage or unreadable input.
`-fail never` always exits 0; an unknown value is rejected rather than silently treated as `error`.
`-quiet` drops the per-finding lines but keeps the summary, the grade and the exit code.

`testdata` is ignored by default, matching the Go toolchain: it holds fixtures that are often
invalid on purpose, and a parse failure there should not fail your build.

### Where the numbers come from

The defaults sit at or below the strict end of what the established tools ship, deliberately. The
`relaxed` preset is those published defaults, if you want them.

| Limit | default | `relaxed` | Prior art |
|---|---|---|---|
| `funcLines` | 30 | 60 | golangci-lint `funlen` 60, ESLint `max-lines-per-function` 50 |
| `funcStatements` | 25 | 40 | golangci-lint `funlen` 40 statements |
| `complexity` | 10 | 15 | SonarQube cognitive threshold 15; `gocognit` defaults to 30 and recommends 10–20 |
| `nesting` | 4 | 5 | ESLint `max-depth` and Sonar S134 allow 4 blocks *inside* a function; filet counts the function's own block too, so `relaxed`'s 5 is their budget and the default is one tighter |
| `params` | 5 | 5 | between ESLint `max-params` 3 and Sonar S107 7 |
| `fileLines` | 250 | 400 | ESLint `max-lines` 300 |
| `lineLength` | 120 | 120 | golangci-lint `lll` default |
| `funcsPerFile` | 8 | off | no prior art — an organisational preference, so `relaxed` turns it off |

A default that fires on healthy code teaches people to add `disabled:` and stop reading, so anything
noisy has to earn its place. `go.global.mutable` only survived on-by-default because the rule got
smarter: an unexported package-level `var` that is never written anywhere in its own file is a
lookup table, not shared state, and is skipped. Exported vars are always flagged — any package can
write those. Vars built through a `Must…`/`Once…` constructor are skipped too.

### Cognitive, not cyclomatic

`complexity` measures **cognitive** complexity, the SonarSource metric, not McCabe's cyclomatic
number. Two differences matter:

- **A `switch` costs one point, not one per `case`.** A twelve-case type switch is one dispatch, not
  twelve things to hold in your head. Counting per case is why cyclomatic tools scream at every
  dispatcher and end up muted.
- **Nesting is penalised.** An `if` at the top of a function costs 1; the same `if` three levels deep
  costs 4. Cyclomatic complexity scores them identically, which is the wrong answer — depth is what
  actually makes code hard to read.

Also counted: `else`/`else if` (+1 each, no nesting penalty), labelled `break`/`continue`/`goto`,
direct recursion, and each *run* of the same boolean operator — `a && b && c` is one point,
`a && b || c` is two.

`funcLines` excludes blank and comment-only lines, matching `funlen`'s `ignore-comments` default, so
documenting a function cannot push it over the limit.

`filet` passes its own defaults with zero findings and no preset — its `filet.yml` pins only the
project's directory layout. Getting there meant splitting 9 files into 22 and rewriting six
dispatchers, not moving a single threshold. `preset: relaxed` exists as an honest escape hatch for
projects that want the published tool defaults instead: name the regime you are in, do not silently
erode the default one threshold at a time.

### Function budget

Counting is exact on Go (`go/ast`, methods included, `_test.go` files exempt) and, with
`treesitter.enabled`, exact on every language with a vendored grammar (`.rs` today). The rest are
pattern-based on `.ts .tsx .js .jsx .svelte .rs .py .rb .java .sh` — arrow functions assigned to a
binding count, anonymous callbacks do not. `funcLines`, statement counts and cognitive complexity
are Go-only (or tree-sitter) measurements; the pattern-based languages get `fileLines` and nesting.

Package-level vars initialised through a `Must…` or `Once…` constructor (`regexp.MustCompile`,
`template.Must`, `sync.OnceValue`) are treated as immutable singletons and skipped.

## Roasting Dockerfiles

```sh
droast .
filet docker ./deploy/Dockerfile.api
```

It reads `FROM` pins, root users, apt hygiene, layer count, cache-busting `COPY . .`, shell-form
`CMD`, secrets baked into `ENV`/`ARG`, missing `.dockerignore`, missing `HEALTHCHECK`, and
single-stage builds on a toolchain base image.

## Running tests

`filet test` detects what the project actually uses and runs it, streaming output as it goes:

| Detected | Command |
|---|---|
| `go.mod` | `go test ./...` |
| `Cargo.toml` | `cargo test` |
| `bun.lock` / `bun.lockb` | `bun test` or `bun run test` |
| `pnpm-lock.yaml` | `pnpm test` |
| `yarn.lock` | `yarn test` |
| `package.json` | `npm test` |
| `deno.json` | `deno test -A` |
| `pyproject.toml`, `pytest.ini`, `tox.ini` | `pytest -q` |

It looks in every unit under the path, the same walk `filet gate` uses (see [The gate](#the-gate)),
and runs each suite from its unit's directory. Below the path a `package.json` counts only when it
has a `check` or `test` script. Every suite it finds runs, and the summary at the end reports
pass/fail and duration per suite, named with its unit (`go apps/api`).
Extra arguments are forwarded to the underlying runner:

```sh
filet test -- -run TestParse -v
filet test -format json
```

## Fuzzing

`filet fuzz` finds every fuzz target under a path and runs it, one target at a time:

- Go: every `func Fuzz*(f *testing.F)` in a `_test.go` file.
- Rust: every `fuzz/fuzz_targets/*.rs`, run through `cargo fuzz` from the crate root.

| Mode | Go | Rust |
|---|---|---|
| `-mode seeds` (default) | `go test -run '^FuzzX$' .` replays the seeds and `testdata/fuzz/` | `cargo fuzz run x -- -runs=0` |
| `-mode smoke` | `go test -run '^$' -fuzz '^FuzzX$' -fuzztime 30s .` | `cargo fuzz run x -- -max_total_time=30` |

Every run is bounded. `-fuzztime` (default `30s`) caps one target in smoke mode, and `-budget`
(default `5m`) stops starting new targets once the total is spent. A target whose tool is missing
(`go`, or `cargo-fuzz` for Rust) is reported as skipped with the reason, never as passed.

```sh
filet fuzz -list .
filet fuzz .
filet fuzz -mode smoke -fuzztime 10s -budget 2m .
filet fuzz -format json .
```

The command exits 1 when a target fails and prints the path of the failing input Go wrote under
`testdata/fuzz/`. Commit that file: from then on, seeds mode replays it as a regression test.

`test.fuzz.missing` is an opt-in hint. Set `style.hintFuzz: true` and filet reports, at `info`, each
function named `Parse*`, `Decode*` or `Unmarshal*` whose first parameter is `[]byte`, `string` or
`io.Reader`, in a package with no `Fuzz*` target.

## The gate

`filet gate` runs a repository's whole quality gate in one command. It first finds every unit
under the path, then runs each phase in order, every per-unit step from that unit's directory.

A unit is a directory holding a `go.mod`, a `Cargo.toml`, a `package.json` with a `check` or `test`
script, or another manifest `filet test` detects. The walk honours `ignore:` in `filet.yml` and
never enters `node_modules`, `vendor`, `dist`, `build`, `target`, `.git`, `testdata` or a hidden
directory. A crate under another crate is left to the outer one, as cargo does with workspace
members. A suite app with `apps/api/go.mod` and `apps/client/package.json` gives two units.

| Phase | Step | Runs per | Command |
|---|---|---|---|
| `format` | `gofmt` | Go module | `gofmt -l .` (fails when it lists a file outside a directory the walk skips, such as `vendor/`) |
| `format` | `cargo-fmt` | crate | `cargo fmt --check` |
| `check` | `check` | repository | `filet check`, in process |
| `analyze` | `vet` | Go module | `go vet ./...` |
| `analyze` | `staticcheck` | Go module | `staticcheck ./...` |
| `analyze` | `clippy` | crate | `cargo clippy -- -D warnings` |
| `analyze` | `bun-check` | `package.json` with a `check` script | `bun run check` |
| `test` | one per suite | unit | the `filet test` command, with `-race` for Go |
| `fuzz` | `go`, `rust` | unit with a fuzz target | `filet fuzz -mode seeds` |
| `douane` | `douane` | repository | `douane scan -fail critical -scope prod .` |

A row names its unit after the step, so a suite app reads `format:gofmt apps/api`,
`analyze:bun-check apps/client`, `test:go apps/api`. A step at the path itself carries no suffix.
Above the table the gate prints the units it covered:

```text
  units: apps/api (go), apps/client (bun)
```

A language with source files but no unit fails instead of vanishing from the table. `.go` files
with no `go.mod` fail `format:gofmt`, `analyze:vet` and `test:go` with
`12 .go files but no go.mod found`; `.rs` files with no `Cargo.toml` and `.ts`, `.tsx`, `.svelte` or
`.vue` files with no `package.json` carrying a `check` or `test` script fail the same way.

The gate stops at the first failed step and lists the rest as `not-run`; `-keep-going` runs every
step. A step whose tool is not on `PATH` is `skipped: <tool> not found`, never passed. The command
exits 0 only when no step failed, and prints one table of step, status and duration.
`-format json` prints the same table as one JSON document
(`{"passed": bool, "units": [{"path", "kinds"}], "steps": [...]}`, every step carrying its `unit`)
on stdout and moves the step output to stderr.

```yaml
gate:
  disable: [douane]          # a phase or a step name
  requireTools: [staticcheck] # a missing tool fails the gate instead of skipping
```

`disable` takes a phase (`format`, `check`, `analyze`, `test`, `fuzz`, `douane`) or a step name, and
applies to that step in every unit. An unknown name in either list is an error that names the
valid ones, so a typo cannot quietly loosen the gate.

## In CI

The repository ships a composite action. Annotations are unconditional; SARIF and the Antenne
report are opt-in.

```yaml
- uses: FacileStudio/filet@v0.18.1
  with:
    version: v0.18.1
    path: .
    fail-on: error
```

Set `command: gate` to run the whole gate instead of `filet check`. The gate needs every tool it
should run on the runner's `PATH`; the ones it does not find are reported as skipped.

It installs filet with `go install`, so the runner needs a Go toolchain — `ubuntu-latest` ships
one, and any job that already uses `actions/setup-go` is covered. It writes the annotations, puts
a graded summary table on the job page, and exposes `grade`, `errors`, `warnings` and `report`
(the path to the JSON) as step outputs.

Public repositories can add code scanning on top:

```yaml
permissions:
  contents: read
  security-events: write

steps:
  - uses: FacileStudio/filet@v0.18.1
    with:
      sarif: 'true'
```

### Reporting runs to Antenne

`antenne-url` posts the run to an Antenne `filet`
webhook provider, signed with HMAC-SHA256 over the body in `x-filet-signature-256`. filet itself
never opens a socket — the action does the posting, so the linter stays runnable offline.

```yaml
- uses: FacileStudio/filet@v0.18.1
  with:
    antenne-url: https://antenne.facile.studio/webhook/filet
    antenne-secret: ${{ secrets.ANTENNE_FILET_SECRET }}
```

Antenne delivers a failure on the default branch and stays quiet otherwise — a green run says
nothing and a pull request failure is already on the author's diff. A quiet run answers 200 and
leaves an application log line, but no event, so do not expect green runs under Antenne's events
API. **Configure the provider's
secret when you create it:** an Antenne webhook provider with an empty secret accepts
unauthenticated calls on that path, which means anyone can post a fake build failure, or a fake
success.

## Rules

`filet rules` prints the full list. Disable any of them by id in `filet.yml`.

`gen.license.required` is opt-in, and no preset turns it on, `facile` included. Set
`style.requireLicense: true` and it warns when the config root holds no `LICENSE`, `LICENSE.md`,
`LICENSE.txt` or `COPYING`. When a `package.json` or `Cargo.toml` sits there too, it also warns
when that manifest's `license` field is unset, or names another licence than the LICENSE text.
filet recognises two licences by their text: `AGPL-3.0-only` and `MIT`. Any other text only needs
the field to be set. The rule runs only when `filet check` covers the whole config root. It reports
at `warn`, so a config with `failOn: info` or `warn` fails on it once enabled.

### Strict Go rules

`style.strict: true`, or `preset: facile-strict`, turns on the `go.strict.*` family. Each rule
reports at `error`, because each finding is a defect rather than a matter of taste, and none of
them runs on a `_test.go` file. They read the package's type information, so a package whose
imports filet cannot resolve gets the `go.leak.resource` skip notice instead of silence.

| Rule | Reports |
|---|---|
| `go.strict.switch` | a `switch` over an enum-like type (a defined integer or string type with two or more constants in its package) that has no `default` and misses some constants. Constants of another package count only when exported |
| `go.strict.close` | a call whose `*http.Response`, `*sql.Rows` or `*os.File` result is discarded (`db.Query(q)` as a statement, `_, err := os.Create(p)`), and an `*http.Response` whose `Body` is never closed, returned, stored, wrapped in something with a `Close`, or passed to a deferred call or a function named for closing or draining it. A kept `*sql.Rows` or `*os.File` that is never closed stays `go.leak.resource`'s finding |
| `go.strict.import` | `unsafe`, `"C"` (cgo) or `reflect` |
| `go.strict.timeout` | an `http.Client` literal, `new(http.Client)` or zero-value `var` with no `Timeout`, and every use of `http.DefaultClient`, through `http.Get`, `Head`, `Post` and `PostForm` included. `Timeout: 0` written out is how a streaming client says it means no limit, and a `c.Timeout = d` elsewhere in the file counts |
| `go.strict.sql` | `Query`, `QueryContext`, `QueryRow`, `QueryRowContext`, `Queryx`, `QueryRowx`, `Exec`, `ExecContext`, `MustExec`, `Prepare`, `PrepareContext` or `Raw` called with statement text built by `+`, `+=` or `fmt.Sprintf` from a string that is not a constant, a number, a `strconv` formatting or a driver-quoted identifier (a function named `*quote*` or `*sanitize*` outside `strconv`). Text that only passes through, such as a wrapper's parameter, is not reported |
| `go.strict.goroutine` | a `go` statement inside a function taking an `http.ResponseWriter` and an `*http.Request` whose call and closure reach no `context.Context`, channel or `sync.WaitGroup`. A bare `context.Background()` or `context.TODO()` argument does not count |
| `go.strict.assert` | a single-value type assertion `x.(T)`, which panics on any other type. The `v, ok` form and type switches are fine |
| `go.strict.nilflow` | a pointer or interface result read before the error returned beside it is checked: `resp, err := c.Do(req)` then `defer resp.Body.Close()`. Only reads that cannot survive nil count (a field through a pointer, a method of an interface, `*p`); a pointer method call does not, since the method may accept a nil receiver. The scan stops at the first statement that mentions the error, compares the value with `nil` or assigns it again |

Each rule is narrow on purpose, so that a finding is worth fixing. `go.strict.nilflow` follows
only the statements after the call in the same block, and `go.strict.close` follows a response
only inside the function that acquired it: a response passed to another function counts as
handed off. Disable a rule by id when a repository has a reason, and use the ratchet baseline to
adopt the family on a codebase that already has findings.

## Stability

Two things are a contract, and a test fails if either moves:

- **Rule ids.** Configs name them in `disabled:`, and the `line` format prints them in brackets.
  Renaming or removing one is a breaking change, so `frozenRules` has to be edited first,
  deliberately.
- **The `line` format**, `path:line:column: severity: message [rule]`, pinned by a regex.

Everything else may move between minor versions: thresholds, severities, which preset ships what,
and how the `text` report is laid out. Findings are a judgement, and a judgement that can never be
revised is not worth much: the first set of defaults was wrong in ways only a real codebase showed.

Pin a tag if you gate CI on the count:

```sh
go install github.com/FacileStudio/filet@v0.18.1
```

## License

filet is licensed under the GNU General Public License v3.0 only (`GPL-3.0-only`); see [LICENSE](LICENSE).
