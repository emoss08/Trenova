package accountingsync

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	inboundChangeIDPrefix   = "acctic_"
	maxInboundResolution    = 1000
	maxInboundNote          = 500
	maxInboundPartyName     = 200
	maxInboundModifiedBy    = 200
	InboundEvaluationSettle = 2 * time.Minute
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

// Payments recorded in the accounting system against documents Trenova sent, and whether each was applied in Trenova
type AccountingInboundChange struct {
	keel.Table `                                                             keel:"accounting_inbound_changes,report=Accounting,label=Payment From the Books,plural=Payments From the Books"`

	ID                 pulid.ID            `json:"id" keel:"id,pk,autoid:acctic_"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	ConnectionID       pulid.ID            `json:"connectionId" keel:"connection_id"`
	Kind               InboundChangeKind   `json:"kind" keel:"kind,label=Payment Kind"`
	ExternalID         string              `json:"externalId" keel:"external_id,label=ID in the Books"`
	ExternalNumber     string              `json:"externalNumber" keel:"external_number,nullzero,label=Number in the Books"`
	ExternalURL        string              `json:"externalUrl" keel:"external_url,nullzero,label=Link in the Books"`
	ProviderModifiedAt *int64              `json:"providerModifiedAt" keel:"provider_modified_at,label=Changed in the Books At"`
	ProviderModifiedBy string              `json:"providerModifiedBy" keel:"provider_modified_by,nullzero,label=Changed in the Books By"`
	TxnDate            int64               `json:"txnDate" keel:"txn_date,label=Payment Date"`
	AmountMinor        int64               `json:"amountMinor" keel:"amount_minor,label=Amount (Minor)"`
	CurrencyCode       string              `json:"currencyCode" keel:"currency_code,label=Currency"`
	PartyExternalID    string              `json:"partyExternalId" keel:"party_external_id,nullzero,noreport"`
	PartyName          string              `json:"partyName" keel:"party_name,nullzero,label=Customer or Vendor"`
	PartyObjectID      pulid.ID            `json:"partyObjectId" keel:"party_object_id,nullzero,label=Trenova Party ID"`
	Document           InboundDocument     `json:"document" keel:"document,jsonb,noreport"`
	Status             InboundChangeStatus `json:"status" keel:"status"`
	Reason             InboundChangeReason `json:"reason" keel:"reason,nullzero,label=Held Back Because"`
	Resolution         string              `json:"resolution" keel:"resolution,nullzero,label=What to Do"`
	AppliedObjects     []AppliedObject     `json:"appliedObjects" keel:"applied_objects,jsonb,noreport"`
	DecidedByID        pulid.ID            `json:"decidedById" keel:"decided_by_id,nullzero"`
	DecidedAt          *int64              `json:"decidedAt" keel:"decided_at"`
	Note               string              `json:"note" keel:"note,nullzero"`
	DetectedAt         int64               `json:"detectedAt" keel:"detected_at,label=Read At"`
	Version            int64               `json:"version" keel:"version,version"`
	CreatedAt          int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64               `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	DecidedBy    *tenant.User         `json:"decidedBy,omitempty" keel:"belongs:decided_by_id,noreport"`
}

func (c *AccountingInboundChange) BeforeUpdate(_ context.Context) error {
	c.fillCollections()

	return nil
}

type InboundObservation struct {
	TenantInfo         pagination.TenantInfo
	ConnectionID       pulid.ID
	Kind               InboundChangeKind
	ExternalID         string
	ExternalNumber     string
	ExternalURL        string
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
	if o.ExternalURL != "" {
		c.ExternalURL = o.ExternalURL
	}
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

func (c *AccountingInboundChange) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	c.fillCollections()
	if c.DetectedAt == 0 {
		c.DetectedAt = now
	}
	if c.DetectedAt == 0 {
		c.DetectedAt = timeutils.NowUnix()
	}

	return nil
}

func (c *AccountingInboundChange) BeforeSave(_ context.Context) error {
	c.fillCollections()
	return nil
}

func (c *AccountingInboundChange) fillCollections() {
	if c.AppliedObjects == nil {
		c.AppliedObjects = []AppliedObject{}
	}
	if c.Document.Lines == nil {
		c.Document.Lines = []*InboundLine{}
	}
}
