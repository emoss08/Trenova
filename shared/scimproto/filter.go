package scimproto

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxFilterLength     = 1024
	maxFilterConditions = 8
)

type Condition struct {
	Attribute string
	Value     string
}

type Filter struct {
	Conditions []Condition
}

func (f *Filter) Value(attribute string) (string, bool) {
	if f == nil {
		return "", false
	}
	for _, condition := range f.Conditions {
		if condition.Attribute == attribute {
			return condition.Value, true
		}
	}

	return "", false
}

func (f *Filter) Empty() bool {
	return f == nil || len(f.Conditions) == 0
}

func ParseFilter(raw string, allowed ...string) (*Filter, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return &Filter{}, nil
	}
	if len(raw) > maxFilterLength || !utf8.ValidString(raw) {
		return nil, InvalidFilter("The filter is too long or not valid text")
	}

	tokens, err := lexFilter(raw)
	if err != nil {
		return nil, err
	}

	p := &filterParser{tokens: tokens}
	conditions, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, InvalidFilter("The filter has text after its last expression")
	}
	if len(conditions) > maxFilterConditions {
		return nil, InvalidFilter("The filter has too many conditions")
	}

	permitted := make(map[string]struct{}, len(allowed))
	for _, attribute := range allowed {
		permitted[attribute] = struct{}{}
	}
	for _, condition := range conditions {
		if _, ok := permitted[condition.Attribute]; !ok {
			return nil, InvalidFilter("Filtering by " + condition.Attribute + " is not supported")
		}
	}

	return &Filter{Conditions: conditions}, nil
}

type tokenKind int

const (
	tokenWord tokenKind = iota
	tokenString
	tokenOpenParen
	tokenCloseParen
	tokenOpenBracket
	tokenCloseBracket
)

type filterToken struct {
	kind tokenKind
	text string
}

func lexFilter(raw string) ([]filterToken, error) {
	tokens := make([]filterToken, 0, 8)
	for i := 0; i < len(raw); {
		c := raw[i]
		switch {
		case c == ' ' || c == '\t':
			i++
		case c == '(':
			tokens = append(tokens, filterToken{kind: tokenOpenParen})
			i++
		case c == ')':
			tokens = append(tokens, filterToken{kind: tokenCloseParen})
			i++
		case c == '[':
			tokens = append(tokens, filterToken{kind: tokenOpenBracket})
			i++
		case c == ']':
			tokens = append(tokens, filterToken{kind: tokenCloseBracket})
			i++
		case c == '"':
			end, value, err := lexString(raw, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, filterToken{kind: tokenString, text: value})
			i = end
		default:
			start := i
			for i < len(raw) && !strings.ContainsRune(" \t()[]\"", rune(raw[i])) {
				i++
			}
			tokens = append(tokens, filterToken{kind: tokenWord, text: raw[start:i]})
		}
	}

	return tokens, nil
}

func lexString(raw string, start int) (end int, value string, err error) {
	for i := start + 1; i < len(raw); i++ {
		switch raw[i] {
		case '\\':
			i++
		case '"':
			unquoted, unquoteErr := strconv.Unquote(raw[start : i+1])
			if unquoteErr != nil {
				return 0, "", InvalidFilter("The filter has a string that is not valid")
			}
			return i + 1, unquoted, nil
		}
	}

	return 0, "", InvalidFilter("The filter has a string that is never closed")
}

type filterParser struct {
	tokens []filterToken
	pos    int
}

func (p *filterParser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *filterParser) peek() (filterToken, bool) {
	if p.done() {
		return filterToken{}, false
	}

	return p.tokens[p.pos], true
}

func (p *filterParser) next() (filterToken, bool) {
	token, ok := p.peek()
	if ok {
		p.pos++
	}

	return token, ok
}

func (p *filterParser) peekWord(word string) bool {
	token, ok := p.peek()
	return ok && token.kind == tokenWord && strings.EqualFold(token.text, word)
}

func (p *filterParser) expect(kind tokenKind, what string) (filterToken, error) {
	token, ok := p.next()
	if !ok || token.kind != kind {
		return filterToken{}, InvalidFilter("The filter is missing " + what)
	}

	return token, nil
}

func (p *filterParser) parseAnd() ([]Condition, error) {
	conditions, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.peekWord("and") {
		p.pos++
		more, termErr := p.parseTerm()
		if termErr != nil {
			return nil, termErr
		}
		conditions = append(conditions, more...)
	}

	if p.peekWord("or") {
		return nil, InvalidFilter("Filters joined with or are not supported")
	}

	return conditions, nil
}

func (p *filterParser) parseTerm() ([]Condition, error) {
	if token, ok := p.peek(); ok && token.kind == tokenOpenParen {
		p.pos++
		conditions, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(tokenCloseParen, "a closing parenthesis"); err != nil {
			return nil, err
		}
		return conditions, nil
	}
	if p.peekWord("not") {
		return nil, InvalidFilter("Filters with not are not supported")
	}

	path, err := p.expect(tokenWord, "an attribute")
	if err != nil {
		return nil, err
	}

	if token, ok := p.peek(); ok && token.kind == tokenOpenBracket {
		return p.parseValuePath(path.text)
	}

	return p.parseComparison(path.text)
}

func (p *filterParser) parseValuePath(path string) ([]Condition, error) {
	p.pos++
	inner, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(tokenCloseBracket, "a closing bracket"); err != nil {
		return nil, err
	}

	if token, ok := p.peek(); ok && token.kind == tokenWord && strings.HasPrefix(token.text, ".") {
		p.pos++
		return p.parseComparison(path + token.text)
	}

	for _, condition := range inner {
		if condition.Attribute == "value" {
			return []Condition{
				{Attribute: NormalizeAttribute(path) + ".value", Value: condition.Value},
			}, nil
		}
	}

	return nil, InvalidFilter("A filter on " + path + " must compare its value")
}

func (p *filterParser) parseComparison(path string) ([]Condition, error) {
	operator, err := p.expect(tokenWord, "a comparison operator")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(operator.text, "eq") {
		return nil, InvalidFilter("Only the eq operator is supported")
	}

	value, ok := p.next()
	if !ok {
		return nil, InvalidFilter("The filter is missing a value")
	}

	var literal string
	switch value.kind {
	case tokenString:
		literal = value.text
	case tokenWord:
		literal = strings.ToLower(value.text)
		if literal != "true" && literal != "false" {
			if _, numErr := strconv.ParseFloat(literal, 64); numErr != nil {
				return nil, InvalidFilter("The filter value " + value.text + " is not valid")
			}
		}
	default:
		return nil, InvalidFilter("The filter is missing a value")
	}

	attribute := NormalizeAttribute(path)
	if attribute == "members" || attribute == "emails" {
		attribute += ".value"
	}

	return []Condition{{Attribute: attribute, Value: literal}}, nil
}

func NormalizeAttribute(path string) string {
	return strings.ToLower(stripSchemaPrefix(path))
}

func stripSchemaPrefix(path string) string {
	trimmed := strings.TrimSpace(path)
	lower := strings.ToLower(trimmed)
	for _, schema := range []string{SchemaUser, SchemaGroup} {
		prefix := strings.ToLower(schema) + ":"
		if strings.HasPrefix(lower, prefix) {
			return trimmed[len(prefix):]
		}
	}

	return trimmed
}
