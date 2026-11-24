package board

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

type Measure string

type MeasureKey struct {
	Kind    recordkind.Kind
	Measure Measure
}

func (m Measure) Spec(kind recordkind.Kind) (boardcatalog.Measure, bool) {
	if m == "" {
		return boardcatalog.Measure{}, false
	}

	return boardregistry.Catalog().MeasureOf(kind, string(m))
}

func (m Measure) ValueKind(kind recordkind.Kind) boardcatalog.ValueKind {
	if measure, ok := m.Spec(kind); ok {
		return measure.Value
	}

	return boardcatalog.ValueNumber
}
