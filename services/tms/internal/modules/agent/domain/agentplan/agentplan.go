package agentplan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*AgentPlan)(nil)
	_ pagination.CursorEntity            = (*AgentPlan)(nil)
	_ domaintypes.PostgresSearchable     = (*AgentPlan)(nil)
)

// AgentPlan is several proposals from one run as one decision.
//
// A run that needs five writes used to produce five cards, each approved on
// its own, in whatever order the approver clicked. A plan is approved once
// and its steps run in the order the agent asked for them; the first step
// that fails stops the plan, and the steps after it are skipped rather than
// run against a world the failed step was meant to change. Steps that each
// change a different record of one kind depend on none of the others, so
// every one of them runs and the plan records how many went through.
type AgentPlan struct {
	keel.Table `keel:"agent_plans"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:apl_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RunID   pulid.ID   `json:"runId"   keel:"run_id"`
	Title   string     `json:"title"   keel:"title"`
	Summary string     `json:"summary" keel:"summary,nullzero"`
	Status  PlanStatus `json:"status"  keel:"status"`

	StepCount      int    `json:"stepCount"      keel:"step_count"`
	CompletedSteps int    `json:"completedSteps" keel:"completed_steps"`
	FailedStep     *int   `json:"failedStep"     keel:"failed_step"`
	FailureError   string `json:"failureError"   keel:"failure_error,nullzero"`

	DecidedByUserID *pulid.ID `json:"decidedByUserId"       keel:"decided_by_user_id"`
	DecidedAt       *int64    `json:"decidedAt"             keel:"decided_at"`
	// ExpiresAt is the latest expiry among the plan's steps: while any step
	// can still be decided, so can the plan.
	ExpiresAt int64 `json:"expiresAt"             keel:"expires_at,nullzero"`
	// CommitsAt is when a plan approved from the person's own conversation
	// starts to run, unless it is undone first; UndoneAt and UndoneByUserID
	// say it was. Empty for a plan decided anywhere else.
	CommitsAt      *int64    `json:"commitsAt"             keel:"commits_at"`
	UndoneAt       *int64    `json:"undoneAt"              keel:"undone_at"`
	UndoneByUserID *pulid.ID `json:"undoneByUserId"        keel:"undone_by_user_id"`
	// The decision as it was made: its reason, the decider's note and the
	// digest of the preview they decided on. An approval in its undo window
	// carries them until it commits.
	ApprovalReasonCode    string `json:"-"                     keel:"approval_reason_code,nullzero"`
	ApprovalNote          string `json:"approvalNote"          keel:"approval_note,nullzero"`
	ApprovalPreviewDigest string `json:"approvalPreviewDigest" keel:"approval_preview_digest,nullzero"`
	// ScriptStepKey is the ledger step of the script whose intents are the
	// plan's steps; empty for a plan of the model's own calls.
	ScriptStepKey string `json:"scriptStepKey"         keel:"script_step_key,nullzero"`
	// Script is what the script that wrote the plan was asked to do and the
	// code it ran, kept with the plan once the run's steps are swept.
	Script *ScriptOrigin `json:"script"                keel:"script,jsonb"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Run          *agent.AgentRun      `json:"-" keel:"belongs:run_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

const maxPlanTitleLength = 200

const MaxNamedPlanTitleLength = 80

var (
	ErrPlanTitleEmpty   = errors.New("title must name the plan")
	ErrPlanTitleTooLong = fmt.Errorf("title is at most %d characters", MaxNamedPlanTitleLength)
)

func NamedPlanTitle(raw string) (string, error) {
	title := strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, raw)), " ")
	switch {
	case title == "":
		return "", ErrPlanTitleEmpty
	case utf8.RuneCountInString(title) > MaxNamedPlanTitleLength:
		return "", ErrPlanTitleTooLong
	default:
		return title, nil
	}
}

// PlanToolName is how a plan reads where a tool's name would: several
// writes decided as one, not any single tool's.
const PlanToolName = "plan"

func (p *AgentPlan) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&p.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&p.RunID, validation.Required.Error("Run is required")),
		validation.Field(
			&p.Title,
			validation.Required.Error("Title is required"),
			validation.Length(1, maxPlanTitleLength).
				Error("Title cannot be longer than 200 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PlanStatus]("Status is invalid"),
		),
		validation.Field(&p.StepCount, validation.Min(2).Error("A plan needs at least two steps")),
	))
}

// Expired reports whether the plan's window has closed as of now.
func (p *AgentPlan) Expired(now int64) bool {
	return p.ExpiresAt > 0 && p.ExpiresAt <= now
}

func (p *AgentPlan) GetID() pulid.ID { return p.ID }

func (p *AgentPlan) GetCreatedAt() int64 { return p.CreatedAt }

func (p *AgentPlan) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *AgentPlan) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *AgentPlan) GetTableName() string { return "agent_plans" }

func (p *AgentPlan) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "apl",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (p *AgentPlan) BeforeCreate(ctx context.Context) error {
	if p.Status == "" {
		p.Status = PlanStatusPending
	}

	return nil
}

func (*AgentPlan) IsPendingDecision() {}

// ScriptOrigin is the script a batch came from, as the model asked for it.
type ScriptOrigin struct {
	Title   string `json:"title"`
	Purpose string `json:"purpose"`
	Code    string `json:"code"`
}
