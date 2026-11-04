# LogCrawler

LogCrawler is a desktop app that reads the diagnostics log written by the ServiceDesk Connect mobile
app and shows it as a tree. A colleague can open a log a tester sent and see what the user was doing
when something failed, without reading hundreds of lines of interleaved HTTP, navigation and
workflow output.

Everything happens on your machine. LogCrawler never uploads a log and makes no network calls.

Built with Go and [Wails v3](https://v3.wails.io/) (Go backend, system web view) and a Svelte 5 +
TypeScript frontend. Version 0.1.0, BSD 3-Clause licensed (see [LICENSE](LICENSE)).

> Status: first versions. Targets are Windows and Linux. It builds and runs on Linux; the Windows
> build cross-compiles but has not been run on Windows yet, and the app is not signed.

## Using it

**Open a log** with *Open…* (Ctrl+O), by dropping the file on the window, from the recent list on the
start screen, or on the command line: `logcrawler path/to/file.log`. Only logs from app 2.6.2 and
newer are supported; 2.7.0 logs also include the headers and bodies of every HTTP call.

**Two views** of the same log, switched at the top:

- *Screens*: session, then screen, then the HTTP calls, warnings and errors that happened while the
  user was on that screen. Use this to answer "what was the user doing when this failed".
- *Requests*: session, then HTTP calls. Use this to answer "which calls went wrong".

**Finding the problem.** The bar under the toolbar counts failed calls, errors, unanswered calls and
slow calls, and steps through the failures (F8 / Shift+F8). The thin strip beside the detail pane
marks where the failures are in the file; click a mark to jump there. Search matches text, and can
be narrowed by tag, to problems only, or to slow calls. F3 / Shift+F3 (or Enter in the search box)
step through the hits.

**The detail pane** shows the log text of the selected row. It rolls out when you select a row and
folds away when nothing is selected, or with the `›` button; the *Detail* tab on the edge brings it
back, and the splitter resizes it.

- *Parsed* shows JSON bodies, headers and similar data pretty-printed and coloured, and offered
  transitions as a table.
- *Raw* shows the log text exactly as written.
- *Show N events before* adds the events that happened just before the selection.
- *Copy* copies the raw text. *Copy bug report* copies a redacted summary (app version, the selected
  event, the events before it) that is safe to paste into a ticket.

**Also in the toolbar:** *Expand all* / *Collapse*, *Export…* (the matches, or the failures when
nothing is searched), *Compare…* (two logs side by side: counts, failing endpoints, screens visited
in only one), *Reload*, and *Info* (the log header, the sessions, and lines the parser could not
place). If the file changes on disk, a bar offers to reload it.

**Settings** (footer): UI size from 80 % to 200 %, also with Ctrl + / Ctrl − / Ctrl 0. The footer
also switches the language (Czech or English) and the theme (auto, light or dark). All of these are
remembered.

| Key | Action |
| --- | --- |
| Ctrl+O | Open a log |
| Ctrl+F | Focus the search box |
| Ctrl+R | Reload the file |
| F8 / Shift+F8 | Next / previous failure |
| F3 / Shift+F3 | Next / previous search hit |
| Ctrl + / Ctrl − / Ctrl 0 | Larger / smaller / reset UI size |

The first run of the unsigned Windows build shows a SmartScreen warning: click *More info*, then
*Run anyway*.

## Building

Tool versions are in `mise.toml` (Go, Node, yarn 1, just); `mise install` sets them up. Tasks are
in the `justfile`; `just --list` shows them all.

```sh
just setup     # install the wails3 CLI (v3.0.0) and the frontend dependencies
just build     # generate bindings, build the frontend, build bin/logcrawler
just build-windows  # cross-compile bin/logcrawler.exe
just run       # build and start; `just run path/to/file.log` opens a log
just dev       # live development: Vite hot reload, Go rebuilds on change
```

The title bar and footer show the git tag, or the short commit when there is no tag. Tag a release
with `git tag v0.1.0` before building to get a clean version number.

**Linux** needs the GTK 3 and WebKitGTK 4.1 development packages (`webkit2gtk-4.1`); the justfile
adds `-tags gtk3` for you. **Windows** needs WebView2, which Windows 10 and 11 already have, so
users install nothing. The Windows build needs no cgo, so `just build-windows` makes
`bin/logcrawler.exe` from Linux too.

`.jenkins/Jenkinsfile` runs the tests and builds both binaries on a Debian agent; for a tag it
attaches them to a GitHub release.

## Development

```sh
just test      # Go tests, Svelte type check, frontend unit tests
just lint      # eslint, prettier check, gofmt
just fmt       # format the frontend and the Go code
```

| Path | What is there |
| --- | --- |
| `internal/logparse/` | The parser: a file to typed events, with HTTP calls paired up. No Wails import. |
| `internal/logtree/` | Tree views, search, failures, bug report, export, compare. No Wails import. |
| `internal/redact/`, `cmd/redact/` | Placeholders for emails, hosts, ids and names; the tool that makes fixtures. |
| `main.go`, `logservice.go`, `settings.go` | The Wails app, the service the frontend calls, remembered window state. |
| `frontend/` | Svelte 5 UI. `bindings/` is generated by `just bindings` and not committed. |
| `docs/log-format.md` | The log format, written from real logs. |

The log format belongs to the mobile app. If the app's format changes, update the parser and
`docs/log-format.md` in the same commit. Translations are in `frontend/src/lib/locales/`; a new
string needs a key in both files.

### Test data and privacy

Real logs contain user emails, account ids, issue numbers and, in older files, live Bearer tokens.

- Put real logs in `testdata/raw/`. It is gitignored: never commit anything from it or paste it
  into an issue, a commit message or a fixture.
- `testdata/fixtures/` is the only committed log data. Generate it with `just fixtures`, which runs
  the raw logs through the redaction tool, then read the result before committing. If a redaction
  rule is missing, add it to the tool and regenerate; do not hand-edit a fixture.

More background, decisions and the project's conventions are in [CLAUDE.md](CLAUDE.md).
