// Package logparse reads the on-device diagnostics log written by the ServiceDesk Connect mobile
// app and turns it into typed events. The format is described in docs/log-format.md; the app owns
// it (SD-mobile-client/src/utils/{log,logFile,diagnosticsLog,httpLog}.ts).
//
// Parsing never fails on content it does not understand: such lines become plain events or
// Problems, with their original line number, so a log written by a misbehaving app still opens.
package logparse

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// File is one log file: a header, then one or two sessions (the previous and the current run).
type File struct {
	Header   Header
	Sessions []*Session
	Problems []Problem
}

// Header is the block before the first session. The keys are in the app's UI language, so match
// on position or value shape, never on Key.
type Header struct {
	Title  string // first line, e.g. "DRIVE! SDTest"
	Fields []Field
}

type Field struct {
	Key, Value string
	Line       int
}

// Session is everything under one "=== … session ===" marker.
type Session struct {
	Label     string // "previous session", "current session", or "" when the file has no marker
	StartedAt string // ISO time from the "--- log started …" line, when present
	Events    []*Event
	Exchanges []*Exchange // HTTP requests paired with their outcome, in request order
}

// Event is one timestamped line plus the continuation lines that follow it.
type Event struct {
	Line         int    // 1-based line number of the first line
	Time         string // "HH:MM:SS.mmm" — the log carries no date
	Level        string // "LOG", "ERROR", …
	Tag          string // without brackets; empty when the message has none
	Message      string // after the tag
	Continuation []string
	HTTP         *HTTPLine // set for a recognised [http] line
}

// HTTPPhase says which of the four [http] shapes a line is.
type HTTPPhase int

const (
	HTTPRequest     HTTPPhase = iota // #1 → POST url
	HTTPResponse                     // #1 ← 200 POST url 186ms
	HTTPFailure                      // #1 ✕ 500 POST url 8526ms — message
	HTTPFailureBody                  // #1 ✕ body {...}
)

type HTTPLine struct {
	Phase      HTTPPhase
	ID         int
	Method     string
	URL        string
	Status     int
	DurationMs int
	Message    string // failure message, or the body of a HTTPFailureBody line
}

// Exchange is one HTTP call across its lines. Request is nil for a response whose request line
// is missing; Outcome is nil while the call never completed (app killed, timeout).
type Exchange struct {
	ID      int
	Method  string
	URL     string
	Request *Event
	Outcome *Event
	Failed  bool
	Status  int
	Body    string // failure body, when logged
}

// Problem is a line the parser could not place.
type Problem struct {
	Line int
	Text string
	Why  string
}

var (
	markerRe  = regexp.MustCompile(`^=== (.+) ===$`)
	eventRe   = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2}\.\d{3}) ([A-Z]+)\s+(.*)$`)
	tagRe     = regexp.MustCompile(`^\[([A-Za-z]+)\]\s*(.*)$`)
	startedRe = regexp.MustCompile(`^--- log started (\S+)`)

	reqRe     = regexp.MustCompile(`^#(\d+) → (\S+) (\S+)$`)
	respRe    = regexp.MustCompile(`^#(\d+) ← (\d+) (\S+) (\S+) (\d+)ms$`)
	failRe    = regexp.MustCompile(`^#(\d+) ✕ (\d+) (\S+) (\S+) (\d+)ms(?: — (.*))?$`)
	failBodyR = regexp.MustCompile(`^#(\d+) ✕ body ?(.*)$`)
)

// Parse reads a whole log. The returned error is only for a failing reader.
func Parse(r io.Reader) (*File, error) {
	f := &File{}
	br := bufio.NewReaderSize(r, 1<<20)

	inHeader := true
	var cur *Session
	var last *Event
	lineNo := 0

	startSession := func(label string) {
		cur = &Session{Label: label}
		f.Sessions = append(f.Sessions, cur)
		last = nil
		inHeader = false
	}

	for {
		raw, err := br.ReadString('\n')
		if raw == "" && err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return f, err
		}
		lineNo++
		line := strings.TrimRight(raw, "\r\n")

		switch {
		case markerRe.MatchString(line):
			startSession(markerRe.FindStringSubmatch(line)[1])
		case eventRe.MatchString(line):
			if cur == nil {
				startSession("")
			}
			last = newEvent(lineNo, line)
			cur.Events = append(cur.Events, last)
			if m := startedRe.FindStringSubmatch(last.Message); m != nil && cur.StartedAt == "" {
				cur.StartedAt = m[1]
			}
		case inHeader:
			addHeaderLine(&f.Header, lineNo, line)
		case last != nil:
			last.Continuation = append(last.Continuation, line)
		default:
			f.Problems = append(f.Problems, Problem{Line: lineNo, Text: line, Why: "text before the first event of a session"})
		}
		if err != nil { // last line had no trailing newline
			break
		}
	}

	for _, s := range f.Sessions {
		s.Exchanges = pair(s.Events)
	}
	return f, nil
}

func addHeaderLine(h *Header, lineNo int, line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	if h.Title == "" && len(h.Fields) == 0 {
		if k, v, ok := strings.Cut(line, ": "); ok {
			h.Fields = append(h.Fields, Field{Key: k, Value: v, Line: lineNo})
		} else {
			h.Title = line
		}
		return
	}
	if k, v, ok := strings.Cut(line, ": "); ok {
		h.Fields = append(h.Fields, Field{Key: k, Value: v, Line: lineNo})
	}
}

func newEvent(lineNo int, line string) *Event {
	m := eventRe.FindStringSubmatch(line)
	e := &Event{Line: lineNo, Time: m[1], Level: m[2], Message: m[3]}
	if t := tagRe.FindStringSubmatch(e.Message); t != nil {
		e.Tag, e.Message = t[1], t[2]
	}
	if e.Tag == "http" {
		e.HTTP = parseHTTP(e.Message)
	}
	return e
}

func parseHTTP(msg string) *HTTPLine {
	if m := failBodyR.FindStringSubmatch(msg); m != nil {
		return &HTTPLine{Phase: HTTPFailureBody, ID: atoi(m[1]), Message: m[2]}
	}
	if m := reqRe.FindStringSubmatch(msg); m != nil {
		return &HTTPLine{Phase: HTTPRequest, ID: atoi(m[1]), Method: m[2], URL: m[3]}
	}
	if m := respRe.FindStringSubmatch(msg); m != nil {
		return &HTTPLine{Phase: HTTPResponse, ID: atoi(m[1]), Status: atoi(m[2]), Method: m[3], URL: m[4], DurationMs: atoi(m[5])}
	}
	if m := failRe.FindStringSubmatch(msg); m != nil {
		return &HTTPLine{Phase: HTTPFailure, ID: atoi(m[1]), Status: atoi(m[2]), Method: m[3], URL: m[4], DurationMs: atoi(m[5]), Message: m[6]}
	}
	return nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// pair joins the [http] lines of one session by id. Ids restart every session, so pairing is
// per session.
func pair(events []*Event) []*Exchange {
	byID := map[int]*Exchange{}
	var order []*Exchange
	get := func(id int) *Exchange {
		if x, ok := byID[id]; ok {
			return x
		}
		x := &Exchange{ID: id}
		byID[id] = x
		order = append(order, x)
		return x
	}
	for _, e := range events {
		h := e.HTTP
		if h == nil {
			continue
		}
		x := get(h.ID)
		switch h.Phase {
		case HTTPRequest:
			x.Request, x.Method, x.URL = e, h.Method, h.URL
		case HTTPResponse:
			x.Outcome, x.Status = e, h.Status
			x.Method, x.URL = h.Method, h.URL
		case HTTPFailure:
			x.Outcome, x.Failed, x.Status = e, true, h.Status
			x.Method, x.URL = h.Method, h.URL
		case HTTPFailureBody:
			x.Body = h.Message
		}
	}
	return order
}
