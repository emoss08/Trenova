package tarifftypes

import (
	"bytes"

	"github.com/emoss08/trenova/pkg/rules"
)

func ImportsEqual(a, b []Import) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ValuesEqual(a, b rules.RawJSON) bool {
	return bytes.Equal(compact(a), compact(b))
}

func DeclsEqual(a, b []InputDecl) bool {
	if len(a) != len(b) {
		return false
	}
	byName := make(map[string]*InputDecl, len(a))
	for i := range a {
		byName[a[i].Name] = &a[i]
	}
	for i := range b {
		other, ok := byName[b[i].Name]
		if !ok ||
			other.Required != b[i].Required ||
			other.Type != b[i].Type ||
			!ValuesEqual(other.Descriptor, b[i].Descriptor) ||
			!ValuesEqual(other.Default, b[i].Default) ||
			!ConstraintsEqual(other.Constraint, b[i].Constraint) {
			return false
		}
	}
	return true
}

func ConstraintsEqual(a, b *rules.TariffConstraint) bool {
	if a == nil || b == nil {
		return a == b
	}
	if (a.Range == nil) != (b.Range == nil) || len(a.OneOf) != len(b.OneOf) {
		return false
	}
	if a.Range != nil && (a.Range.MaxInclusive != b.Range.MaxInclusive ||
		!sameBound(a.Range.Min, b.Range.Min) ||
		!sameBound(a.Range.Max, b.Range.Max)) {
		return false
	}
	for i := range a.OneOf {
		if !EqualValues(a.OneOf[i], b.OneOf[i]) {
			return false
		}
	}
	return true
}

func sameBound(a, b rules.RawJSON) bool {
	if a.IsNone() || b.IsNone() {
		return a.IsNone() == b.IsNone()
	}
	return EqualValues(a, b)
}
