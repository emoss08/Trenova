package accountingsync

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	inboundChangeIDPrefix   = "acctic_"
	maxInboundResolution    = 1000
	maxInboundNote          = 500
	maxInboundPartyName     = 200
	maxInboundModifiedBy    = 200
	InboundEvaluationSettle = 2 * time.Minute
)

var (
	ErrInboundChangeClosed = errors.New(
		"the change was already applied, ignored or superseded",
	)
	ErrInboundChangeNotProposed = errors.New("only a proposed change can be applied")
	ErrInboundNoteRequired      = errors.New("a note is required to ignore a change")
	ErrInboundNotApplicable     = errors.New(
		"this change cannot be applied until what it pays matches Trenova",
	)
)

type InboundLine struct {
	DocumentKind       InboundDocumentKind `json:"documentKind"`
	DocumentExternalID string              `json:"documentExternalId"`
	AmountMinor        int64               `json:"amountMinor"`
	ObjectType         SyncObjectType      `json:"objectType,omitempty"`
	ObjectID           pulid.ID            `json:"objectId,omitempty"`
	ObjectNumber       string              `json:"objectNumber,omitempty"`
	OpenMinor          int64               `json:"openMinor,omitempty"`
}

func (l *InboundLine) Matched() bool {
	return !l.ObjectID.IsNil()
}

type InboundDocument struct {
	ReferenceNumber   string         `json:"referenceNumber,omitempty"`
	MethodExternalID  string         `json:"methodExternalId,omitempty"`
	MethodName        string         `json:"methodName,omitempty"`
	AccountExternalID string         `json:"accountExternalId,omitempty"`
	UnappliedMinor    int64          `json:"unappliedMinor"`
	Voided            bool           `json:"voided"`
	Lines             []*InboundLine `json:"lines"`
}

func (d *InboundDocument) LinesOf(kinds ...InboundDocumentKind) []*InboundLine {
	lines := make([]*InboundLine, 0, len(d.Lines))
	for _, line := range d.Lines {
		if slices.Contains(kinds, line.DocumentKind) {
			lines = append(lines, line)
		}
	}
	return lines
}

func (d *InboundDocument) AnyMatched() bool {
	return slices.ContainsFunc(d.Lines, (*InboundLine).Matched)
}

type AppliedObject struct {
	Type AppliedObjectType `json:"type"`
	ID   pulid.ID          `json:"id"`
}

type AccountingInboundChange struct {
	bun.BaseModel             `bun:"table:accounting_inbound_changes,alias:acctic" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                        json:"-"`

	ID                 pulid.ID            `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID            `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID       pulid.ID            `json:"connectionId"       bun:"connection_id,type:VARCHAR(100),notnull"`
	Kind               InboundChangeKind   `json:"kind"               bun:"kind,type:VARCHAR(30),notnull"`
	ExternalID         string              `json:"externalId"         bun:"external_id,type:VARCHAR(100),notnull"`
	ExternalNumber     string              `json:"externalNumber"     bun:"external_number,type:VARCHAR(100),nullzero"`
	ProviderModifiedAt *int64              `json:"providerModifiedAt" bun:"provider_modified_at,type:BIGINT,nullzero"`
	ProviderModifiedBy string              `json:"providerModifiedBy" bun:"provider_modified_by,type:VARCHAR(200),nullzero"`
	TxnDate            int64               `json:"txnDate"            bun:"txn_date,type:BIGINT,notnull"`
	AmountMinor        int64               `json:"amountMinor"        bun:"amount_minor,type:BIGINT,notnull"`
	CurrencyCode       string              `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull"`
	PartyExternalID    string              `json:"partyExternalId"    bun:"party_external_id,type:VARCHAR(100),nullzero"`
	PartyName          string              `json:"partyName"          bun:"party_name,type:VARCHAR(200),nullzero"`
	PartyObjectID      pulid.ID            `json:"partyObjectId"      bun:"party_object_id,type:VARCHAR(100),nullzero"`
	Document           InboundDocument     `json:"document"           bun:"document,type:JSONB,notnull"`
	Status             InboundChangeStatus `json:"status"             bun:"status,type:VARCHAR(20),notnull"`
	Reason             InboundChangeReason `json:"reason"             bun:"reason,type:VARCHAR(30),nullzero"`
	Resolution         string              `json:"resolution"         bun:"resolution,type:TEXT,nullzero"`
	AppliedObjects     []AppliedObject     `json:"appliedObjects"     bun:"applied_objects,type:JSONB,notnull"`
	DecidedByID        pulid.ID            `json:"decidedById"        bun:"decided_by_id,type:VARCHAR(100),nullzero"`
	DecidedAt          *int64              `json:"decidedAt"          bun:"decided_at,type:BIGINT,nullzero"`
	Note               string              `json:"note"               bun:"note,type:TEXT,nullzero"`
	DetectedAt         int64               `json:"detectedAt"         bun:"detected_at,type:BIGINT,notnull"`
	Version            int64               `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64               `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64               `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	DecidedBy    *tenant.User         `json:"decidedBy,omitempty"    bun:"rel:belongs-to,join:decided_by_id=id"`
}

type InboundObservation struct {
	TenantInfo         pagination.TenantInfo
	ConnectionID       pulid.ID
	Kind               InboundChangeKind
	ExternalID         string
	ExternalNumber     string
	ProviderModifiedAt *int64
	ProviderModifiedBy string
	TxnDate            int64
	AmountMinor        int64
	CurrencyCode       string
	PartyExternalID    string
	PartyName          string
	Document           InboundDocument
	At                 int64
}

func NewAccountingInboundChange(o *InboundObservation) *AccountingInboundChange {
	change := &AccountingInboundChange{
		ID:             pulid.MustNew(inboundChangeIDPrefix),
		OrganizationID: o.TenantInfo.OrgID,
		BusinessUnitID: o.TenantInfo.BuID,
		ConnectionID:   o.ConnectionID,
		Kind:           o.Kind,
		ExternalID:     o.ExternalID,
		Status:         InboundStatusDetected,
		AppliedObjects: []AppliedObject{},
		DetectedAt:     o.At,
	}
	change.observe(o)
	return change
}

func (c *AccountingInboundChange) observe(o *InboundObservation) {
	c.ExternalNumber = stringutils.TruncateRunes(o.ExternalNumber, 100)
	c.ProviderModifiedAt = o.ProviderModifiedAt
	c.ProviderModifiedBy = stringutils.TruncateRunes(o.ProviderModifiedBy, maxInboundModifiedBy)
	c.TxnDate = o.TxnDate
	c.AmountMinor = max(o.AmountMinor, 0)
	c.CurrencyCode = strings.ToUpper(o.CurrencyCode)
	c.PartyExternalID = o.PartyExternalID
	c.PartyName = stringutils.TruncateRunes(o.PartyName, maxInboundPartyName)
	c.Document = o.Document
}

func (c *AccountingInboundChange) unchangedBy(o *InboundObservation) bool {
	return c.ProviderModifiedAt != nil && o.ProviderModifiedAt != nil &&
		*c.ProviderModifiedAt == *o.ProviderModifiedAt &&
		c.AmountMinor == o.AmountMinor &&
		c.Document.Voided == o.Document.Voided
}

func (c *AccountingInboundChange) Observe(o *InboundObservation) bool {
	switch {
	case c.Status.IsOpen() && !c.unchangedBy(o):
		c.observe(o)
		c.Status = InboundStatusDetected
		c.Reason = ""
		c.Resolution = ""
		c.DetectedAt = o.At
		return true
	default:
		return false
	}
}

func (c *AccountingInboundChange) SettledAt(now int64) bool {
	return now-c.DetectedAt >= int64(InboundEvaluationSettle/time.Second)
}

func (c *AccountingInboundChange) Propose(reason InboundChangeReason, resolution string) {
	c.Status = InboundStatusProposed
	c.Reason = reason
	c.Resolution = stringutils.TruncateRunes(resolution, maxInboundResolution)
}

func (c *AccountingInboundChange) Ignore(
	reason InboundChangeReason,
	resolution string,
	actorID pulid.ID,
	at int64,
) {
	c.Status = InboundStatusIgnored
	c.Reason = reason
	c.Resolution = stringutils.TruncateRunes(resolution, maxInboundResolution)
	c.DecidedByID = actorID
	c.DecidedAt = &at
}

func (c *AccountingInboundChange) Dismiss(actorID pulid.ID, note string, at int64) error {
	if !c.Status.IsOpen() {
		return ErrInboundChangeClosed
	}
	note = stringutils.TruncateRunes(stringutils.OneLine(note, maxInboundNote), maxInboundNote)
	if note == "" {
		return ErrInboundNoteRequired
	}
	c.Status = InboundStatusIgnored
	c.Note = note
	c.DecidedByID = actorID
	c.DecidedAt = &at
	return nil
}

func (c *AccountingInboundChange) CanApply() error {
	switch {
	case !c.Status.IsOpen():
		return ErrInboundChangeClosed
	case c.Status != InboundStatusProposed:
		return ErrInboundChangeNotProposed
	case !c.Reason.Applicable():
		return ErrInboundNotApplicable
	default:
		return nil
	}
}

func (c *AccountingInboundChange) MarkApplied(
	objects []AppliedObject,
	actorID pulid.ID,
	resolution string,
	at int64,
) {
	c.Status = InboundStatusApplied
	c.Reason = ""
	c.Resolution = stringutils.TruncateRunes(resolution, maxInboundResolution)
	c.AppliedObjects = append([]AppliedObject{}, objects...)
	c.DecidedByID = actorID
	c.DecidedAt = &at
}

func (c *AccountingInboundChange) Supersede(reason InboundChangeReason, resolution string) bool {
	if !c.Status.IsOpen() {
		return false
	}
	c.Status = InboundStatusSuperseded
	c.Reason = reason
	c.Resolution = stringutils.TruncateRunes(resolution, maxInboundResolution)
	return true
}

func (c *AccountingInboundChange) GetTableName() string { return "accounting_inbound_changes" }

func (c *AccountingInboundChange) GetID() pulid.ID { return c.ID }

func (c *AccountingInboundChange) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *AccountingInboundChange) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *AccountingInboundChange) GetCreatedAt() int64 { return c.CreatedAt }

func (c *AccountingInboundChange) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: c.OrganizationID, BuID: c.BusinessUnitID}
}

func (c *AccountingInboundChange) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "acctic",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "external_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "party_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "resolution",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (c *AccountingInboundChange) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if c.AppliedObjects == nil {
		c.AppliedObjects = []AppliedObject{}
	}
	if c.Document.Lines == nil {
		c.Document.Lines = []*InboundLine{}
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew(inboundChangeIDPrefix)
		}
		if c.DetectedAt == 0 {
			c.DetectedAt = now
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}
	return nil
}
