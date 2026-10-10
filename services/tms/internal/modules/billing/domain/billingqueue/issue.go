package billingqueue

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Issue)(nil)

// IssueCode names a finding the deterministic checks raise.
type IssueCode string

const (
	IssueAccessorialNotOnRateCon   = IssueCode("AccessorialNotOnRateCon")
	IssueChargeOverRateCon         = IssueCode("ChargeOverRateCon")
	IssuePODMissing                = IssueCode("PODMissing")
	IssuePODUnsigned               = IssueCode("PODUnsigned")
	IssueDetentionELDMismatch      = IssueCode("DetentionELDMismatch")
	IssueDetentionAwaitingApproval = IssueCode("DetentionAwaitingApproval")
	IssueChargesUnsplittable       = IssueCode("ChargesUnsplittable")
	IssueBillToCreditHold          = IssueCode("BillToCreditHold")
	IssuePossibleDuplicate         = IssueCode("PossibleDuplicate")
)

// IssueSource is who raised an issue. The checks raise every issue; an agent
// may only add reasoning to one.
type IssueSource string

const (
	IssueSourceDeterministic = IssueSource("Deterministic")
	IssueSourceAgent         = IssueSource("Agent")
)

// EffectKind is what choosing an option does beyond recording the choice.
type EffectKind string

const (
	// EffectKeep bills the flagged charge as it is.
	EffectKeep = EffectKind("keep")
	// EffectDrop takes the flagged charge off the shipment.
	EffectDrop = EffectKind("drop")
	// EffectSet reprices the flagged charge.
	EffectSet = EffectKind("set")
	// EffectRequest asks the driver for the missing paperwork. It does not
	// settle the issue; the paperwork arriving does.
	EffectRequest = EffectKind("request")
	// EffectAccept bills without what the check wanted.
	EffectAccept = EffectKind("accept")
)

// ResolutionCleared is the resolution the checks themselves record when the
// finding goes away, such as a signed POD arriving.
const ResolutionCleared = "cleared"

// IssueEffect is what an option changes.
type IssueEffect struct {
	Kind     EffectKind          `json:"kind"`
	ChargeID pulid.ID            `json:"chargeId,omitempty"`
	Amount   decimal.NullDecimal `json:"amount"`
	Basis    string              `json:"basis,omitempty"`
}

// IssueOption is one way to settle an issue, worded for its button.
type IssueOption struct {
	Key    string              `json:"key"`
	Label  string              `json:"label"`
	Amount decimal.NullDecimal `json:"amount"`
	// Done is the check's line once this option settled the issue.
	Done   string      `json:"done"`
	Effect IssueEffect `json:"effect"`
}

// ChargeSnapshot is a charge as it was before an option changed it, which is
// what undo puts back.
type ChargeSnapshot struct {
	AdditionalChargeID  pulid.ID        `json:"additionalChargeId"`
	AccessorialChargeID pulid.ID        `json:"accessorialChargeId"`
	Method              string          `json:"method"`
	Amount              decimal.Decimal `json:"amount"`
	Unit                int16           `json:"unit"`
	Label               string          `json:"label"`
	Basis               string          `json:"basis"`
	Billed              decimal.Decimal `json:"billed"`
	Expected            *string         `json:"expected"`
}

// EffectSnapshot is what an applied option changed.
type EffectSnapshot struct {
	Charge *ChargeSnapshot `json:"charge,omitempty"`
	// RestoredChargeID is the charge undo recreated, when it had to.
	RestoredChargeID pulid.ID `json:"restoredChargeId,omitempty"`
}

// Issue is something about a billing queue item that a person settles before
// it can be approved.
type Issue struct {
	bun.BaseModel `bun:"table:billing_queue_issues,alias:bqis" json:"-"`

	ID              pulid.ID        `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID        `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ItemID          pulid.ID        `json:"itemId"          bun:"item_id,type:VARCHAR(100),notnull"`
	CheckKey        CheckKey        `json:"checkKey"        bun:"check_key,type:VARCHAR(20),notnull"`
	Code            IssueCode       `json:"code"            bun:"code,type:VARCHAR(50),notnull"`
	SubjectKey      string          `json:"subjectKey"      bun:"subject_key,type:VARCHAR(100),notnull,default:''"`
	Summary         string          `json:"summary"         bun:"summary,type:TEXT,notnull"`
	Reasoning       string          `json:"reasoning"       bun:"reasoning,type:TEXT,nullzero"`
	Source          IssueSource     `json:"source"          bun:"source,type:VARCHAR(20),notnull,default:'Deterministic'"`
	AgentRunID      *pulid.ID       `json:"agentRunId"      bun:"agent_run_id,type:VARCHAR(100),nullzero"`
	FlaggedChargeID *pulid.ID       `json:"flaggedChargeId" bun:"flagged_charge_id,type:VARCHAR(100),nullzero"`
	Options         []IssueOption   `json:"options"         bun:"options,type:JSONB,notnull,default:'[]'"`
	ResolutionKey   *string         `json:"resolutionKey"   bun:"resolution_key,type:VARCHAR(30),nullzero"`
	ResolutionText  string          `json:"resolutionText"  bun:"resolution_text,type:TEXT,nullzero"`
	EffectSnapshot  *EffectSnapshot `json:"-"               bun:"effect_snapshot,type:JSONB,nullzero"`
	ResolvedByID    *pulid.ID       `json:"resolvedById"    bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt      *int64          `json:"resolvedAt"      bun:"resolved_at,type:BIGINT,nullzero"`
	RequestedAt     *int64          `json:"requestedAt"     bun:"requested_at,type:BIGINT,nullzero"`
	RequestedByID   *pulid.ID       `json:"requestedById"   bun:"requested_by_id,type:VARCHAR(100),nullzero"`
	Version         int64           `json:"version"         bun:"version,type:BIGINT,notnull"`
	CreatedAt       int64           `json:"createdAt"       bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt       int64           `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull"`

	// Undoable is whether the person can take the settlement back from the
	// item. It is decided when the item is read.
	Undoable bool `json:"undoable" bun:"-"`
}

func (i *Issue) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("bqis_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *Issue) GetID() pulid.ID             { return i.ID }
func (i *Issue) GetOrganizationID() pulid.ID { return i.OrganizationID }
func (i *Issue) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }
func (i *Issue) GetTableName() string        { return "billing_queue_issues" }
func (i *Issue) IsOpen() bool                { return i.ResolutionKey == nil }
func (i *Issue) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{TableAlias: "bqis"}
}

// Option finds one of the issue's options by key.
func (i *Issue) Option(key string) *IssueOption {
	for idx := range i.Options {
		if i.Options[idx].Key == key {
			return &i.Options[idx]
		}
	}

	return nil
}

// Settled reports whether a person chose one of the issue's options, as
// opposed to the checks clearing it.
func (i *Issue) Settled() bool {
	return i.ResolutionKey != nil && *i.ResolutionKey != ResolutionCleared
}

// FindingKey is what makes a finding the same finding on the next read.
func (i *Issue) FindingKey() string {
	return string(i.Code) + "|" + i.SubjectKey
}
