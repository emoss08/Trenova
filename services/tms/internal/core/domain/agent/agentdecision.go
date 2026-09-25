package agent

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AgentDecision)(nil)
	_ validationframework.TenantedEntity = (*AgentDecision)(nil)
	_ pagination.CursorEntity            = (*AgentDecision)(nil)
)

type AgentDecision struct {
	bun.BaseModel `bun:"table:agent_decisions,alias:ad" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	ProposalID      *pulid.ID      `json:"proposalId"      bun:"proposal_id,type:VARCHAR(100),nullzero"`
	ExceptionID     *pulid.ID      `json:"exceptionId"     bun:"exception_id,type:VARCHAR(100),nullzero"`
	DecidedByUserID pulid.ID       `json:"decidedByUserId" bun:"decided_by_user_id,type:VARCHAR(100),notnull"`
	Decision        DecisionType   `json:"decision"        bun:"decision,type:agent_decision_type_enum,notnull"`
	Modifications   map[string]any `json:"modifications"   bun:"modifications,type:JSONB,nullzero"`
	ReasonCode      string         `json:"reasonCode"      bun:"reason_code,type:VARCHAR(100),notnull"`
	// TraceID is the trace the decision was made in.
	TraceID string `json:"traceId" bun:"trace_id,type:VARCHAR(32),nullzero"`
	// Preview is what the decider was shown of the write, filtered for them,
	// and PreviewDigest its digest. PreviewReviewed says the decision named
	// that digest, so the person approved what they saw. PreviewTargetVersion
	// is the target record's version the preview was read at.
	Preview              *ProposalPreview `json:"preview"              bun:"preview,type:JSONB,nullzero"`
	PreviewDigest        string           `json:"previewDigest"        bun:"preview_digest,type:VARCHAR(64),nullzero"`
	PreviewReviewed      bool             `json:"previewReviewed"      bun:"preview_reviewed,type:BOOLEAN,notnull,default:false"`
	PreviewTargetVersion *int64           `json:"previewTargetVersion" bun:"preview_target_version,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	DecidedBy    *tenant.User         `bun:"rel:belongs-to,join:decided_by_user_id=id" json:"-"`
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"   json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"    json:"-"`
}

func (d *AgentDecision) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		d,
		validation.Field(&d.DecidedByUserID,
			validation.Required.Error("Decisions must be attributed to a human user"),
		),
		validation.Field(&d.Decision,
			validation.Required.Error("Decision is required"),
			domainvalidation.ValidEnum[DecisionType]("Invalid decision"),
		),
		validation.Field(&d.ReasonCode, validation.Required.Error("Reason code is required")),
		validation.Field(&d.TraceID, domainvalidation.TraceID("Trace id is invalid")),
		validation.Field(&d.PreviewDigest, validation.By(func(any) error {
			if d.PreviewDigest != "" && !stringutils.IsLowerHexOfLength(d.PreviewDigest, 64) {
				return errors.New("preview digest must be 64 lowercase hex characters")
			}
			return nil
		})),
		validation.Field(&d.PreviewTargetVersion,
			validation.Min(int64(0)).Error("Preview target version cannot be negative"),
		),
	))

	if d.PreviewReviewed && d.PreviewDigest == "" {
		multiErr.Add(
			"previewReviewed",
			errortypes.ErrInvalid,
			"A reviewed preview must name the digest that was reviewed",
		)
	}

	if d.subjectCount() != 1 {
		multiErr.Add(
			"proposalId",
			errortypes.ErrInvalid,
			"A decision must reference exactly one proposal or exception",
		)
	}

	if d.Decision == DecisionModified && len(d.Modifications) == 0 {
		multiErr.Add(
			"modifications",
			errortypes.ErrRequired,
			"Modifications are required when the decision is Modified",
		)
	}
}

func (d *AgentDecision) subjectCount() int {
	count := 0
	if d.ProposalID != nil && d.ProposalID.IsNotNil() {
		count++
	}
	if d.ExceptionID != nil && d.ExceptionID.IsNotNil() {
		count++
	}
	return count
}

func (d *AgentDecision) GetID() pulid.ID {
	return d.ID
}

func (d *AgentDecision) GetCreatedAt() int64 {
	return d.CreatedAt
}

func (d *AgentDecision) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *AgentDecision) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *AgentDecision) GetTableName() string {
	return "agent_decisions"
}

func (d *AgentDecision) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("ad_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
