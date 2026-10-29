package agentdecision

import (
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AgentDecision)(nil)
	_ pagination.CursorEntity            = (*AgentDecision)(nil)
)

type AgentDecision struct {
	keel.Table `keel:"agent_decisions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ad_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ProposalID      *pulid.ID      `json:"proposalId"           keel:"proposal_id"`
	ExceptionID     *pulid.ID      `json:"exceptionId"          keel:"exception_id"`
	DecidedByUserID pulid.ID       `json:"decidedByUserId"      keel:"decided_by_user_id"`
	Decision        DecisionType   `json:"decision"             keel:"decision"`
	Modifications   map[string]any `json:"modifications"        keel:"modifications,jsonb,nullzero"`
	ReasonCode      string         `json:"reasonCode"           keel:"reason_code"`
	Note            string         `json:"note"                 keel:"note,nullzero"`
	// TraceID is the trace the decision was made in.
	TraceID string `json:"traceId"              keel:"trace_id,nullzero"`
	// Preview is what the decider was shown of the write, filtered for them,
	// and PreviewDigest its digest. PreviewReviewed says the decision named
	// that digest, so the person approved what they saw. PreviewTargetVersion
	// is the target record's version the preview was read at.
	Preview              *agentpreview.ProposalPreview `json:"preview"              keel:"preview,jsonb"`
	PreviewDigest        string                        `json:"previewDigest"        keel:"preview_digest,nullzero"`
	PreviewReviewed      bool                          `json:"previewReviewed"      keel:"preview_reviewed"`
	PreviewTargetVersion *int64                        `json:"previewTargetVersion" keel:"preview_target_version"`

	// CommitsAt is when an approval made from the person's own conversation
	// goes through, unless it is undone first. CommittedAt is when it did,
	// and UndoneAt and UndoneByUserID when and by whom it was taken back; an
	// undone decision is no decision, and the proposal waits again.
	// CommitWorkflowID names the workflow that commits it, which every
	// decision of one batch shares. All are empty for a decision that took
	// effect when it was made.
	CommitsAt        *int64    `json:"commitsAt"        keel:"commits_at"`
	CommittedAt      *int64    `json:"committedAt"      keel:"committed_at"`
	UndoneAt         *int64    `json:"undoneAt"         keel:"undone_at"`
	UndoneByUserID   *pulid.ID `json:"undoneByUserId"   keel:"undone_by_user_id"`
	CommitWorkflowID string    `json:"commitWorkflowId" keel:"commit_workflow_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	DecidedBy    *tenant.User         `json:"-" keel:"belongs:decided_by_user_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

	if err := CheckDecisionNote(d.Note); err != nil {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"A note to the agent can be at most {0} characters",
			MaxDecisionNoteLength,
		)
	}

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
