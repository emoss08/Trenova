package agentdefinition

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	VersionIDPrefix   = "agdv_"
	MaxVersionSummary = 500
	// VersionsKept is how much of an agent's history is offered to restore
	// from; older versions stay for the audit of what the agent once did.
	VersionsKept = 50
)

// DefinitionVersion is an agent as it stood after one save: the whole definition, who
// saved it and a one-line account of what that save changed. Restoring a
// version loads it into the builder as a draft; nothing is saved until the
// person saves it, which writes a new version.
type DefinitionVersion struct {
	bun.BaseModel `bun:"table:agent_definition_versions,alias:agdv" json:"-"`

	ID                pulid.ID    `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID    `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID    `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	AgentDefinitionID pulid.ID    `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	Version           int64       `json:"version"           bun:"version,type:BIGINT,notnull"`
	Snapshot          *Definition `json:"snapshot"          bun:"snapshot,type:JSONB,notnull"`
	AuthorID          *pulid.ID   `json:"authorId"          bun:"author_id,type:VARCHAR(100),nullzero"`
	Summary           string      `json:"summary"           bun:"summary,type:VARCHAR(500),nullzero"`
	CreatedAt         int64       `json:"createdAt"         bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Author *tenant.User `json:"author,omitempty" bun:"rel:belongs-to,join:author_id=id"`
}

// NewVersion records how the agent stands after a save. The summary names
// what the save changed from before, or that the agent was created.
func NewVersion(saved, before *Definition, authorID *pulid.ID) *DefinitionVersion {
	summary := "Created"
	if before != nil {
		summary = editchange.Summary(Changes(before, saved))
	}
	if len(summary) > MaxVersionSummary {
		summary = summary[:MaxVersionSummary]
	}

	snapshot := *saved
	snapshot.BusinessUnit = nil
	snapshot.Organization = nil

	return &DefinitionVersion{
		BusinessUnitID:    saved.BusinessUnitID,
		OrganizationID:    saved.OrganizationID,
		AgentDefinitionID: saved.ID,
		Version:           saved.Version,
		Snapshot:          &snapshot,
		AuthorID:          authorID,
		Summary:           summary,
	}
}

func (v *DefinitionVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if v.ID.IsNil() {
			v.ID = pulid.MustNew(VersionIDPrefix)
		}
		v.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
