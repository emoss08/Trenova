package apichain

import (
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
)

const (
	scopeFixture = "fixture"
	scopeRun     = "run"

	RunUnique       = "unique"
	RunNow          = "now"
	RunOrganization = "organization"
	RunBusinessUnit = "businessUnit"
	RunUser         = "user"

	StepRequest  = "request"
	StepResponse = "response"
)

var runValues = []string{RunUnique, RunNow, RunOrganization, RunBusinessUnit, RunUser}

var (
	refPattern     = regexp.MustCompile(`\$\{([^}]+)\}`)
	nowOffsetRegex = regexp.MustCompile(`^now([+-]\d+)$`)
)

func runOffset(name string) (int64, bool) {
	match := nowOffsetRegex.FindStringSubmatch(name)
	if match == nil {
		return 0, false
	}
	offset, err := strconv.ParseInt(match[1], 10, 64)

	return offset, err == nil
}

type StepValues struct {
	Request  any
	Response any
	Exports  map[string]any
}

type Scope struct {
	Fixtures map[string]any
	Run      map[string]any
	Steps    map[string]StepValues
}

func NewScope() *Scope {
	return &Scope{
		Fixtures: map[string]any{},
		Run:      map[string]any{},
		Steps:    map[string]StepValues{},
	}
}

func (s *Scope) Clone() *Scope {
	return &Scope{
		Fixtures: maps.Clone(s.Fixtures),
		Run:      maps.Clone(s.Run),
		Steps:    maps.Clone(s.Steps),
	}
}

func References(value any) []string {
	var out []string
	walkStrings(value, func(text string) {
		for _, match := range refPattern.FindAllStringSubmatch(text, -1) {
			out = append(out, match[1])
		}
	})

	return out
}

func walkStrings(value any, visit func(string)) {
	switch typed := value.(type) {
	case string:
		visit(typed)
	case map[string]any:
		for _, item := range typed {
			walkStrings(item, visit)
		}
	case map[string]string:
		for _, item := range typed {
			visit(item)
		}
	case []any:
		for _, item := range typed {
			walkStrings(item, visit)
		}
	}
}

func (s *Scope) Resolve(value any) (any, error) {
	switch typed := value.(type) {
	case string:
		return s.resolveString(typed)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			resolved, err := s.Resolve(item)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			resolved, err := s.Resolve(item)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}

func (s *Scope) ResolveString(text string) (string, error) {
	resolved, err := s.resolveString(text)
	if err != nil {
		return "", err
	}

	return textOf(resolved), nil
}

func (s *Scope) resolveString(text string) (any, error) {
	matches := refPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}
	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(text) {
		return s.Lookup(text[matches[0][2]:matches[0][3]])
	}

	var out strings.Builder
	last := 0
	for _, match := range matches {
		out.WriteString(text[last:match[0]])
		value, err := s.Lookup(text[match[2]:match[3]])
		if err != nil {
			return nil, err
		}
		out.WriteString(textOf(value))
		last = match[1]
	}
	out.WriteString(text[last:])

	return out.String(), nil
}

func (s *Scope) Lookup(ref string) (any, error) {
	scope, rest, _ := strings.Cut(ref, ".")
	switch scope {
	case scopeFixture:
		value, ok := s.Fixtures[rest]
		if !ok {
			return nil, fmt.Errorf("${%s}: no such fixture", ref)
		}
		return value, nil
	case scopeRun:
		if offset, isOffset := runOffset(rest); isOffset {
			now, ok := s.Run[RunNow].(int64)
			if !ok {
				return nil, fmt.Errorf("${%s}: the run has no clock", ref)
			}
			return now + offset, nil
		}
		value, ok := s.Run[rest]
		if !ok {
			return nil, fmt.Errorf("${%s}: no such run value", ref)
		}
		return value, nil
	}

	step, ok := s.Steps[scope]
	if !ok {
		return nil, fmt.Errorf("${%s}: step %q has not run", ref, scope)
	}
	head, path, _ := strings.Cut(rest, ".")
	var root any
	switch head {
	case StepRequest:
		root = step.Request
	case StepResponse:
		root = step.Response
	default:
		exported, found := step.Exports[head]
		if !found {
			return nil, fmt.Errorf(
				"${%s}: step %q exports no %q (use %s.%s.<path> or %s.%s.<path>)",
				ref, scope, head, scope, StepResponse, scope, StepRequest,
			)
		}
		root = exported
	}
	if path == "" {
		return root, nil
	}
	value, found := Get(root, path)
	if !found {
		return nil, fmt.Errorf("${%s}: nothing at %q", ref, path)
	}

	return value, nil
}

func Get(root any, path string) (any, bool) {
	if path == "" {
		return root, true
	}
	current := root
	for segment := range strings.SplitSeq(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}

	return current, true
}

func textOf(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}
