// Package logtree turns a parsed log into trees for the UI. A tree is a view over the parser's
// events: the parser stays flat and typed, and several views can share it.
//
// Two views exist: ViewScreens (session → screen → events, screens from [nav]) answers "what was the
// user doing when this failed"; ViewRequests (session → HTTP exchanges) answers "which calls went
// wrong". Nodes are numbered by their index in Tree.Nodes so the UI can ask for children lazily and
// only expanded nodes cross the bridge.
package logtree

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"logcrawler/internal/logparse"
)

type View string

const (
	ViewScreens  View = "screens"
	ViewRequests View = "requests"
)

type Kind string

const (
	KindSession  Kind = "session"
	KindScreen   Kind = "screen"
	KindGroup    Kind = "group"
	KindExchange Kind = "exchange"
	KindEvent    Kind = "event"
	KindLine     Kind = "line" // one log line of an exchange
)

type Severity int

const (
	SevNone Severity = iota
	SevWarn
	SevError
)

const maxLabel = 300

// SlowMs is the duration from which a call is called slow.
const SlowMs = 3000

// Node is one row of a tree. Events holds the log events it stands for (one for an event or line,
// all lines for an exchange, none for a pure container).
type Node struct {
	ID       int
	Kind     Kind
	Label    string
	Time     string
	Level    string
	Tag      string
	Line     int
	Sev      Severity
	Errors   int // error and failed-call nodes in this subtree (the node itself included)
	Warnings int
	Events   []*logparse.Event
	// Echo marks an ERROR event that repeats the failed call logged just before it.
	Echo bool
	// Unanswered marks an exchange that never got a response.
	Unanswered bool
	// Slow marks an exchange that took SlowMs or longer; DurationMs is its time in ms.
	Slow       bool
	DurationMs int
	// GapMs is the time since the previous event of the session; Date is the day the event
	// happened, when the session start says so (the log carries no date, and time wraps at midnight).
	GapMs int
	Date  string
	// LinkID is the exchange an echo ERROR belongs to; valid when HasLink.
	LinkID   int
	HasLink  bool
	Parent   *Node
	Children []*Node

	text string // lowercase searchable text, built lazily
}

type Tree struct {
	View  View
	Nodes []*Node
	Roots []*Node
}

// Node returns the node with the given id, or nil.
func (t *Tree) Node(id int) *Node {
	if id < 0 || id >= len(t.Nodes) {
		return nil
	}
	return t.Nodes[id]
}

// Build makes the tree for one view.
func Build(f *logparse.File, v View) *Tree {
	t := &Tree{View: v}
	for _, s := range f.Sessions {
		root := t.add(nil, &Node{Kind: KindSession, Label: sessionLabel(s)})
		switch v {
		case ViewRequests:
			buildRequests(t, root, s)
		default:
			buildScreens(t, root, s)
		}
	}
	annotate(t, f)
	for _, r := range t.Roots {
		count(r)
	}
	return t
}

func (t *Tree) add(parent, n *Node) *Node {
	n.ID = len(t.Nodes)
	n.Parent = parent
	t.Nodes = append(t.Nodes, n)
	if parent == nil {
		t.Roots = append(t.Roots, n)
	} else {
		parent.Children = append(parent.Children, n)
	}
	return n
}

func sessionLabel(s *logparse.Session) string {
	label := s.Label
	if label == "" {
		label = "session"
	}
	if s.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, s.StartedAt); err == nil {
			label += " — started " + t.Format("2006-01-02 15:04:05") + " UTC"
		} else {
			label += " — started " + s.StartedAt
		}
	}
	return label
}

// buildScreens groups events under the screen the user was on. Each [http] call becomes one
// exchange node, placed at its first line.
func buildScreens(t *Tree, root *Node, s *logparse.Session) {
	exch := exchangeIndex(s)
	placed := map[int]*Node{}
	var screen *Node
	for _, e := range s.Events {
		if e.Tag == "nav" {
			screen = t.add(root, &Node{
				Kind: KindScreen, Label: navTarget(e.Message), Time: e.Time, Line: e.Line,
				Events: []*logparse.Event{e},
			})
			continue
		}
		if screen == nil {
			screen = t.add(root, &Node{Kind: KindScreen, Label: "(before the first screen)", Line: e.Line})
		}
		if e.HTTP != nil {
			if x, ok := placed[e.HTTP.ID]; ok {
				addExchangeLine(t, x, e)
				continue
			}
			placed[e.HTTP.ID] = newExchange(t, screen, e, exch[e.HTTP.ID])
			continue
		}
		t.add(screen, eventNode(KindEvent, e))
	}
}

// buildRequests lists every exchange in order, after a group of the errors and warnings that are
// not part of an HTTP call.
func buildRequests(t *Tree, root *Node, s *logparse.Session) {
	exch := exchangeIndex(s)
	var problems []*logparse.Event
	for _, e := range s.Events {
		if e.HTTP == nil && severity(e) != SevNone {
			problems = append(problems, e)
		}
	}
	if len(problems) > 0 {
		g := t.add(root, &Node{Kind: KindGroup, Label: "Errors and warnings outside HTTP calls"})
		for _, e := range problems {
			t.add(g, eventNode(KindEvent, e))
		}
	}
	placed := map[int]*Node{}
	for _, e := range s.Events {
		if e.HTTP == nil {
			continue
		}
		if x, ok := placed[e.HTTP.ID]; ok {
			addExchangeLine(t, x, e)
			continue
		}
		placed[e.HTTP.ID] = newExchange(t, root, e, exch[e.HTTP.ID])
	}
}

func exchangeIndex(s *logparse.Session) map[int]*logparse.Exchange {
	m := make(map[int]*logparse.Exchange, len(s.Exchanges))
	for _, x := range s.Exchanges {
		m[x.ID] = x
	}
	return m
}

func newExchange(t *Tree, parent *Node, first *logparse.Event, x *logparse.Exchange) *Node {
	n := t.add(parent, &Node{Kind: KindExchange, Time: first.Time, Tag: "http", Line: first.Line})
	n.Label, n.Sev = exchangeLabel(x)
	n.Unanswered = x != nil && x.Outcome == nil
	if x != nil && x.Outcome != nil {
		n.DurationMs = x.Outcome.HTTP.DurationMs
		n.Slow = n.DurationMs >= SlowMs
	}
	addExchangeLine(t, n, first)
	return n
}

func addExchangeLine(t *Tree, x *Node, e *logparse.Event) {
	x.Events = append(x.Events, e)
	t.add(x, eventNode(KindLine, e))
}

func exchangeLabel(x *logparse.Exchange) (string, Severity) {
	if x == nil {
		return "(unmatched http line)", SevWarn
	}
	call := x.Method + " " + x.URL
	switch {
	case x.Outcome == nil:
		return fmt.Sprintf("#%d %s — no response", x.ID, call), SevWarn
	case x.Failed:
		return fmt.Sprintf("#%d %s ✕ %d %dms", x.ID, call, x.Status, x.Outcome.HTTP.DurationMs), SevError
	}
	sev := SevNone
	if x.Status >= 400 {
		sev = SevWarn
	}
	return fmt.Sprintf("#%d %s ← %d %dms", x.ID, call, x.Status, x.Outcome.HTTP.DurationMs), sev
}

func eventNode(k Kind, e *logparse.Event) *Node {
	return &Node{
		Kind: k, Label: truncate(eventLabel(e)), Time: e.Time, Level: e.Level, Tag: e.Tag,
		Line: e.Line, Sev: severity(e), Events: []*logparse.Event{e},
	}
}

func eventLabel(e *logparse.Event) string {
	if e.Tag == "" {
		return e.Message
	}
	return "[" + e.Tag + "] " + e.Message
}

func severity(e *logparse.Event) Severity {
	switch {
	case e.Level == "ERROR":
		return SevError
	case e.Level == "WARN" || strings.HasPrefix(e.Message, "WARN"):
		return SevWarn
	}
	return SevNone
}

// navTarget is the screen a [nav] message moves to: "<previous> → <current>".
func navTarget(msg string) string {
	if i := strings.LastIndex(msg, " → "); i >= 0 {
		return msg[i+len(" → "):]
	}
	return msg
}

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxLabel {
		return s
	}
	r := []rune(s)
	return string(r[:maxLabel]) + "…"
}

// count fills Errors and Warnings bottom-up. Line nodes are parts of an exchange and are not
// counted on their own.
func count(n *Node) {
	if n.Kind == KindEvent || n.Kind == KindExchange {
		switch n.Sev {
		case SevError:
			n.Errors++
		case SevWarn:
			n.Warnings++
		}
	}
	for _, c := range n.Children {
		count(c)
		n.Errors += c.Errors
		n.Warnings += c.Warnings
	}
}

// EventText renders one event as it appeared in the log, continuation lines included.
func EventText(e *logparse.Event) string {
	var b strings.Builder
	b.WriteString(e.Time + " " + e.Level + "  " + eventLabel(e))
	for _, c := range e.Continuation {
		b.WriteString("\n" + c)
	}
	return b.String()
}

// Text renders a node and its subtree for pasting into a bug report. Containers become an
// indented outline; exchanges and events are the original log lines.
func (n *Node) Text() string {
	var b strings.Builder
	writeText(&b, n, 0)
	return b.String()
}

func writeText(b *strings.Builder, n *Node, depth int) {
	pad := strings.Repeat("  ", depth)
	switch n.Kind {
	case KindEvent, KindLine:
		writeIndented(b, pad, EventText(n.Events[0]))
		return
	case KindExchange:
		for _, e := range n.Events {
			writeIndented(b, pad, EventText(e))
		}
		return
	}
	head := n.Label
	if n.Time != "" {
		head = n.Time + " " + head
	}
	writeIndented(b, pad, head)
	for _, c := range n.Children {
		writeText(b, c, depth+1)
	}
}

func writeIndented(b *strings.Builder, pad, text string) {
	for _, l := range strings.Split(text, "\n") {
		b.WriteString(pad + l + "\n")
	}
}

// annotate fills what needs a look at the whole session: which ERROR echoes which failed call,
// the gap since the previous event, and the date.
func annotate(t *Tree, f *logparse.File) {
	byEvent := map[*logparse.Event]*Node{} // exchange node of each of its events
	for _, n := range t.Nodes {
		if n.Kind == KindExchange {
			for _, e := range n.Events {
				byEvent[e] = n
			}
		}
	}
	echo := map[*logparse.Event]*Node{}
	gap := map[*logparse.Event]int{}
	date := map[*logparse.Event]string{}
	for _, s := range f.Sessions {
		day := 0
		base, hasBase := sessionDate(s)
		for i, e := range s.Events {
			if i > 0 {
				prev := s.Events[i-1]
				d := clockMs(e.Time) - clockMs(prev.Time)
				if d < -12*3600*1000 { // midnight
					day++
					d += 24 * 3600 * 1000
				}
				if d > 0 {
					gap[e] = d
				}
				if prev.HTTP != nil && e.Level == "ERROR" &&
					(prev.HTTP.Phase == logparse.HTTPFailure || prev.HTTP.Phase == logparse.HTTPFailureBody) {
					echo[e] = byEvent[prev]
				}
			}
			if hasBase {
				date[e] = base.AddDate(0, 0, day).Format("2006-01-02")
			}
		}
	}
	for _, n := range t.Nodes {
		if len(n.Events) == 0 {
			continue
		}
		e := n.Events[0]
		n.GapMs, n.Date = gap[e], date[e]
		if ex, ok := echo[e]; ok && n.Kind == KindEvent {
			n.Echo = true
			if ex != nil {
				n.LinkID, n.HasLink = ex.ID, true
			}
		}
	}
}

func sessionDate(s *logparse.Session) (time.Time, bool) {
	if len(s.StartedAt) < 10 {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", s.StartedAt[:10])
	return d, err == nil
}

// clockMs is the time of day of an "HH:MM:SS.mmm" stamp in milliseconds.
func clockMs(t string) int {
	if len(t) < 12 {
		return 0
	}
	n := func(a, b int) int { v, _ := strconv.Atoi(t[a:b]); return v }
	return ((n(0, 2)*60+n(3, 5))*60+n(6, 8))*1000 + n(9, 12)
}
