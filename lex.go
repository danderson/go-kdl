package kdl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

func identifierCharacter(r rune) bool {
	if r < 0x20 || r > 0x10FFFF {
		return false
	}

	const excluded = `\/<>{};=,"`
	for _, e := range excluded {
		if r == e {
			return false
		}
	}

	return true
}

func digit(r rune) bool {
	return r >= '0' && r <= '9'
}

func numberStart(r rune) bool {
	return digit(r) || r == '+' || r == '-'
}

func identifierStart(r rune) bool {
	return identifierCharacter(r) && !digit(r)
}

const spaces = "\t \xA0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u202F\u205F\u3000"

type tokenType int

const (
	tokEOF tokenType = iota
	tokErr
	tokInt
	tokFloat
	tokNewline
	tokIgnoreNode
	tokSpace
)

type token struct {
	typ tokenType
	err error // set iff typ==itemErr
}

type lexer struct {
	tokens chan token
	close  chan struct{} // closed by Close

	r  *bufio.Reader
	rs []rune
	// TODO: will we ever need to peek >1 rune? If not, can save some
	// array nonsense here.
	peekrs []rune // if non-zero, un-next()-ed runes in reverse order (last first)
	atEOF  bool   // flips once to true when lexer finds EOF
}

func newLexer(r io.Reader) *lexer {
	var br *bufio.Reader
	if sr, ok := r.(*bufio.Reader); ok {
		br = sr
	} else {
		br = bufio.NewReader(r)
	}
	ret := &lexer{
		tokens: make(chan token),
		close:  make(chan struct{}),
		r:      br,
		rs:     make([]rune, 0, 1024),
	}
	go ret.lex()
	return ret
}

func (l *lexer) Next() token {
	// Handily, when the channel is closed, the zero value is
	// returned, whose typ is tokEOF. So, we EOF for ever once
	// closed.
	return <-l.tokens
}

var lexClosed = errors.New("lexer closed")

func (l *lexer) emit(typ tokenType) {
	select {
	case l.tokens <- token{typ: typ}:
		l.rs = l.rs[:0]
	case <-l.close:
		// Will get recovered at the top level of lex()
		panic(lexClosed)
	}
}

func (l *lexer) err(format string, args ...interface{}) lexFn {
	select {
	case l.tokens <- token{typ: tokErr, err: fmt.Errorf(format, args...)}:
	case <-l.close:
		panic(lexClosed)
	}
	return nil // Will break out of the top-level lex loop and clean up.
}

const eof = -1 // outside the valid range for unicode codepoints

func (l *lexer) next() rune {
	if len(l.peekrs) > 0 {
		r := l.peekrs[len(l.peekrs)-1]
		l.peekrs = l.peekrs[:len(l.peekrs)-1]
		return r
	}
	if l.atEOF {
		return eof
	}

	r, _, err := l.r.ReadRune()
	if err == io.EOF {
		l.atEOF = true
		return eof
	} else if err != nil {
		// TODO: something else?
		l.atEOF = true
		return eof
	}
	l.rs = append(l.rs, r)
	return r
}

func (l *lexer) backup() {
	if l.atEOF {
		// "backing up" from EOF is meaningless, therefore do nothing.
		return
	}
	if len(l.rs) == 0 {
		panic("cannot backup with nothing buffered")
	}
	l.peekrs = append(l.peekrs, l.rs[len(l.rs)-1])
	l.rs = l.rs[:len(l.rs)-1]
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptNewline() bool {
	r := l.next()
	switch {
	case r == eof:
		return false
	case r == '\r':
		if l.peek() == '\n' {
			l.next()
		}
		return true
	case strings.IndexRune(newline, r) >= 0:
		return true
	default:
		return false
	}
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) until(invalid string) {
	for strings.IndexRune(invalid, l.next()) < 0 {
	}
	l.backup()
}

type lexFn func() lexFn

func (l *lexer) lex() {
	defer func() {
		close(l.tokens)
		if r := recover(); r != nil {
			if r != lexClosed {
				panic(r)
			}
		}
	}()

	for st := l.lexStart; st != nil; {
		st = st()
	}
}

// lex an anything, switch to a more precise state when we know what
// we have.
func (l *lexer) lexStart() lexFn {
	r := l.peek()
	switch {
	case r == eof:
		return nil
	case numberStart(r):
		return l.lexNumber
	case identifierStart(r):
		return l.lexIdentifier
	case r == '/':
		return l.lexComment
	case strings.IndexRune(spaces, r) >= 0:
		return l.lexSpace
	default:
		return l.err("don't know how to lex %q", r)
	}
}

func (l *lexer) lexNumber() lexFn {
	l.accept("+-")
	if l.accept("0") {
		// Could be a radix prefix, with simpler parsing rules.
		switch l.next() {
		case eof:
			l.emit(tokInt)
			return nil
		case 'x':
			l.acceptRun("0123456789abcdefABCDEF_")
			l.emit(tokInt)
			return l.lexSpace
		case 'b':
			l.acceptRun("01_")
			l.emit(tokInt)
			return l.lexSpace
		case 'o':
			l.acceptRun("01234567_")
			l.emit(tokInt)
			return l.lexSpace
		}
	}
	// Full decimal/float.
	fl := false
	const digits = "0123456789_"
	l.acceptRun(digits)
	if l.accept(".") {
		fl = true
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun(digits)
	}
	if fl {
		l.emit(tokFloat)
	} else {
		l.emit(tokInt)
	}
	return l.lexSpace
}

const newline = "\x0D\x0A\x85\x0C\u2028\u2029"

func (l *lexer) lexComment() lexFn {
	if l.next() != '/' {
		panic("how did we end up in lexComment without a slash?!")
	}

	r := l.next()
	switch r {
	case '/':
		l.until(newline)
		return l.lexNewline
	case '*':
		for depth := 1; depth > 0; {
			l.until("*/")
			switch l.next() {
			case eof:
				return l.err("EOF during multiline comment")
			case '*':
				if l.peek() != '/' {
					continue
				}
				l.next()
				depth--
			case '/':
				if l.peek() != '*' {
					continue
				}
				l.next()
				depth++
			}
		}
		return l.lexSpace
	case '-':
		l.emit(tokIgnoreNode)
		return l.lexSpace
	default:
		return l.err("unknown kind of comment \"/%s\"", r)
	}
}

func (l *lexer) lexIdentifier() lexFn {
	return nil
}

func (l *lexer) lexSpace() lexFn {
	l.acceptRun(spaces)
	switch l.peek() {
	case eof:
		l.emit(tokSpace)
		return nil
	case '\\':
		l.next() // TODO: check if there _must_ be at least one space, currently accept zero.
		l.acceptRun(spaces)
		if l.peek() == '/' {
			l.next()
			if r := l.peek(); r != '/' {
				return l.err("unexpected rune %q in newline continuation", r)
			}
			l.until(newline)
		}
		if !l.acceptNewline() {
			return l.err("unexpected rune %q in newline continuation", l.peek())
		}
		return l.lexSpace
	default:
		l.emit(tokSpace)
		return l.lexStart
	}
}

func (l *lexer) lexNewline() lexFn {
	if !l.acceptNewline() {
		l.err("tried to lex newline when not at newline")
	}
	l.emit(tokNewline)
	return l.lexSpace
}
