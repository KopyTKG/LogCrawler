# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

LogCrawler reads the on-device diagnostics log written by the ServiceDesk Connect mobile app
(`/work/servicedesk-connect-app/SD-mobile-client`) and shows it as a tree, so a colleague can open a
log a tester sent and see what the user was doing when something failed — without reading 400
lines of interleaved HTTP, navigation and workflow output.

It is a desktop GUI app for colleagues, built as **Go + Wails v3** (Go backend, web frontend, one
binary per OS). The UI toolkit is an experiment, not a commitment: the parser is plain Go and does
not import Wails, so the UI can change without touching it.

**Status: Phase 1 and 2 first versions exist.** The format spec (`docs/log-format.md`), the parser
(`internal/logparse`), the redaction tool (`cmd/redact`, `internal/redact`) and committed fixtures
(`testdata/fixtures/real-*.log`, generated from the raw logs by `just fixtures`) are in. The Wails
app opens a log (dialog, drag-drop, command line, recent files), shows it as a screens or requests
tree with colours, search and filters, a failure bar with jump and a timeline strip, a detail pane
with pretty-printed JSON, the events before a failure, a redacted bug-report copy, export, compare
and reload on change. Not done: Windows and macOS builds have never been run (the `Jenkinsfile` is
untested), no signing, default Wails app icon. The fixtures are machine-redacted; read them before
committing, as the privacy section says.

## Platform and toolkit decisions

- **Targets: Windows and macOS** — what the colleagues who will use it run. Linux is the developer's
  machine here; it builds and runs, but nobody is promised a Linux binary.
- **Wails v3, pinned to `v3.0.0`** (`github.com/wailsapp/wails/v3`, tagged 2026-10-26 — the first
  stable v3; it follows beta.34 and a `v3.1.0-nightly` already exists). Do not follow nightlies.
  It is days old: expect thinner docs than v2 and rough edges, and check an API against the v3 docs
  and the module source rather than from memory of v2 — v2 patterns (the `wails` CLI, `wails.json`,
  `runtime` package) do not carry over; v3 uses the `wails3` CLI and Taskfiles.
- **Frontend: Svelte 5 + TypeScript + Vite**, a plain single-page app (no SvelteKit, no router — one
  window, one file, one tree). Latest at the time of writing: svelte 5.57, vite 8.3, vitest 5; pin
  what the Wails v3 Svelte template ships and bump deliberately. Written with **runes**
  (`$state`, `$derived`, `$effect`, `$props`), not the Svelte 4 `export let` / `$:` style — most
  tutorials and Stack Overflow answers are still Svelte 4, so check the Svelte 5 docs. React and
  Preact were considered; Svelte was picked for its size and because the developer wants to learn it.
  - **The tree is ours to build.** The React options (`react-arborist`) do not apply. Flatten the
    expanded nodes into one list and render only the visible window of it (virtual list). A log is
    tens of thousands of lines; a plain nested `{#each}` locks the UI. Start with a small hand-written
    windowing component and only reach for a library if it earns its place.
  - **State** lives in Svelte stores or runes modules; no state library.
  - **Search and parsing stay in Go.** The frontend asks Go for children lazily, so only expanded
    nodes cross the Wails bridge.
  - **Styling:** plain CSS with variables, light and dark; no UI kit and no Tailwind.
  - **Tooling:** `yarn` (like the mobile repo), Vitest + Testing Library for the frontend, ESLint
    with the Svelte plugin, Prettier with `prettier-plugin-svelte`.
- **Fyne was considered and rejected**: its tree widget is basic and slows on thousands of nodes, text
  inside it is hard to select and copy, and it needs cgo/OpenGL, which makes shipping to two OSes
  harder. This tool is a large searchable tree of text people copy from; a web view does that well.
- **Build and shipping notes (provisional until the first real build):**
  - macOS can only be built on a Mac. Windows can be built there or here; verify v3's
    cross-compile path before relying on it.
  - Windows needs WebView2, present on Windows 10 and 11, so colleagues install nothing. An unsigned
    `.exe` shows a SmartScreen warning on first run; clicking through is fine.
  - An unsigned macOS app gets a Gatekeeper warning (right-click → Open once). Signing and
    notarising needs the team's Apple developer account.
  - Build each OS on its own machine and attach both binaries to a tagged release.

## Plan

1. **Phase 1 — format spec and parser, no UI.** Write `docs/log-format.md` from the real logs, then
   a Go package that turns a file into a flat, typed list of events with the pairings resolved.
   Tests run against `testdata/fixtures/`. Also a small redaction tool that produces those fixtures
   from `testdata/raw/`.
2. **Phase 2 — the Wails app.** Open a file (or drop one on the window), show a tree, search and
   filter, copy a node as text for a bug report.

The tree shape is deliberately undecided. The parser produces events; a tree is a view over them.
Try "session → screen → events" (screens from `[nav]`, with the `[http]`, `[transition]`, warnings
and errors that happened while on that screen as children) and "session → request chains" against
the real logs and keep what actually answers "what was the user doing when this failed". Both views
can share one parser.

## The log format

Full description in [`docs/log-format.md`](docs/log-format.md); the parser is
`internal/logparse`. The format belongs to the mobile app (`src/utils/{log,logFile,diagnosticsLog,httpLog}.ts`
and `src/engine/drive/**`), only the current one (app 2.6.2+) is supported, and the points that
trip a parser are:

- the header's keys are in the app's UI language — never match on them;
- a failed request is `#id ✕ …` followed by `#id ✕ body …`, not a `←` line;
- lines with no timestamp are continuations of the previous event (a third of one real log);
- unknown tags and malformed lines must become leaves or `Problems`, never an error.

## Privacy — read before touching test data

Real logs contain user emails, account ids, issue numbers and, in older files, a live Bearer token.

- `testdata/raw/` holds the real logs and is **gitignored**. Never commit anything from it, never
  paste its contents into an issue, a commit message or a fixture.
- `testdata/fixtures/` is the only committed log data. A fixture is a raw file passed through the
  redaction tool and then read by a human. If a redaction rule is missing, add it to the tool and
  regenerate; do not hand-edit one fixture.
- The app masks secrets on its own side (`authorization|password|secret|token|cookie|api-key` keys
  and `Bearer …` strings), but that is a safety net, not a guarantee. Redact again here.
- The app itself never uploads logs; keep it that way. LogCrawler is a local reader: no network
  calls, no telemetry, no crash reporting that includes file contents.

## Conventions

- Go standard library first. The parser package has no third-party dependencies.
- Table-driven tests, fixtures in `testdata/fixtures/`, one fixture per behaviour (clean session,
  failed request, rotated log, continuation lines, unknown tag, truncated final line).
- The parser reports events and problems as data (`Event`, plus a list of `Problem`), not as errors
  that abort parsing.
- Keep `docs/log-format.md` true in the same commit as a parser change — a spec that describes last
  month's format is worse than none.

## Layout

```
CLAUDE.md                 this file
go.mod                    module `logcrawler`
internal/logparse/        the parser (no Wails import) and its tests
testdata/raw/             real logs, gitignored
testdata/fixtures/        redacted logs the tests read (real-N.log, from `just fixtures`)
docs/log-format.md        the format spec
internal/logtree/         tree views over parsed events, search, failures, report, export, compare (no Wails import)
internal/redact/          placeholders for emails, hosts, ids, names; used by the fixture tool and the bug report
cmd/redact/               turns testdata/raw into testdata/fixtures
main.go, settings.go      window options, remembered size, recent files (config dir, paths only)
justfile, mise.toml       tasks (`just --list`) and tool versions
Jenkinsfile               CI: test on linux, build on linux/windows/macos agents
main.go, logservice.go    Wails app and the service the frontend calls
frontend/                 Svelte 5 + TS + Vite UI; bindings/ is generated (gitignored)
```

Linux dev build: `just build` (tasks in the `justfile`; it adds `-tags gtk3` on Linux because webkitgtk-6.0 is not
installed here), `just run [file]` to start it, `just test` and `just lint` before a commit.

The parser lives in its own package with no Wails import, and the module's
`go.mod` pins `github.com/wailsapp/wails/v3 v3.0.0`.
