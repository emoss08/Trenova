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
			!ValuesEqual(other.Default, b[i].Default) {
			return false
		}
	}
	return true
}
