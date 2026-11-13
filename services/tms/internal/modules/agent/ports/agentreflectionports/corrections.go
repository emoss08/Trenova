package agentreflectionports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ListCorrectedWorkRequest asks for what people changed by hand, between
// ChangedSince and Until, of records an agent had written since WroteSince.
type ListCorrectedWorkRequest struct {
	TenantInfo   pagination.TenantInfo
	WroteSince   int64
	ChangedSince int64
	Until        int64
}

// CorrectedField is one field an agent set and a person then changed: what
// stood after the agent's work, and what the person made it.
type CorrectedField struct {
	Field string `json:"field"`
	Was   string `json:"was"`
	Now   string `json:"now"`
}

// CorrectedWork is one record an agent wrote and a person later edited by
// hand, with the fields of the agent's that the person changed.
type CorrectedWork struct {
	AgentDefinitionID pulid.ID         `json:"agentDefinitionId"`
	Resource          string           `json:"resource"`
	ResourceID        string           `json:"resourceId"`
	Label             string           `json:"label"`
	Fields            []CorrectedField `json:"fields"`
	WrittenAt         int64            `json:"writtenAt"`
	ChangedAt         int64            `json:"changedAt"`
}

type CorrectedWorkRepository interface {
	ListCorrectedWork(ctx context.Context, req *ListCorrectedWorkRequest) ([]CorrectedWork, error)
}

// ReflectOnCorrectionsRequest is one habit of correction to look back over:
// the same field of the same kind of record, set by one agent and changed by
// people on several records.
type ReflectOnCorrectionsRequest struct {
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
	AgentDefinitionID pulid.ID              `json:"agentDefinitionId"`
	PatternKey        string                `json:"patternKey"`
	Resource          string                `json:"resource"`
	Field             string                `json:"field"`
	Day               int                   `json:"day"`
	Work              []CorrectedWork       `json:"work"`
}

// ListCorrectionLookBacksRequest asks which of the patterns a look back has
// already read since a time, failed ones aside.
type ListCorrectionLookBacksRequest struct {
	TenantInfo  pagination.TenantInfo
	PatternKeys []string
	Since       int64
}
