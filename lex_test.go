package kdl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestConformance(t *testing.T) {
	// Verify that all valid inputs from the conformance suite can lex
	// without error.

	ms, err := filepath.Glob("testdata/valid/*.kdl")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	for _, n := range ms {
		t.Run(n, func(t *testing.T) {
			var b bytes.Buffer
			f, err := os.Open(n)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			l := NewLexer(f)
			for {
				tok := l.Next()
				fmt.Fprintln(&b, tok)
				if tok.typ == tokErr {
					t.Errorf("got error:\n\n%s", b.String())
				} else if tok.typ == tokEOF {
					return
				}
			}
		})
	}
}
