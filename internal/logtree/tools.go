package logtree

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"logcrawler/internal/logparse"
	"logcrawler/internal/redact"
)

type loc struct{ session, index int }

func (d *Doc) locate(e *logparse.Event) (loc, bool) {
	if d.index == nil {
		d.index = map[*logparse.Event]loc{}
		for si, s := range d.File.Sessions {
			for i, ev := range s.Events {
				d.index[ev] = loc{si, i}
			}
		}
	}
	l, ok := d.index[e]
	return l, ok
}

// Before returns up to n events that happened just before the node's first event, in the same
// session and whatever screen they were on, as log text. Containers have no events, so no context.
func (d *Doc) Before(t *Tree, id, n int) string {
	node := t.Node(id)
	if node == nil || len(node.Events) == 0 {
		return ""
	}
	l, ok := d.locate(node.Events[0])
	if !ok {
		return ""
	}
	events := d.File.Sessions[l.session].Events
	from := max(0, l.index-n)
	var parts []string
	for _, e := range events[from:l.index] {
		parts = append(parts, EventText(e))
	}
	return strings.Join(parts, "\n")
}

// Report is a bug-report text for a node: what app and platform, what happened, and the events
// before it. It is redacted, because it is made to be pasted into a ticket.
func (d *Doc) Report(t *Tree, id, n int) string {
	node := t.Node(id)
	if node == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("LogCrawler report for " + filepath.Base(d.Path) + "\n")
	if node.Date != "" {
		b.WriteString("Date: " + node.Date + "\n")
	}
	for _, f := range d.File.Header.Fields {
		if !strings.HasPrefix(f.Value, "http") { // the addresses say nothing a reader needs
			b.WriteString(f.Key + ": " + f.Value + "\n")
		}
	}
	b.WriteString("\n== What happened ==\n")
	b.WriteString(node.Text())
	if before := d.Before(t, id, n); before != "" {
		fmt.Fprintf(&b, "\n== The %d events before ==\n%s\n", n, before)
	}
	return redact.New().Text(b.String())
}

// ExportItem is one entry of an export.
type ExportItem struct {
	Line  int    `json:"line"`
	Date  string `json:"date,omitempty"`
	Time  string `json:"time"`
	Level string `json:"level"`
	Tag   string `json:"tag,omitempty"`
	Text  string `json:"text"`
}

// Export lists the nodes a query matches, or the failures when the query is empty, as text or JSON.
func (t *Tree) Export(q Query, format string) ([]byte, error) {
	var ids []int
	if q.empty() {
		for _, f := range t.Failures().Items {
			ids = append(ids, f.ID)
		}
	} else {
		ids = t.Search(q).Hits
	}
	if format == "json" {
		items := make([]ExportItem, 0, len(ids))
		for _, id := range ids {
			n := t.Nodes[id]
			items = append(items, ExportItem{Line: n.Line, Date: n.Date, Time: n.Time, Level: n.Level, Tag: n.Tag, Text: n.Text()})
		}
		return json.MarshalIndent(items, "", "  ")
	}
	var b strings.Builder
	for _, id := range ids {
		b.WriteString(t.Nodes[id].Text() + "\n")
	}
	return []byte(b.String()), nil
}

// Summary counts what happened in one log.
type Summary struct {
	Name       string `json:"name"`
	Events     int    `json:"events"`
	Requests   int    `json:"requests"`
	Failed     int    `json:"failed"`
	Errors     int    `json:"errors"`
	Unanswered int    `json:"unanswered"`
	Slow       int    `json:"slow"`
}

// EndpointStat is how many calls to one endpoint failed in each log.
type EndpointStat struct {
	Endpoint string `json:"endpoint"`
	A        int    `json:"a"`
	B        int    `json:"b"`
}

type Comparison struct {
	A         Summary        `json:"a"`
	B         Summary        `json:"b"`
	Endpoints []EndpointStat `json:"endpoints"` // endpoints that failed in either log
	OnlyA     []string       `json:"onlyA"`     // screens visited only in A
	OnlyB     []string       `json:"onlyB"`
}

var (
	idSegRe = regexp.MustCompile(`^(?:\d+|[0-9a-fA-F]{8}-[0-9a-fA-F-]{27})$`)
	hostRe  = regexp.MustCompile(`^https?://[^/]+`)
)

// endpoint is "METHOD /path" with ids as {id} and no host or query, so the same call in two logs
// (or on two servers) compares equal.
func endpoint(method, url string) string {
	url = hostRe.ReplaceAllString(url, "")
	if i := strings.IndexAny(url, "?#"); i >= 0 {
		url = url[:i]
	}
	segs := strings.Split(url, "/")
	for i, s := range segs {
		if idSegRe.MatchString(s) {
			segs[i] = "{id}"
		}
	}
	return method + " " + strings.Join(segs, "/")
}

func (d *Doc) summary() (Summary, map[string]int, map[string]bool) {
	t := d.Tree(ViewScreens)
	f := t.Failures()
	s := Summary{Name: filepath.Base(d.Path), Failed: f.Failed, Errors: f.Errors, Unanswered: f.Unanswered, Slow: f.Slow}
	failing := map[string]int{}
	screens := map[string]bool{}
	for _, ss := range d.File.Sessions {
		s.Events += len(ss.Events)
		s.Requests += len(ss.Exchanges)
		for _, x := range ss.Exchanges {
			if x.Failed {
				failing[endpoint(x.Method, x.URL)]++
			}
		}
		for _, e := range ss.Events {
			if e.Tag == "nav" {
				if name, _, _ := strings.Cut(navTarget(e.Message), " "); name != "" {
					screens[name] = true
				}
			}
		}
	}
	return s, failing, screens
}

// Compare sets two logs side by side: the counts, the endpoints that failed, the screens visited.
func Compare(a, b *Doc) *Comparison {
	sa, fa, scA := a.summary()
	sb, fb, scB := b.summary()
	c := &Comparison{A: sa, B: sb, Endpoints: []EndpointStat{}, OnlyA: []string{}, OnlyB: []string{}}
	keys := map[string]bool{}
	for k := range fa {
		keys[k] = true
	}
	for k := range fb {
		keys[k] = true
	}
	for k := range keys {
		c.Endpoints = append(c.Endpoints, EndpointStat{Endpoint: k, A: fa[k], B: fb[k]})
	}
	sort.Slice(c.Endpoints, func(i, j int) bool {
		x, y := c.Endpoints[i], c.Endpoints[j]
		if x.A+x.B != y.A+y.B {
			return x.A+x.B > y.A+y.B
		}
		return x.Endpoint < y.Endpoint
	})
	for k := range scA {
		if !scB[k] {
			c.OnlyA = append(c.OnlyA, k)
		}
	}
	for k := range scB {
		if !scA[k] {
			c.OnlyB = append(c.OnlyB, k)
		}
	}
	sort.Strings(c.OnlyA)
	sort.Strings(c.OnlyB)
	return c
}
