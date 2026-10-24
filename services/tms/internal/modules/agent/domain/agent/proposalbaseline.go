package agent

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ProposalBaseline)(nil)

// ProposalBaseline is what a write would have done when it was proposed,
// taken in the same snapshot as the version the proposal pins. It is kept
// apart from the proposal so the workflow never carries it, and it holds
// every value but a Confidential one, unfiltered: it is read only to say
// which values moved since, and is never served as it is.
type ProposalBaseline struct {
	bun.BaseModel `bun:"table:agent_proposal_baselines,alias:apb" json:"-"`
	keel.Table    `keel:"agent_proposal_baselines"`

	ProposalID     pulid.ID `json:"proposalId"     bun:"proposal_id,pk,type:VARCHAR(100),notnull" keel:"proposal_id,pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`

	ToolName      string       `json:"toolName"      bun:"tool_name,type:VARCHAR(100),notnull" keel:"tool_name"`
	Preview       *ToolPreview `json:"preview"       bun:"preview,type:JSONB,notnull" keel:"preview,jsonb"`
	TargetVersion *int64       `json:"targetVersion" bun:"target_version,type:BIGINT,nullzero" keel:"target_version"`
	CreatedAt     int64        `json:"createdAt"     bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
}

func (b *ProposalBaseline) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		b,
		validation.Field(&b.ProposalID, validation.Required.Error("Proposal id is required")),
		validation.Field(&b.OrganizationID,
			validation.Required.Error("Organization id is required"),
		),
		validation.Field(&b.BusinessUnitID,
			validation.Required.Error("Business unit id is required"),
		),
		validation.Field(&b.ToolName, validation.Required.Error("Tool name is required")),
		validation.Field(&b.Preview, validation.Required.Error("Preview is required")),
		validation.Field(&b.TargetVersion,
			validation.Min(int64(0)).Error("Target version cannot be negative"),
		),
	))

	if b.Preview != nil && EncodedPreviewSize(b.Preview) > MaxPreviewBaselineBytes {
		multiErr.Add("preview", errortypes.ErrInvalid, "Preview is larger than a baseline keeps")
	}
}

func (b *ProposalBaseline) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok && b.CreatedAt == 0 {
		b.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
