package oracle

import (
	"errors"
	"fmt"
	"strconv"
)

type syntaxNote struct {
	kind      byte
	positions []int
	text      string
}

type noteBreach struct {
	pos  int
	code string
	msg  string
}

func parseSyntaxNote(text string) (syntaxNote, error) {
	if len(text) < 5 || len(text)%2 == 0 {
		return syntaxNote{}, fmt.Errorf("%q is not a syntax note like P0304", text)
	}
	switch text[0] {
	case 'P', 'R', 'E', 'C', 'L':
	default:
		return syntaxNote{}, fmt.Errorf("%q does not start with P, R, E, C or L", text)
	}
	n := syntaxNote{kind: text[0], text: text}
	seen := make(map[int]bool)
	for i := 1; i < len(text); i += 2 {
		p, err := strconv.Atoi(text[i : i+2])
		if err != nil || p < 1 {
			return syntaxNote{}, fmt.Errorf("%q has a position that is not two digits", text)
		}
		if seen[p] {
			return syntaxNote{}, fmt.Errorf("%q names position %02d twice", text, p)
		}
		seen[p] = true
		n.positions = append(n.positions, p)
	}
	if n.kind == 'L' && len(n.positions) < 3 {
		return syntaxNote{}, errors.New("a list conditional names at least three positions")
	}
	return n, nil
}

func (n syntaxNote) check(present func(int) bool) []noteBreach {
	var out []noteBreach
	count := 0
	for _, p := range n.positions {
		if present(p) {
			count++
		}
	}
	first := n.positions[0]
	switch n.kind {
	case 'P':
		if count == 0 || count == len(n.positions) {
			return nil
		}
		for _, p := range n.positions {
			if !present(p) {
				out = append(out, n.missing(p, "is required when any of them is present"))
			}
		}
	case 'R':
		if count == 0 {
			out = append(out, n.missing(first, "at least one of them is required"))
		}
	case 'E':
		if count <= 1 {
			return nil
		}
		seen := 0
		for _, p := range n.positions {
			if present(p) {
				seen++
				if seen == 2 {
					out = append(out, noteBreach{
						pos:  p,
						code: codeExclusionViolated,
						msg:  "only one of them may be present",
					})
				}
			}
		}
	case 'C':
		if !present(first) {
			return nil
		}
		for _, p := range n.positions[1:] {
			if !present(p) {
				out = append(
					out,
					n.missing(p, fmt.Sprintf("is required when %02d is present", first)),
				)
			}
		}
	case 'L':
		if present(first) && count == 1 {
			out = append(out, n.missing(
				n.positions[1],
				fmt.Sprintf("one of them is required when %02d is present", first),
			))
		}
	}
	return out
}

func (n syntaxNote) missing(pos int, msg string) noteBreach {
	return noteBreach{pos: pos, code: codeConditionalMissing, msg: msg}
}
