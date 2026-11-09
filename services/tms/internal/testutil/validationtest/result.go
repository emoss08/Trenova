package validationtest

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
)

type Validatable interface {
	Validate(multiErr *errortypes.MultiError)
}

type Want struct {
	Field string
	Code  errortypes.ErrorCode
}

func At(field string, code errortypes.ErrorCode) Want {
	return Want{Field: field, Code: code}
}

func (w Want) String() string {
	return describePlace(fieldpath.Parse(w.Field)) + " " + string(w.Code)
}

type Result struct {
	t          testing.TB
	errors     []*errortypes.Error
	advisories []*errortypes.AdvisoryError
}

func Outcome(t testing.TB, err error) *Result {
	t.Helper()

	if err == nil {
		return &Result{t: t}
	}
	multiErr, ok := errortypes.RefusalOf(err)
	if !ok {
		t.Fatalf("want a refusal or no error, got an error that refuses no field: %v", err)
		return nil
	}

	return newResult(t, multiErr)
}

func Refused(t testing.TB, err error) *Result {
	t.Helper()

	if err == nil {
		t.Fatalf("want a refusal, got no error")
		return nil
	}
	result := Outcome(t, err)
	if len(result.errors) == 0 {
		t.Fatalf("want a refusal, got one that names no field: %v", err)
	}

	return result
}

func Report(t testing.TB, multiErr *errortypes.MultiError) *Result {
	t.Helper()

	if multiErr == nil {
		return &Result{t: t}
	}
	if internal := multiErr.Internal(); internal != nil {
		t.Fatalf("validation could not complete: %v", internal)
		return nil
	}
	err := multiErr.Err()
	if err == nil {
		return &Result{t: t}
	}

	return Outcome(t, err)
}

func Validate(t testing.TB, entity Validatable) *Result {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	return Report(t, multiErr)
}

func newResult(t testing.TB, multiErr *errortypes.MultiError) *Result {
	return &Result{
		t:          t,
		errors:     slices.Clone(multiErr.Errors),
		advisories: slices.Clone(multiErr.Advisories),
	}
}

func (r *Result) Errors() []*errortypes.Error {
	return slices.Clone(r.errors)
}

func (r *Result) Advisories() []*errortypes.AdvisoryError {
	return slices.Clone(r.advisories)
}

func (r *Result) Fields() []string {
	fields := make([]string, 0, len(r.errors))
	for _, err := range r.errors {
		fields = append(fields, err.Field())
	}

	return fields
}

func (r *Result) Codes(field string) []errortypes.ErrorCode {
	path := fieldpath.Parse(field)
	codes := make([]errortypes.ErrorCode, 0, 1)
	for _, err := range r.errors {
		if err.Path.Equal(path) {
			codes = append(codes, err.Code)
		}
	}

	return codes
}

func (r *Result) Messages(field string) []string {
	path := fieldpath.Parse(field)
	messages := make([]string, 0, 1)
	for _, err := range r.errors {
		if err.Path.Equal(path) {
			messages = append(messages, err.Error())
		}
	}

	return messages
}

func (r *Result) Valid() bool {
	r.t.Helper()

	if len(r.errors) == 0 {
		return true
	}
	r.t.Errorf("want no refusal, got:\n%s", r.describe())

	return false
}

func (r *Result) Refuses(field string, code errortypes.ErrorCode) *errortypes.Error {
	r.t.Helper()

	want := At(field, code)
	path := fieldpath.Parse(field)
	for _, err := range r.errors {
		if err.Path.Equal(path) && err.Code == code {
			return err
		}
	}
	r.t.Fatalf("want a refusal at %s, got:\n%s", want, r.describe())

	return nil
}

func (r *Result) Allows(field string) bool {
	r.t.Helper()

	path := fieldpath.Parse(field)
	var refused []string
	for _, err := range r.errors {
		if err.Path.HasPrefix(path) {
			refused = append(refused, describeError(err))
		}
	}
	if len(refused) == 0 {
		return true
	}
	r.t.Errorf(
		"want nothing refused at or under %s, got:\n%s",
		describePlace(path),
		bulleted(refused),
	)

	return false
}

func (r *Result) Exactly(want ...Want) bool {
	r.t.Helper()

	unmatched := slices.Clone(r.errors)
	var missing []string
	for _, expected := range want {
		path := fieldpath.Parse(expected.Field)
		index := slices.IndexFunc(unmatched, func(err *errortypes.Error) bool {
			return err.Path.Equal(path) && err.Code == expected.Code
		})
		if index < 0 {
			missing = append(missing, expected.String())
			continue
		}
		unmatched = slices.Delete(unmatched, index, index+1)
	}
	if len(missing) == 0 && len(unmatched) == 0 {
		return true
	}

	var report strings.Builder
	if len(missing) > 0 {
		report.WriteString("missing:\n" + bulleted(missing))
	}
	if len(unmatched) > 0 {
		unexpected := make([]string, 0, len(unmatched))
		for _, err := range unmatched {
			unexpected = append(unexpected, describeError(err))
		}
		report.WriteString("unexpected:\n" + bulleted(unexpected))
	}
	r.t.Errorf("refusals differ from the ones wanted\n%s", report.String())

	return false
}

func (r *Result) Advises(field string, code errortypes.ErrorCode) *errortypes.AdvisoryError {
	r.t.Helper()

	path := fieldpath.Parse(field)
	for _, advisory := range r.advisories {
		if advisory.Path.Equal(path) && advisory.Code == code {
			return advisory
		}
	}

	described := make([]string, 0, len(r.advisories))
	for _, advisory := range r.advisories {
		described = append(described, fmt.Sprintf(
			"%s %s: %s",
			describePlace(advisory.Path),
			advisory.Code,
			advisory.Error(),
		))
	}
	r.t.Fatalf(
		"want an advisory at %s, got:\n%s",
		At(field, code),
		bulletedOrNone(described),
	)

	return nil
}

func (r *Result) describe() string {
	described := make([]string, 0, len(r.errors))
	for _, err := range r.errors {
		described = append(described, describeError(err))
	}

	return bulletedOrNone(described)
}

func describeError(err *errortypes.Error) string {
	return fmt.Sprintf("%s %s: %s", describePlace(err.Path), err.Code, err.Error())
}

func describePlace(path fieldpath.Path) string {
	if path.IsZero() {
		return "(the record)"
	}

	return path.String()
}

func bulletedOrNone(lines []string) string {
	if len(lines) == 0 {
		return "  (none)\n"
	}

	return bulleted(lines)
}

func bulleted(lines []string) string {
	var out strings.Builder
	for _, line := range lines {
		out.WriteString("  - " + line + "\n")
	}

	return out.String()
}
