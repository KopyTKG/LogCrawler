package logtree

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+'-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`)

// The committed fixtures are redacted real logs. They must open, build both trees with no
// unplaced lines, and still hold no real addresses.
func TestFixtures(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/fixtures/*.log")
	if len(files) == 0 {
		t.Fatal("no fixtures: run `just fixtures`")
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			d, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(d.File.Sessions) == 0 || len(d.File.Problems) != 0 {
				t.Fatalf("sessions=%d problems=%+v", len(d.File.Sessions), d.File.Problems)
			}
			for _, v := range []View{ViewScreens, ViewRequests} {
				tr := d.Tree(v)
				tr.Failures()
				for _, r := range tr.Roots {
					_ = r.Text()
				}
			}
			raw, _ := os.ReadFile(path)
			for _, m := range emailRe.FindAllString(string(raw), -1) {
				if len(m) < 13 || m[len(m)-12:] != "example.test" {
					t.Errorf("address left in fixture: %q", m)
				}
			}
		})
	}
}
