// Package statdesc parses stat description files (Metadata/StatDescriptions/*.txt) into
// pbt-engine statdesc scopes. Only English text is kept.
package statdesc

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/xaroth/pbt-engine/statdesc"

	"github.com/xaroth/pbt-extract/ot"
)

// Load parses the file at path and the files it includes. The scope for path comes first.
func Load(r ot.Reader, path string) ([]statdesc.Scope, error) {
	var out []statdesc.Scope
	seen := map[string]bool{}
	for p := path; p != ""; {
		if seen[p] {
			return nil, fmt.Errorf("%s: include cycle", p)
		}
		seen[p] = true
		b, err := r.ReadFile(p)
		if err != nil {
			return nil, err
		}
		sc, err := Parse(p, b)
		if err != nil {
			return nil, err
		}
		out = append(out, sc)
		p = sc.Parent
	}
	return out, nil
}

// Parse parses one UTF-16LE stat description file. name becomes the scope name; an include
// becomes its parent.
func Parse(name string, b []byte) (statdesc.Scope, error) {
	if len(b)%2 != 0 {
		return statdesc.Scope{}, fmt.Errorf("%s: odd length, not UTF-16", name)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	text := strings.TrimPrefix(string(utf16.Decode(u)), "\ufeff")
	p := &parser{lines: strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")}
	sc, err := p.scope(name)
	if err != nil {
		return statdesc.Scope{}, fmt.Errorf("%s line %d: %w", name, p.pos, err)
	}
	return sc, nil
}

type parser struct {
	lines []string
	pos   int // 1-based number of the last line read
}

// next returns the next non-blank line's fields, or nil at the end.
func (p *parser) next() []string {
	for p.pos < len(p.lines) {
		p.pos++
		if f := strings.Fields(p.lines[p.pos-1]); len(f) > 0 {
			return f
		}
	}
	return nil
}

func (p *parser) peek() []string {
	for i := p.pos; i < len(p.lines); i++ {
		if f := strings.Fields(p.lines[i]); len(f) > 0 {
			return f
		}
	}
	return nil
}

func (p *parser) scope(name string) (statdesc.Scope, error) {
	sc := statdesc.Scope{Name: name}
	for f := p.next(); f != nil; f = p.next() {
		switch f[0] {
		case "include":
			if sc.Parent != "" {
				return sc, errors.New("second include")
			}
			inc, err := unquote(strings.Join(f[1:], " "))
			if err != nil {
				return sc, err
			}
			sc.Parent = inc
		case "has_identifiers", "no_identifiers":
			// Marks whether description headers carry names; names are not used yet.
		case "no_description":
			if len(f) != 2 {
				return sc, errors.New("no_description wants one stat")
			}
			sc.Descriptions = append(sc.Descriptions, statdesc.Description{Stats: f[1:]})
		case "description":
			d, err := p.description()
			if err != nil {
				return sc, err
			}
			sc.Descriptions = append(sc.Descriptions, d)
		default:
			return sc, fmt.Errorf("unexpected %q", f[0])
		}
	}
	return sc, nil
}

func (p *parser) description() (statdesc.Description, error) {
	var d statdesc.Description
	f := p.next()
	n, err := count(f)
	if err != nil || len(f) != n+1 {
		return d, fmt.Errorf("stat line %q", strings.Join(f, " "))
	}
	d.Stats = f[1:]

	variants, err := p.block(n, true)
	if err != nil {
		return d, err
	}
	d.Variants = variants
	// Other languages follow as lang "X" blocks. They are skipped unparsed; they contain
	// typos the English lines do not.
	for f := p.peek(); f != nil && f[0] == "lang"; f = p.peek() {
		p.next()
		if _, err := p.block(n, false); err != nil {
			return d, err
		}
	}
	return d, nil
}

// block reads a variant count line and that many variant lines for n stats. Without parse the
// lines are only skipped.
func (p *parser) block(n int, parse bool) ([]statdesc.Variant, error) {
	f := p.next()
	k, err := count(f)
	if err != nil || len(f) != 1 {
		return nil, fmt.Errorf("variant count %q", strings.Join(f, " "))
	}
	out := make([]statdesc.Variant, k)
	for i := range out {
		if p.pos >= len(p.lines) {
			return nil, errors.New("unexpected end of file")
		}
		p.next()
		if !parse {
			continue
		}
		if out[i], err = variant(p.lines[p.pos-1], n); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func variant(line string, n int) (statdesc.Variant, error) {
	var v statdesc.Variant
	open := strings.IndexByte(line, '"')
	end := strings.LastIndexByte(line, '"')
	if open < 0 || end == open {
		return v, fmt.Errorf("variant without text: %q", line)
	}
	v.Text = line[open+1 : end]

	lims := strings.Fields(line[:open])
	if len(lims) != n {
		return v, fmt.Errorf("%d limits for %d stats: %q", len(lims), n, line)
	}
	for _, l := range lims {
		lim, err := limit(l)
		if err != nil {
			return v, err
		}
		v.Limits = append(v.Limits, lim)
	}

	hs := strings.Fields(line[end+1:])
	for i := 0; i < len(hs); i++ {
		h := statdesc.Handler{Name: hs[i]}
		if h.Name == "reminderstring" {
			i++ // reminder text id, not displayed
		} else if i+1 < len(hs) {
			if idx, err := strconv.Atoi(hs[i+1]); err == nil {
				if idx < 1 || idx > n {
					return v, fmt.Errorf("handler %s stat %d out of range: %q", h.Name, idx, line)
				}
				h.Stat = idx - 1
				i++
			}
		}
		v.Handlers = append(v.Handlers, h)
	}
	return v, nil
}

func limit(s string) (statdesc.Limit, error) {
	if rest, ok := strings.CutPrefix(s, "!"); ok {
		x, err := strconv.ParseInt(rest, 10, 32)
		return statdesc.Limit{Min: int32(x), Max: int32(x), Not: true}, err
	}
	lo, hi, rng := strings.Cut(s, "|")
	if !rng {
		hi = lo
	}
	l := statdesc.Limit{Min: math.MinInt32, Max: math.MaxInt32}
	for _, b := range []struct {
		s   string
		dst *int32
	}{{lo, &l.Min}, {hi, &l.Max}} {
		if b.s == "#" {
			continue
		}
		x, err := strconv.ParseInt(b.s, 10, 32)
		if err != nil {
			return l, fmt.Errorf("limit %q: %w", s, err)
		}
		*b.dst = int32(x)
	}
	return l, nil
}

func count(f []string) (int, error) {
	if len(f) == 0 {
		return 0, errors.New("unexpected end of file")
	}
	n, err := strconv.Atoi(f[0])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("bad count %q", f[0])
	}
	return n, nil
}

func unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("want quoted string, got %q", s)
	}
	return s[1 : len(s)-1], nil
}
