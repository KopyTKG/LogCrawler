// Command redact turns raw device logs into committable fixtures.
//
//	redact -o testdata/fixtures testdata/raw/*.log
//
// Outputs are named real-1.log, real-2.log, … in input order, because the raw file names carry the
// customer. A fixture is still a raw log with placeholders; read it before committing.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"logcrawler/internal/redact"
)

func main() {
	out := flag.String("o", "testdata/fixtures", "output directory")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: redact [-o dir] file.log ...")
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i, in := range flag.Args() {
		b, err := os.ReadFile(in)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		dst := filepath.Join(*out, fmt.Sprintf("real-%d.log", i+1))
		if err := os.WriteFile(dst, []byte(redact.New().File(string(b))), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %s\n", filepath.Base(in), dst)
	}
}
