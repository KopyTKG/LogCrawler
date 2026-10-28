package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"logcrawler/internal/logparse"
	"logcrawler/internal/logtree"
)

// LogService is what the frontend calls. It holds the opened log; the frontend asks for children
// and search results lazily so only what is on screen crosses the bridge.
type LogService struct {
	mu       sync.Mutex
	doc      *logtree.Doc
	settings *Settings
	version  string
	stamp    fileStamp // of the open file when it was read
	notified bool      // the frontend was told the file changed
}

type fileStamp struct {
	mod  time.Time
	size int64
}

func stampOf(path string) (fileStamp, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return fileStamp{}, false
	}
	return fileStamp{st.ModTime(), st.Size()}, true
}

type SessionInfo struct {
	Label     string `json:"label"`
	StartedAt string `json:"startedAt"`
	Events    int    `json:"events"`
	Requests  int    `json:"requests"`
	Failed    int    `json:"failed"`
}

type ProblemInfo struct {
	Line int    `json:"line"`
	Text string `json:"text"`
	Why  string `json:"why"`
}

type FieldInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type FileInfo struct {
	Lines    int           `json:"lines"` // last line of the file that holds an event
	Path     string        `json:"path"`
	Name     string        `json:"name"`
	Title    string        `json:"title"`
	Header   []FieldInfo   `json:"header"`
	Sessions []SessionInfo `json:"sessions"`
	Problems []ProblemInfo `json:"problems"`
	Tags     []string      `json:"tags"`
}

// InitialFile is the path given on the command line, or "".
func (s *LogService) InitialFile() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return ""
}

// PickFile shows the open dialog. It returns "" when the user cancels.
func (s *LogService) PickFile() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle("Open a log").
		SetDirectory(s.settings.LastDir).
		AddFilter("Log files", "*.log;*.txt").
		AddFilter("All files", "*").
		PromptForSingleSelection()
}

// Open parses a log and makes it the current one.
func (s *LogService) Open(path string) (*FileInfo, error) {
	doc, err := logtree.Load(path)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.doc = doc
	s.stamp, _ = stampOf(path)
	s.notified = false
	s.mu.Unlock()
	s.settings.opened(path)
	return fileInfo(doc), nil
}

func fileInfo(d *logtree.Doc) *FileInfo {
	f := d.File
	info := &FileInfo{
		Path: d.Path, Name: filepath.Base(d.Path), Title: f.Header.Title, Tags: d.Tags(),
		Header: []FieldInfo{}, Sessions: []SessionInfo{}, Problems: []ProblemInfo{},
	}
	for _, h := range f.Header.Fields {
		info.Header = append(info.Header, FieldInfo{Key: h.Key, Value: h.Value})
	}
	for _, ss := range f.Sessions {
		si := SessionInfo{Label: ss.Label, StartedAt: ss.StartedAt, Events: len(ss.Events), Requests: len(ss.Exchanges)}
		for _, x := range ss.Exchanges {
			if x.Failed {
				si.Failed++
			}
		}
		info.Sessions = append(info.Sessions, si)
		for _, e := range ss.Events {
			info.Lines = max(info.Lines, e.Line+len(e.Continuation))
		}
	}
	for _, p := range f.Problems {
		info.Problems = append(info.Problems, problemInfo(p))
	}
	return info
}

func problemInfo(p logparse.Problem) ProblemInfo {
	return ProblemInfo{Line: p.Line, Text: p.Text, Why: p.Why}
}

var errNoLog = errors.New("no log is open")

func (s *LogService) tree(view string) (*logtree.Tree, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc == nil {
		return nil, errNoLog
	}
	return s.doc.Tree(logtree.View(view)), nil
}

// Children lists the children of a node; id -1 lists the roots.
func (s *LogService) Children(view string, id int) ([]logtree.NodeInfo, error) {
	t, err := s.tree(view)
	if err != nil {
		return nil, err
	}
	return t.Children(id), nil
}

// ChildrenBatch lists the children of several nodes in one call, keyed by node id.
func (s *LogService) ChildrenBatch(view string, ids []int) (map[int][]logtree.NodeInfo, error) {
	t, err := s.tree(view)
	if err != nil {
		return nil, err
	}
	out := make(map[int][]logtree.NodeInfo, len(ids))
	for _, id := range ids {
		out[id] = t.Children(id)
	}
	return out, nil
}

// Search finds events and exchanges matching q, with the ancestors that lead to them.
func (s *LogService) Search(view string, q logtree.Query) (*logtree.Result, error) {
	t, err := s.tree(view)
	if err != nil {
		return nil, err
	}
	r := t.Search(q)
	return &r, nil
}

// NodeText is the log text of a node and its subtree.
func (s *LogService) NodeText(view string, id int) (string, error) {
	t, err := s.tree(view)
	if err != nil {
		return "", err
	}
	n := t.Node(id)
	if n == nil {
		return "", errors.New("no such node")
	}
	return n.Text(), nil
}

// Copy puts text on the system clipboard.
func (s *LogService) Copy(text string) bool {
	return application.Get().Clipboard.SetText(text)
}

// Failures lists the failed and unanswered calls and the errors in the open log.
func (s *LogService) Failures(view string) (*logtree.Failures, error) {
	t, err := s.tree(view)
	if err != nil {
		return nil, err
	}
	f := t.Failures()
	return &f, nil
}

// Path lists the ancestors of a node, root first.
func (s *LogService) Path(view string, id int) ([]int, error) {
	t, err := s.tree(view)
	if err != nil {
		return nil, err
	}
	return t.Path(id), nil
}

// Version is the build's version, shown in the title and the Info panel.
func (s *LogService) Version() string { return s.version }

// Recent lists the files opened before, newest first, leaving out ones that no longer exist.
func (s *LogService) Recent() []string {
	s.settings.mu.Lock()
	defer s.settings.mu.Unlock()
	out := []string{}
	for _, p := range s.settings.Recent {
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// SavedView is the view the user left on last time.
func (s *LogService) SavedView() string { return s.settings.View }

func (s *LogService) SaveView(view string) {
	s.settings.mu.Lock()
	s.settings.View = view
	s.settings.mu.Unlock()
	s.settings.save()
}

// Reload reads the open file again.
func (s *LogService) Reload() (*FileInfo, error) {
	s.mu.Lock()
	doc := s.doc
	s.mu.Unlock()
	if doc == nil {
		return nil, errNoLog
	}
	return s.Open(doc.Path)
}

// pollChanged tells the frontend when the open file changed on disk. It is called on a timer.
func (s *LogService) pollChanged() {
	s.mu.Lock()
	doc, notified, was := s.doc, s.notified, s.stamp
	s.mu.Unlock()
	if doc == nil || notified {
		return
	}
	if now, ok := stampOf(doc.Path); ok && now != was {
		s.mu.Lock()
		s.notified = true
		s.mu.Unlock()
		application.Get().Event.Emit("file-changed", doc.Path)
	}
}

func (s *LogService) current() (*logtree.Doc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc == nil {
		return nil, errNoLog
	}
	return s.doc, nil
}

// Before is the log text of the n events before a node, whatever screen they were on.
func (s *LogService) Before(view string, id, n int) (string, error) {
	d, err := s.current()
	if err != nil {
		return "", err
	}
	return d.Before(d.Tree(logtree.View(view)), id, n), nil
}

// Report is the redacted bug-report text for a node.
func (s *LogService) Report(view string, id, n int) (string, error) {
	d, err := s.current()
	if err != nil {
		return "", err
	}
	return d.Report(d.Tree(logtree.View(view)), id, n), nil
}

// Export writes the matches of q (the failures when q is empty) to a file the user picks, as JSON
// when the name ends in .json and as text otherwise. It returns the path, or "" when cancelled.
func (s *LogService) Export(view string, q logtree.Query) (string, error) {
	d, err := s.current()
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.SaveFile().
		SetFilename("logcrawler-export.txt").
		SetDirectory(s.settings.LastDir).
		AddFilter("Text", "*.txt").
		AddFilter("JSON", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	format := "text"
	if strings.EqualFold(filepath.Ext(path), ".json") {
		format = "json"
	}
	b, err := d.Tree(logtree.View(view)).Export(q, format)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, b, 0o644)
}

// PickCompare asks for a second log and compares it with the open one.
func (s *LogService) PickCompare() (*logtree.Comparison, error) {
	d, err := s.current()
	if err != nil {
		return nil, err
	}
	path, err := s.PickFile()
	if err != nil || path == "" {
		return nil, err
	}
	other, err := logtree.Load(path)
	if err != nil {
		return nil, err
	}
	return logtree.Compare(d, other), nil
}
