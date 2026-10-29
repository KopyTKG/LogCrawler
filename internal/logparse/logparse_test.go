package logparse

import (
	"strings"
	"testing"
)

// A synthetic log in the app's format. Nothing here comes from a real device.
const sample = `DRIVE! Example
Variant: exampletest
App version: 1.2.3
Build: 9

=== previous session ===
10:00:00.100 LOG   --- log started 2026-01-01T10:00:00.100Z — Example 1.2.3 (exampletest) ---
10:00:00.200 LOG   App mounted.
10:00:01.000 LOG   [http] #1 → POST https://api.example/sd/v1/mobile/App/Init
10:00:01.186 LOG   [http] #1 ← 200 POST https://api.example/sd/v1/mobile/App/Init 186ms
10:00:02.000 LOG   [http] #2 → GET https://api.example/sd/v1/helpdesk/Thing/7
10:00:02.500 LOG   [http] #2 ✕ 500 GET https://api.example/sd/v1/helpdesk/Thing/7 500ms — Request failed with status code 500
10:00:02.500 LOG   [http] #2 ✕ body {"status":500,"requestId":"abc"}
10:00:02.510 ERROR {"message":"Request failed with status code 500","name":"AxiosError"}
10:00:03.000 LOG   [http] #3 → GET https://api.example/never/answered
=== current session ===
10:05:00.000 LOG   --- log started 2026-01-01T10:05:00.000Z — Example 1.2.3 (exampletest) ---
10:05:01.000 LOG   [nav] (start) → TodoIssues
10:05:02.000 LOG   [detail] parameters for 42:
  [1197] Dates (2)
    hidden A.B mode=Visible value=—
10:05:03.000 LOG   [somethingnew] a tag we have never seen
10:05:04.000 LOG   WARN something mirrored from console.warn
`

func parse(t *testing.T, s string) *File {
	t.Helper()
	f, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestHeader(t *testing.T) {
	f := parse(t, sample)
	if f.Header.Title != "DRIVE! Example" {
		t.Errorf("title = %q", f.Header.Title)
	}
	if len(f.Header.Fields) != 3 || f.Header.Fields[0].Value != "exampletest" {
		t.Errorf("fields = %+v", f.Header.Fields)
	}
}

func TestSessions(t *testing.T) {
	f := parse(t, sample)
	if len(f.Sessions) != 2 {
		t.Fatalf("sessions = %d", len(f.Sessions))
	}
	if f.Sessions[0].Label != "previous session" || f.Sessions[1].Label != "current session" {
		t.Errorf("labels = %q, %q", f.Sessions[0].Label, f.Sessions[1].Label)
	}
	if f.Sessions[1].StartedAt != "2026-01-01T10:05:00.000Z" {
		t.Errorf("startedAt = %q", f.Sessions[1].StartedAt)
	}
	if len(f.Problems) != 0 {
		t.Errorf("problems = %+v", f.Problems)
	}
}

func TestTagsAndUntagged(t *testing.T) {
	cur := parse(t, sample).Sessions[1]
	byTag := map[string]int{}
	for _, e := range cur.Events {
		byTag[e.Tag]++
	}
	if byTag["nav"] != 1 || byTag["somethingnew"] != 1 || byTag[""] != 2 {
		t.Errorf("tags = %v", byTag)
	}
}

func TestContinuationLinesAttach(t *testing.T) {
	cur := parse(t, sample).Sessions[1]
	var detail *Event
	for _, e := range cur.Events {
		if e.Tag == "detail" {
			detail = e
		}
	}
	if detail == nil || len(detail.Continuation) != 2 {
		t.Fatalf("detail = %+v", detail)
	}
	if !strings.Contains(detail.Continuation[1], "hidden A.B") {
		t.Errorf("continuation = %q", detail.Continuation)
	}
}

func TestHTTPPairing(t *testing.T) {
	prev := parse(t, sample).Sessions[0]
	if len(prev.Exchanges) != 3 {
		t.Fatalf("exchanges = %d", len(prev.Exchanges))
	}
	ok, failed, pending := prev.Exchanges[0], prev.Exchanges[1], prev.Exchanges[2]
	if ok.Failed || ok.Status != 200 || ok.Request == nil || ok.Outcome == nil {
		t.Errorf("ok = %+v", ok)
	}
	if !failed.Failed || failed.Status != 500 || !strings.Contains(failed.Body, "abc") {
		t.Errorf("failed = %+v", failed)
	}
	if pending.Request == nil || pending.Outcome != nil {
		t.Errorf("pending = %+v", pending)
	}
}

func TestIDsRestartPerSession(t *testing.T) {
	s := "=== previous session ===\n00:00:01.000 LOG   [http] #1 → GET http://a/x\n" +
		"=== current session ===\n00:00:02.000 LOG   [http] #1 → GET http://a/y\n"
	f := parse(t, s)
	if f.Sessions[0].Exchanges[0].URL != "http://a/x" || f.Sessions[1].Exchanges[0].URL != "http://a/y" {
		t.Errorf("ids were paired across sessions")
	}
}

func TestNoMarkerFile(t *testing.T) {
	f := parse(t, "00:00:01.000 LOG   hello\n")
	if len(f.Sessions) != 1 || f.Sessions[0].Label != "" || len(f.Sessions[0].Events) != 1 {
		t.Errorf("sessions = %+v", f.Sessions)
	}
}

func TestMalformedInputDoesNotFail(t *testing.T) {
	for _, in := range []string{"", "\n\n", "garbage\nmore garbage", "=== current session ===\nnot an event\n", "00:00:01.000 LOG   [http] #x → ???"} {
		if _, err := Parse(strings.NewReader(in)); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
	f := parse(t, "=== current session ===\nnot an event\n")
	if len(f.Problems) != 1 || f.Problems[0].Line != 2 {
		t.Errorf("problems = %+v", f.Problems)
	}
}

func TestLastLineWithoutNewline(t *testing.T) {
	f := parse(t, "00:00:01.000 LOG   first\n00:00:02.000 LOG   last")
	if len(f.Sessions[0].Events) != 2 {
		t.Errorf("events = %d", len(f.Sessions[0].Events))
	}
}
