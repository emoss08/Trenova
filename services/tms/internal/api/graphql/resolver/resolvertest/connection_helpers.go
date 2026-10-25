package resolvertest

import (
	"github.com/emoss08/trenova/shared/pulid"
)

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
