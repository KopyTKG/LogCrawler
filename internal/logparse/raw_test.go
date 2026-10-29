package logparse

import (
	"os"
	"path/filepath"
	"testing"
)

// Local-only smoke test over the real logs in testdata/raw (gitignored, so absent on a fresh
// clone and in CI — the test then skips). It checks the parser's promises, not the contents:
// nothing fails, every file has a session, and no line is left unplaced.
func TestRawLogs(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/raw/*.log")
	if len(files) == 0 {
		t.Skip("no raw logs in testdata/raw")
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			fh, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer fh.Close()
			f, err := Parse(fh)
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Sessions) == 0 {
				t.Fatal("no sessions")
			}
			if len(f.Problems) != 0 {
				t.Errorf("unplaced lines: %+v", f.Problems)
			}
			for _, s := range f.Sessions {
				for _, e := range s.Events {
					if e.Tag == "http" && e.HTTP == nil {
						t.Errorf("line %d: [http] line not recognised: %q", e.Line, e.Message)
					}
				}
			}
			t.Logf("%d session(s), header title %q, %d fields", len(f.Sessions), f.Header.Title, len(f.Header.Fields))
		})
	}
}
