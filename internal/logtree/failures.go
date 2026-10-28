package logtree

import "sort"

// Failure is one thing that went wrong: a failed or unanswered call, or an ERROR that is not just
// the echo of a failed call.
type Failure struct {
	ID   int `json:"id"`
	Line int `json:"line"`
}

type Failures struct {
	Items      []Failure `json:"items"` // in log order
	Failed     int       `json:"failed"`
	Errors     int       `json:"errors"`
	Unanswered int       `json:"unanswered"`
	Slow       int       `json:"slow"` // slow calls that did not fail; not in Items
}

// Failures lists what went wrong, ordered by line in the file.
func (t *Tree) Failures() Failures {
	out := Failures{Items: []Failure{}}
	for _, n := range t.Nodes {
		switch {
		case n.Kind == KindExchange && n.Sev == SevError:
			out.Failed++
		case n.Kind == KindExchange && n.Unanswered:
			out.Unanswered++
		case n.Kind == KindEvent && n.Sev == SevError && !n.Echo:
			out.Errors++
		case n.Kind == KindExchange && n.Slow:
			out.Slow++
			continue
		default:
			continue
		}
		out.Items = append(out.Items, Failure{ID: n.ID, Line: n.Line})
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].Line < out.Items[j].Line })
	return out
}

// Path lists the ancestors of a node, root first, so the UI can expand down to it.
func (t *Tree) Path(id int) []int {
	path := []int{}
	n := t.Node(id)
	if n == nil {
		return path
	}
	for p := n.Parent; p != nil; p = p.Parent {
		path = append([]int{p.ID}, path...)
	}
	return path
}
