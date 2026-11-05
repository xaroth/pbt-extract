// Package ot parses Path of Exile object templates (.ot files).
package ot

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

// File is a parsed object template.
type File struct {
	// Extends is the parent object path, or "nothing".
	Extends string
	Blocks  []Block
}

// Block is a named brace block such as Stats { ... }.
type Block struct {
	Name   string
	Fields []Field
	Blocks []Block
}

// Field is a key = value line. Quoted values are unquoted and script values ({ ... }) hold the
// text between the braces.
type Field struct {
	Key   string
	Value string
}

// Block returns the first top-level block with the given name.
func (f *File) Block(name string) (Block, bool) {
	for _, b := range f.Blocks {
		if b.Name == name {
			return b, true
		}
	}
	return Block{}, false
}

// Parse parses a UTF-16LE object template.
func Parse(b []byte) (*File, error) {
	if len(b)%2 != 0 {
		return nil, errors.New("odd length, not UTF-16")
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	src := strings.TrimPrefix(string(utf16.Decode(u)), "\ufeff")
	p := parser{src: src}
	f, err := p.file()
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", p.line(), err)
	}
	return f, nil
}

type parser struct {
	src string
	pos int
}

func (p *parser) line() int { return 1 + strings.Count(p.src[:p.pos], "\n") }

func (p *parser) file() (*File, error) {
	f := &File{}
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}
		switch tok {
		case "":
			return f, nil
		case "version":
			if _, err := p.next(); err != nil {
				return nil, err
			}
		case "abstract":
		case "extends":
			v, err := p.next()
			if err != nil {
				return nil, err
			}
			f.Extends, err = unquote(v)
			if err != nil {
				return nil, err
			}
		default:
			b, err := p.block(tok)
			if err != nil {
				return nil, err
			}
			f.Blocks = append(f.Blocks, b)
		}
	}
}

func (p *parser) block(name string) (Block, error) {
	if !isWord(name) {
		return Block{}, fmt.Errorf("unexpected %q", name)
	}
	tok, err := p.next()
	if err != nil {
		return Block{}, err
	}
	if tok != "{" {
		return Block{}, fmt.Errorf("block %s: want {, got %q", name, tok)
	}
	b := Block{Name: name}
	for {
		tok, err := p.next()
		if err != nil {
			return Block{}, err
		}
		switch tok {
		case "":
			return Block{}, fmt.Errorf("block %s: unexpected end of file", name)
		case "}":
			return b, nil
		}
		if !isWord(tok) {
			return Block{}, fmt.Errorf("block %s: unexpected %q", name, tok)
		}
		p.skipSpace()
		if !strings.HasPrefix(p.src[p.pos:], "=") {
			sub, err := p.block(tok)
			if err != nil {
				return Block{}, err
			}
			b.Blocks = append(b.Blocks, sub)
			continue
		}
		p.pos++
		v, err := p.value()
		if err != nil {
			return Block{}, fmt.Errorf("%s.%s: %w", name, tok, err)
		}
		b.Fields = append(b.Fields, Field{Key: tok, Value: v})
	}
}

func (p *parser) value() (string, error) {
	p.skipSpace()
	if !strings.HasPrefix(p.src[p.pos:], "{") {
		tok, err := p.next()
		if err != nil {
			return "", err
		}
		if tok == "" || tok == "{" || tok == "}" {
			return "", fmt.Errorf("missing value, got %q", tok)
		}
		if strings.HasPrefix(tok, `"`) {
			return unquote(tok)
		}
		return tok, nil
	}
	start, depth := p.pos+1, 0
	for p.pos < len(p.src) {
		if strings.HasPrefix(p.src[p.pos:], "//") {
			p.skipSpace()
			continue
		}
		switch c := p.src[p.pos]; c {
		case '"', '\'':
			end := strings.IndexByte(p.src[p.pos+1:], c)
			if end < 0 {
				return "", errors.New("unterminated string in script")
			}
			p.pos += end + 2
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				p.pos++
				return strings.TrimSpace(p.src[start : p.pos-1]), nil
			}
		}
		p.pos++
	}
	return "", errors.New("unterminated script")
}

// next returns the next token: "{", "}", "=", a quoted string with its quotes, a word, or ""
// at the end of input.
func (p *parser) next() (string, error) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return "", nil
	}
	start := p.pos
	switch c := p.src[p.pos]; c {
	case '{', '}', '=':
		p.pos++
	case '"':
		end := strings.IndexByte(p.src[p.pos+1:], '"')
		if end < 0 {
			return "", errors.New("unterminated string")
		}
		p.pos += end + 2
	default:
		for p.pos < len(p.src) && !strings.ContainsRune(" \t\r\n{}=\"", rune(p.src[p.pos])) &&
			!strings.HasPrefix(p.src[p.pos:], "//") {
			p.pos++
		}
	}
	return p.src[start:p.pos], nil
}

func (p *parser) skipSpace() {
	for p.pos < len(p.src) {
		switch {
		case strings.ContainsRune(" \t\r\n", rune(p.src[p.pos])):
			p.pos++
		case strings.HasPrefix(p.src[p.pos:], "//"):
			end := strings.IndexByte(p.src[p.pos:], '\n')
			if end < 0 {
				p.pos = len(p.src)
				return
			}
			p.pos += end
		default:
			return
		}
	}
}

func unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("want quoted string, got %q", s)
	}
	return s[1 : len(s)-1], nil
}

func isWord(s string) bool {
	return s != "" && !strings.ContainsAny(s[:1], `{}="`)
}
