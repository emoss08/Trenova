package partner

import (
	"fmt"
	"strings"
)

type Problem struct {
	File    string
	Path    string
	Message string
}

func (p Problem) String() string {
	if p.Path == "" {
		return p.File + ": " + p.Message
	}
	return p.File + ": " + p.Path + ": " + p.Message
}

type Problems []Problem

func (p Problems) Error() string {
	lines := make([]string, 0, len(p))
	for _, problem := range p {
		lines = append(lines, problem.String())
	}
	return strings.Join(lines, "\n")
}

type problems struct {
	file string
	list Problems
}

func (p *problems) add(path, format string, args ...any) {
	p.list = append(
		p.list,
		Problem{File: p.file, Path: path, Message: fmt.Sprintf(format, args...)},
	)
}

func (p *problems) required(path, value string) {
	if strings.TrimSpace(value) == "" {
		p.add(path, "is required")
	}
}

func (p *problems) oneOf(path, value string, allowed ...string) {
	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}
	p.add(path, "%q is not one of %s", value, strings.Join(allowed, ", "))
}

func (p *problems) err() error {
	if len(p.list) == 0 {
		return nil
	}
	return p.list
}
