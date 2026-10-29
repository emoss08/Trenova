package graphqlread

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
)

const (
	VerbGet = "get"
	// IDVariable is the variable a get's operation takes the id in.
	IDVariable = "id"
)

// MarkedRecord is a get's record that shows text written outside the
// organization, encoded as the record and naming itself so the run marks
// what it read.
type MarkedRecord struct {
	Record  map[string]any
	Outside []agentrecord.RecordRef
}

func (r *MarkedRecord) MarshalJSON() ([]byte, error) { return sonic.Marshal(r.Record) }

func (r *MarkedRecord) TaintedRecords() []agentrecord.RecordRef { return r.Outside }
