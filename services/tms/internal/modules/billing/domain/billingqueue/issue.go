package billingqueue

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// IssueCode names a finding the deterministic checks raise.
type IssueCode string

const (
	IssueAccessorialNotOnRateCon   = IssueCode("AccessorialNotOnRateCon")
	IssueChargeOverRateCon         = IssueCode("ChargeOverRateCon")
	IssuePODMissing                = IssueCode("PODMissing")
	IssuePODUnsigned               = IssueCode("PODUnsigned")
	IssueDetentionELDMismatch      = IssueCode("DetentionELDMismatch")
	IssueDetentionAwaitingApproval = IssueCode("DetentionAwaitingApproval")
	IssueRatingAwaitingReview      = IssueCode("RatingAwaitingReview")
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

const OptionKeyRequest = "request"

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
	keel.Table `keel:"billing_queue_issues"`

	ID              pulid.ID        `json:"id"              keel:"id,pk,autoid:bqis_"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID        `json:"organizationId"  keel:"organization_id,pk,tenant"`
	ItemID          pulid.ID        `json:"itemId"          keel:"item_id"`
	CheckKey        CheckKey        `json:"checkKey"        keel:"check_key"`
	Code            IssueCode       `json:"code"            keel:"code"`
	SubjectKey      string          `json:"subjectKey"      keel:"subject_key"`
	Summary         string          `json:"summary"         keel:"summary"`
	Reasoning       string          `json:"reasoning"       keel:"reasoning,nullzero"`
	Source          IssueSource     `json:"source"          keel:"source"`
	AgentRunID      *pulid.ID       `json:"agentRunId"      keel:"agent_run_id"`
	FlaggedChargeID *pulid.ID       `json:"flaggedChargeId" keel:"flagged_charge_id"`
	Options         []IssueOption   `json:"options"         keel:"options,jsonb"`
	ResolutionKey   *string         `json:"resolutionKey"   keel:"resolution_key"`
	ResolutionText  string          `json:"resolutionText"  keel:"resolution_text,nullzero"`
	EffectSnapshot  *EffectSnapshot `json:"-"               keel:"effect_snapshot,jsonb"`
	ResolvedByID    *pulid.ID       `json:"resolvedById"    keel:"resolved_by_id"`
	ResolvedAt      *int64          `json:"resolvedAt"      keel:"resolved_at"`
	RequestedAt     *int64          `json:"requestedAt"     keel:"requested_at"`
	RequestedByID   *pulid.ID       `json:"requestedById"   keel:"requested_by_id"`
	Version         int64           `json:"version"         keel:"version,version"`
	CreatedAt       int64           `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64           `json:"updatedAt"       keel:"updated_at,autoupdate"`

	// Undoable is whether the person can take the settlement back from the
	// item. It is decided when the item is read.
	Undoable bool `json:"undoable"`
}

func (i *Issue) BeforeCreate(_ context.Context) error {
	if i.Source == "" {
		i.Source = IssueSourceDeterministic
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
