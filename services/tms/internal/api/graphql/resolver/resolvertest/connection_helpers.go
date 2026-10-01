package resolvertest

import (
	"github.com/emoss08/trenova/shared/pulid"
)

type TestCursorValueProvider struct {
	Values [][]any
}

func (p TestCursorValueProvider) CursorValuesAt(index int) ([]any, bool) {
	if index < 0 || index >= len(p.Values) {
		return nil, false
	}

	return p.Values[index], true
}

type TestConnectionEdge struct {
	Node   string
	Cursor string
}

type TestCursorEntity struct {
	Node      string
	ID        pulid.ID
	CreatedAt int64
}

func (e TestCursorEntity) GetID() pulid.ID {
	return e.ID
}

func (e TestCursorEntity) GetCreatedAt() int64 {
	return e.CreatedAt
}
