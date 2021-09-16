package main

import (
	"fmt"
	"log"
	"os"

	"github.com/danderson/go-kdl"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("open %s: %v", os.Args[1], err)
	}
	l := kdl.NewLexer(f)
	for {
		tok := l.Next()
		fmt.Println(tok)
		if tok.String() == "EOF" {
			return
		}
	}
}
