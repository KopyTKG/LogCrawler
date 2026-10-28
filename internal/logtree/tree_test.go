package logtree

import (
	"strings"
	"testing"

	"logcrawler/internal/logparse"
)

// A synthetic log in the app's format. Nothing here comes from a real device.
const sample = `DRIVE! Example
Variant: exampletest

=== previous session ===
10:00:00.100 LOG   --- log started 2026-01-01T10:00:00.100Z — Example 1.2.3 (exampletest) ---
10:00:00.200 LOG   App mounted.
10:00:01.000 LOG   [nav] (start) → Login
10:00:02.000 LOG   [http] #1 → GET https://api.example/ok
10:00:02.186 LOG   [http] #1 ← 200 GET https://api.example/ok 186ms
10:00:03.000 LOG   [nav] Login → TodoIssues
10:00:04.000 LOG   [http] #2 → GET https://api.example/thing/7
10:00:04.500 LOG   [http] #2 ✕ 500 GET https://api.example/thing/7 500ms — Request failed
10:00:04.500 LOG   [http] #2 ✕ body {"requestId":"abc"}
10:00:04.510 ERROR {"message":"Request failed"}
10:00:05.000 LOG   [detail] parameters for 42:
  [1197] Dates (2)
10:00:06.000 LOG   [http] #3 → GET https://api.example/never
=== current session ===
10:05:00.000 LOG   [somethingnew] hello
`

func load(t *testing.T) *Doc {
	t.Helper()
	f, err := logparse.Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	return NewDoc("x.log", f)
}

func labels(infos []NodeInfo) []string {
	var out []string
	for _, i := range infos {
		out = append(out, i.Label)
	}
	return out
}

func TestScreensTree(t *testing.T) {
	d := load(t)
	tr := d.Tree(ViewScreens)
	roots := tr.Children(-1)
	if len(roots) != 2 {
		t.Fatalf("roots = %v", labels(roots))
	}
	if roots[0].Errors != 2 || roots[0].Warnings != 1 {
		t.Errorf("session counts = %d errors, %d warnings; want 2 (call + its ERROR line), 1", roots[0].Errors, roots[0].Warnings)
	}
	screens := tr.Children(roots[0].ID)
	want := []string{"(before the first screen)", "Login", "TodoIssues"}
	if got := labels(screens); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("screens = %v, want %v", got, want)
	}
	todo := tr.Children(screens[2].ID)
	// failed exchange, its ERROR event, the detail dump, the unfinished exchange
	if len(todo) != 4 || todo[0].Kind != KindExchange || todo[0].Sev != SevError {
		t.Fatalf("TodoIssues children = %+v", todo)
	}
	if lines := tr.Children(todo[0].ID); len(lines) != 3 {
		t.Errorf("failed exchange has %d lines, want 3", len(lines))
	}
	if todo[3].Sev != SevWarn || !strings.Contains(todo[3].Label, "no response") {
		t.Errorf("unfinished exchange = %+v", todo[3])
	}
}

func TestRequestsTree(t *testing.T) {
	tr := load(t).Tree(ViewRequests)
	roots := tr.Children(-1)
	kids := tr.Children(roots[0].ID)
	if len(kids) != 4 || kids[0].Kind != KindGroup {
		t.Fatalf("children = %v", labels(kids))
	}
	if g := tr.Children(kids[0].ID); len(g) != 1 || g[0].Level != "ERROR" {
		t.Errorf("group = %+v", g)
	}
}

func TestSearch(t *testing.T) {
	tr := load(t).Tree(ViewScreens)
	r := tr.Search(Query{Text: "REQUESTID"})
	if len(r.Hits) != 1 || tr.Node(r.Hits[0]).Kind != KindExchange {
		t.Fatalf("hits = %v", r.Hits)
	}
	// hit + exchange's screen + session
	if len(r.Visible) != 3 {
		t.Errorf("visible = %v", r.Visible)
	}
	if r := tr.Search(Query{Problems: true}); len(r.Hits) != 3 {
		t.Errorf("problem hits = %d, want 3", len(r.Hits))
	}
	if r := tr.Search(Query{Tag: "somethingnew"}); len(r.Hits) != 1 {
		t.Errorf("tag hits = %d, want 1", len(r.Hits))
	}
	if r := tr.Search(Query{}); len(r.Hits) != 0 {
		t.Errorf("empty query hit %d", len(r.Hits))
	}
	// continuation lines are searchable
	if r := tr.Search(Query{Text: "Dates (2)"}); len(r.Hits) != 1 {
		t.Errorf("continuation hits = %d", len(r.Hits))
	}
}

func TestText(t *testing.T) {
	tr := load(t).Tree(ViewScreens)
	for _, n := range tr.Nodes {
		if n.Kind == KindExchange && n.Sev == SevError {
			got := n.Text()
			if strings.Count(got, "\n") != 3 || !strings.Contains(got, "✕ body") {
				t.Errorf("exchange text = %q", got)
			}
		}
		if n.Kind == KindEvent && n.Tag == "detail" {
			if got := n.Text(); !strings.HasSuffix(got, "  [1197] Dates (2)\n") {
				t.Errorf("continuation lost: %q", got)
			}
		}
	}
	if got := tr.Roots[1].Text(); !strings.Contains(got, "  (before the first screen)") {
		t.Errorf("outline = %q", got)
	}
}

func TestFailures(t *testing.T) {
	for _, v := range []View{ViewScreens, ViewRequests} {
		f := load(t).Tree(v).Failures()
		// failed #2 and unanswered #3; the ERROR after #2 is its echo, not a third failure
		if f.Failed != 1 || f.Unanswered != 1 || f.Errors != 0 || len(f.Items) != 2 {
			t.Errorf("%s: %+v", v, f)
		}
		if f.Items[0].Line >= f.Items[1].Line {
			t.Errorf("%s: not in log order: %+v", v, f.Items)
		}
	}
}

func TestPath(t *testing.T) {
	tr := load(t).Tree(ViewScreens)
	f := tr.Failures()
	path := tr.Path(f.Items[0].ID)
	if len(path) != 2 || tr.Node(path[0]).Kind != KindSession || tr.Node(path[1]).Kind != KindScreen {
		t.Errorf("path = %v", path)
	}
}

func TestAnnotations(t *testing.T) {
	d := load(t)
	tr := d.Tree(ViewScreens)
	for _, n := range tr.Nodes {
		if n.Kind == KindEvent && n.Level == "ERROR" {
			if !n.Echo || !n.HasLink || tr.Node(n.LinkID).Kind != KindExchange {
				t.Errorf("ERROR echo not linked to its call: %+v", n.Info())
			}
		}
		if n.Kind == KindEvent && n.Tag == "detail" {
			if n.GapMs != 490 {
				t.Errorf("gap = %d, want 490", n.GapMs)
			}
			if n.Date != "2026-01-01" {
				t.Errorf("date = %q", n.Date)
			}
		}
	}
}

func TestBeforeAndReport(t *testing.T) {
	d := load(t)
	tr := d.Tree(ViewScreens)
	f := tr.Failures()
	got := d.Before(tr, f.Items[0].ID, 2)
	if strings.Count(got, "\n") != 1 || !strings.Contains(got, "TodoIssues") {
		t.Errorf("before = %q", got)
	}
	rep := d.Report(tr, f.Items[0].ID, 3)
	for _, want := range []string{"== What happened ==", "✕ 500", "events before"} {
		if !strings.Contains(rep, want) {
			t.Errorf("report lacks %q:\n%s", want, rep)
		}
	}
	if strings.Contains(rep, "http") && strings.Contains(rep, "api.example") {
		t.Errorf("host not redacted:\n%s", rep)
	}
}

func TestExportAndCompare(t *testing.T) {
	d := load(t)
	tr := d.Tree(ViewScreens)
	b, err := tr.Export(Query{}, "json")
	if err != nil || !strings.Contains(string(b), `"line"`) {
		t.Errorf("json export = %s, %v", b, err)
	}
	txt, _ := tr.Export(Query{Text: "never"}, "text")
	if !strings.Contains(string(txt), "api.example/never") {
		t.Errorf("text export = %q", txt)
	}
	c := Compare(d, d)
	if len(c.Endpoints) != 1 || c.Endpoints[0].Endpoint != "GET /thing/{id}" || c.Endpoints[0].A != 1 || c.Endpoints[0].B != 1 {
		t.Errorf("endpoints = %+v", c.Endpoints)
	}
	if len(c.OnlyA)+len(c.OnlyB) != 0 {
		t.Errorf("screens differ against itself: %+v", c)
	}
}
