package logtree

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"logcrawler/internal/logparse"
)

// Doc is one opened log with its trees.
type Doc struct {
	Path  string
	File  *logparse.File
	trees map[View]*Tree
	index map[*logparse.Event]loc // built on first use by locate
}

// Load reads and parses a log file. Content the parser does not understand never fails the load.
func Load(path string) (*Doc, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	f, err := logparse.Parse(fh)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.Base(path), err)
	}
	return NewDoc(path, f), nil
}

func NewDoc(path string, f *logparse.File) *Doc {
	return &Doc{Path: path, File: f, trees: map[View]*Tree{
		ViewScreens:  Build(f, ViewScreens),
		ViewRequests: Build(f, ViewRequests),
	}}
}

// Tree returns the tree for a view; an unknown view gets the screens tree.
func (d *Doc) Tree(v View) *Tree {
	if t, ok := d.trees[v]; ok {
		return t
	}
	return d.trees[ViewScreens]
}

// NodeInfo is a node as the UI sees it.
type NodeInfo struct {
	ID          int      `json:"id"`
	Kind        Kind     `json:"kind"`
	Label       string   `json:"label"`
	Time        string   `json:"time"`
	Level       string   `json:"level"`
	Tag         string   `json:"tag"`
	Line        int      `json:"line"`
	Sev         Severity `json:"sev"`
	Errors      int      `json:"errors"`
	Warnings    int      `json:"warnings"`
	HasChildren bool     `json:"hasChildren"`
	Link        int      `json:"link"` // node id of the call an echo ERROR belongs to, or -1
	DurationMs  int      `json:"durationMs"`
	Slow        bool     `json:"slow"`
	GapMs       int      `json:"gapMs"`
	Date        string   `json:"date"`
}

func (n *Node) Info() NodeInfo {
	link := -1
	if n.HasLink {
		link = n.LinkID
	}
	return NodeInfo{
		Link: link, DurationMs: n.DurationMs, Slow: n.Slow, GapMs: n.GapMs, Date: n.Date,
		ID: n.ID, Kind: n.Kind, Label: n.Label, Time: n.Time, Level: n.Level, Tag: n.Tag,
		Line: n.Line, Sev: n.Sev, Errors: n.Errors, Warnings: n.Warnings,
		HasChildren: len(n.Children) > 0,
	}
}

// Children lists the children of a node; id -1 lists the roots.
func (t *Tree) Children(id int) []NodeInfo {
	list := t.Roots
	if id >= 0 {
		n := t.Node(id)
		if n == nil {
			return []NodeInfo{}
		}
		list = n.Children
	}
	out := make([]NodeInfo, len(list))
	for i, c := range list {
		out[i] = c.Info()
	}
	return out
}

// Query is a search. Empty fields do not filter.
type Query struct {
	Text     string `json:"text"`
	Tag      string `json:"tag"`
	Problems bool   `json:"problems"` // only warnings, errors and failed calls
	Slow     bool   `json:"slow"`     // only calls that took SlowMs or longer
}

func (q Query) empty() bool {
	return strings.TrimSpace(q.Text) == "" && q.Tag == "" && !q.Problems && !q.Slow
}

// Result lists the matching nodes and every node that must stay visible to reach them.
type Result struct {
	Hits      []int `json:"hits"`
	Visible   []int `json:"visible"`
	Truncated bool  `json:"truncated"`
}

const maxHits = 5000

// Search finds events and exchanges matching q. Matching is case-insensitive over the whole log
// text of the node, continuation lines included.
func (t *Tree) Search(q Query) Result {
	res := Result{Hits: []int{}, Visible: []int{}}
	if q.empty() {
		return res
	}
	needle := strings.ToLower(strings.TrimSpace(q.Text))
	visible := map[int]bool{}
	for _, n := range t.Nodes {
		if n.Kind != KindEvent && n.Kind != KindExchange {
			continue
		}
		if q.Problems && n.Sev == SevNone {
			continue
		}
		if q.Slow && !n.Slow {
			continue
		}
		if q.Tag != "" && n.Tag != q.Tag {
			continue
		}
		if needle != "" && !strings.Contains(n.searchText(), needle) {
			continue
		}
		if len(res.Hits) >= maxHits {
			res.Truncated = true
			break
		}
		res.Hits = append(res.Hits, n.ID)
		for a := n; a != nil && !visible[a.ID]; a = a.Parent {
			visible[a.ID] = true
		}
	}
	for id := range visible {
		res.Visible = append(res.Visible, id)
	}
	sort.Ints(res.Visible)
	return res
}

func (n *Node) searchText() string {
	if n.text == "" {
		var b strings.Builder
		for _, e := range n.Events {
			b.WriteString(EventText(e))
			b.WriteByte('\n')
		}
		n.text = strings.ToLower(b.String())
	}
	return n.text
}

// Tags lists the distinct tags in the log, sorted, for the filter.
func (d *Doc) Tags() []string {
	seen := map[string]bool{}
	for _, s := range d.File.Sessions {
		for _, e := range s.Events {
			if e.Tag != "" {
				seen[e.Tag] = true
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}
