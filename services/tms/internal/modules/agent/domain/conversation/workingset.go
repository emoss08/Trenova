package conversation

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

type WorkingSource string

const (
	WorkingSourceSubject WorkingSource = "subject"
	WorkingSourceMention WorkingSource = "mention"
	WorkingSourcePage    WorkingSource = "page"
	WorkingSourceRead    WorkingSource = "read"
	WorkingSourceWrote   WorkingSource = "wrote"
)

type WorkingRecord struct {
	Kind      permission.RecordKind `json:"kind"`
	ID        string                `json:"id"`
	Label     string                `json:"label,omitempty"`
	Source    WorkingSource         `json:"source"`
	TouchedAt int64                 `json:"touchedAt"`
}

func (r WorkingRecord) Ref() agentpagecontext.EntityRef {
	return agentpagecontext.EntityRef{Type: string(r.Kind), ID: r.ID, Label: r.Label}
}
