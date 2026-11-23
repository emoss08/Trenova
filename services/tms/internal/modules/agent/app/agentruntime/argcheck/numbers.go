package argcheck

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxRecordNumberLength    = 100
	maxNumberCandidatesNamed = 3
)

type NumberRef struct {
	Path    string
	Number  string
	Kinds   []recordkind.Kind
	trail   []any
	refusal string
}

type NumberCandidate struct {
	Kind       recordkind.Kind
	ID         pulid.ID
	NumberName string
	Number     string
}

type NumberReading struct {
	Ref       NumberRef
	Candidate NumberCandidate
}

func (c Contract) Numbers() []NumberRef {
	return c.numbers
}

func (c Contract) ReadNumbers(candidates map[string][]NumberCandidate) (Contract, error) {
	if len(c.numbers) == 0 {
		return c, nil
	}
	problems := make([]argumentProblem, 0)
	read := make([]NumberReading, 0, len(c.numbers))
	args := c.Args
	for _, ref := range c.numbers {
		found := candidates[ref.Path]
		if len(found) != 1 {
			problems = append(problems, argumentProblem{Path: ref.Path, Message: unreadNumber(ref, found)})

			continue
		}
		args = replacedArgs(args, ref.trail, found[0].ID.String())
		read = append(read, NumberReading{Ref: ref, Candidate: found[0]})
	}
	if len(problems) > 0 {
		sort.Slice(problems, func(i, j int) bool { return problems[i].Path < problems[j].Path })

		return Contract{}, refusalOf(problems)
	}

	c.Args = args
	c.numbers = nil
	c.read = read

	return c, nil
}

func recordNumber(value string, check idCheck) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if len(check.kinds) == 0 || trimmed == "" || len(trimmed) > maxRecordNumberLength {
		return "", false
	}
	if strings.ContainsFunc(trimmed, unicode.IsSpace) || pulid.LooksLike(trimmed) {
		return "", false
	}
	if _, _, misCopied := pulid.MisCopied(trimmed); misCopied {
		return "", false
	}

	return trimmed, true
}

func unreadNumber(ref NumberRef, found []NumberCandidate) string {
	if len(found) <= 1 {
		return ref.refusal
	}

	names := make([]string, 0, min(len(found), maxNumberCandidatesNamed)+1)
	for _, candidate := range found[:min(len(found), maxNumberCandidatesNamed)] {
		names = append(names, fmt.Sprintf("the %s with %s %s",
			candidate.Kind.Noun(), candidate.NumberName, candidate.Number))
	}
	if len(found) > maxNumberCandidatesNamed {
		names = append(names, fmt.Sprintf("%d more", len(found)-maxNumberCandidatesNamed))
	}

	return fmt.Sprintf(
		"%q is not a record id, and more than one record carries it as its number: %s. "+
			"Ask the person which one they mean, then send that record's id.",
		ref.Number, stringutils.JoinAlternatives(names),
	)
}

func numbersNote(read []NumberReading) string {
	if len(read) == 0 {
		return ""
	}

	notes := make([]string, 0, min(len(read), maxCoercionsNoted))
	for _, reading := range read[:min(len(read), maxCoercionsNoted)] {
		notes = append(notes, fmt.Sprintf("%s %q is the %s with that %s, id %s",
			reading.Ref.Path, reading.Ref.Number, reading.Candidate.Kind.Noun(),
			reading.Candidate.NumberName, reading.Candidate.ID))
	}
	text := "Record numbers were read as the records they name: " + strings.Join(notes, "; ")
	if len(read) > maxCoercionsNoted {
		text += fmt.Sprintf("; and %d more", len(read)-maxCoercionsNoted)
	}

	return text + ". A number is not a record id: send the id from now on."
}

func replacedArgs(args map[string]any, trail []any, id string) map[string]any {
	if replaced, ok := replacedAt(args, trail, id).(map[string]any); ok {
		return replaced
	}

	return args
}

func replacedAt(value any, trail []any, id string) any {
	if len(trail) == 0 {
		return id
	}
	switch step := trail[0].(type) {
	case string:
		object, ok := value.(map[string]any)
		if !ok {
			return value
		}
		replaced := maps.Clone(object)
		replaced[step] = replacedAt(object[step], trail[1:], id)

		return replaced
	case int:
		list, ok := value.([]any)
		if !ok || step < 0 || step >= len(list) {
			return value
		}
		replaced := slices.Clone(list)
		replaced[step] = replacedAt(list[step], trail[1:], id)

		return replaced
	}

	return value
}
