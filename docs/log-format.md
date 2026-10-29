# Log format

What the ServiceDesk Connect mobile app writes to its on-device diagnostics log, as of app 2.6.2
(build 34), and what `internal/logparse` makes of it. The app owns the format; when its logging code
changes, this file and the fixtures change with it.

The format belongs to the mobile app, not to this project: the authority is its
`src/utils/{log,logFile,diagnosticsLog,httpLog}.ts` and `src/engine/drive/**` (the `[transition]`
lines). When they change, this project's spec and fixtures change with them. Only the current format
is supported — apps from before the on-device log existed have no log files, so there is nothing
older to read.

A file is a **header**, then one or two **sessions**:

- **Header** — `Key: value` lines (variant, app version, build, Expo SDK, platform, the three API
  URLs, the web panel), ending at a blank line. **The keys are in the app's UI language**: Czech on
  sdtest (`Varianta`, `Verze aplikace`), English on grosvenorprod (`Variant`, `App version`). Do not
  match on the key text; match on position, or on the value shape (`http…`, `android 36`).
- **Session markers** — `=== previous session ===` and `=== current session ===`. The app rotates the
  log on every start, so a file is the session before plus the current one. A file may hold only the
  current marker. Each session opens with `--- log started <ISO time> — <app> <version> (<variant>) ---`.
- **Events** — `HH:MM:SS.mmm LEVEL  message`. Levels seen: `LOG`, `ERROR`. Time has no date and
  wraps at midnight; the date is only in the `log started` line.
- **Tags** — a message usually starts with `[tag]`. Seen: `[http]`, `[nav]`, `[transition]`,
  `[detail]`, `[app]`, `[auth]`, `[sync]`. Untagged messages exist (`App mounted.`, `WARN …` for a
  mirrored `console.warn`). Treat the tag set as open: an unknown tag is a leaf, never an error.
- **`[http]`** — three shapes, paired by `#<id>`:
  `#<id> → <METHOD> <url>` is a request; `#<id> ← <status> <METHOD> <url> <n>ms` is a successful
  response; `#<id> ✕ <status> <METHOD> <url> <n>ms — <message>` is a failed one, directly followed
  by `#<id> ✕ body <text>` (JSON with `requestId`, or `{}` for an empty body). Ids restart each
  session. A request can have no response at all (app killed, timeout).
- **`[nav]`** — `<previous> → <current>` with masked route params; `(start)` is the first screen.
- **`ERROR`** — the message is a JSON object (`{"message": …, "name": "AxiosError", "stack": …}`),
  with no tag. It follows a failed `[http]` call (`✕`) and repeats its message.
- **Continuation lines** — a line with no leading timestamp belongs to the previous event. Seen in
  multi-line dumps: `[detail] parameters for <issue id>:` followed by indented `[1197] <category> (n)`
  and `hidden <code> mode=… value=…` lines (about a third of the lines in `sdtest-…-3.log`). The
  `=== … session ===` markers also have no timestamp but are not continuations. Never drop a
  continuation line; attach it.

Unknown or malformed lines are kept as plain leaves with their original line number. The parser
must never throw on input it does not understand: a crawler that stops at the first odd line is
useless on exactly the logs people send because something went wrong.

## How the parser models it

`logparse.Parse` returns a `File`: a `Header` (title plus `Key: value` fields), `Sessions` and a list of
`Problems`. A `Session` holds its `Events` (timestamp, level, tag, message, continuation lines, and for
`[http]` lines a decoded `HTTPLine`) and its `Exchanges`: each HTTP call with its request, its outcome
(success or failure) and the failure body. An exchange with no `Outcome` never completed. A `Problem`
is a line that belongs to no event; it should stay empty on real logs, and `TestRawLogs` checks that.
