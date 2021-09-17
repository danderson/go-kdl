package kdl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConformance(t *testing.T) {
	updateOne, _ := strconv.ParseBool(os.Getenv("KDL_TEST_UPDATE_ONE"))
	// Verify that all valid inputs from the conformance suite can lex
	// without error.

	ms, err := filepath.Glob("testdata/valid/*.kdl")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}

	for _, n := range ms {
		t.Run(n, func(t *testing.T) {
			var b bytes.Buffer
			bs, err := os.ReadFile(n)
			if err != nil {
				t.Fatal(err)
			}
			bin := bytes.NewBuffer(bs)
			l := NewLexer(bin)
			for {
				tok := l.Next()
				fmt.Fprintln(&b, tok)
				if tok.typ == tokErr {
					t.Fatalf("got error:\n%s", b.String())
				} else if tok.typ == tokEOF {
					break
				}
			}
			wantName := strings.Replace(n, "/valid/", "/lex/", 1)
			wantbs, err := os.ReadFile(wantName)
			if os.IsNotExist(err) {
				if updateOne {
					if err := os.WriteFile(wantName, b.Bytes(), 0644); err != nil {
						t.Fatalf("trying to update %s: %v", wantName, err)
					}
					updateOne = false
					wantbs = b.Bytes()
				} else {
					t.Fatalf("no expected lexer output, got:\n%s\n%s", b.String(), string(bs))
				}
			} else if err != nil {
				t.Fatalf("reading valid file: %v", err)
			}
			if diff := cmp.Diff(strings.Split(b.String(), "\n"), strings.Split(string(wantbs), "\n")); diff != "" {
				if updateOne {
					if err := os.WriteFile(wantName, b.Bytes(), 0644); err != nil {
						t.Fatalf("trying to update %s: %v", wantName, err)
					}
					updateOne = false
				} else {
					t.Fatalf("unexpected lex output (-got+want):\n%s\n%s", diff, string(bs))
				}
			}
		})
	}
}
