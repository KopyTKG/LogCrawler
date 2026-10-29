# Test data

- `raw/` — real logs pulled from devices. **Gitignored, never commit.** They contain user emails,
  account ids, issue numbers and, in older files, a live Bearer token. Use them for local runs and for
  exploring new formats.
- `fixtures/` — what the tests read, and the only log data that may be committed. Each file is a raw
  log passed through the redaction tool (emails, ids, tokens, hostnames replaced with stable
  placeholders) and then read through by a human before the commit. Keep them small: one file per
  behaviour (a clean session, a failed request, a rotated session, a continuation-line dump).

## Raw files here

All four are `2.6.2` build 34 from 2026-10-27, each holding a `=== previous session ===` and a
`=== current session ===` block. `sdtest-…-2` and `-3` repeat the same previous session, so
parse them as overlapping, not as separate runs.

| file | variant | notes |
|---|---|---|
| `grosvenorprod-2026-10-27.log` | grosvenorprod | English header; smallest |
| `sdtest-2026-10-27-1.log` | sdtest | Czech header |
| `sdtest-2026-10-27-2.log` | sdtest | Czech header; shares its previous session with `-3` |
| `sdtest-2026-10-27-3.log` | sdtest | Czech header; has `ERROR` JSON lines (HTTP 400/500) |

The header of a file is in the app's UI language (Czech on sdtest, English on grosvenorprod) — see
`CLAUDE.md`.

## Fixtures

`fixtures/real-1.log` … `real-4.log` are the four raw files above, in the same order, passed through
`just fixtures` (`cmd/redact`). Names are by number because the raw names carry the customer. The
tool replaces emails, hosts, UUIDs, issue numbers, long numeric ids, free-text JSON values (names,
titles, content) and the variant/brand name wherever it appears. It is a machine pass: `go test
./internal/logtree` fails if an address is left, but read a fixture before committing a new one.
If a rule is missing, add it to `internal/redact` and regenerate; do not hand-edit a fixture.
