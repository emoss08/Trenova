package accountingdrift

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	driftFindingIDPrefix = "acctdf_"
	maxDriftNote         = 500
	maxDriftPartyName    = 200
	maxDriftModifiedBy   = 200
	maxDriftState        = 50
	maxDriftDetailLines  = 50
)

type DriftLine struct {
	ObjectType    accountingsync.SyncObjectType `json:"objectType"`
	ObjectID      pulid.ID                      `json:"objectId"`
	ObjectNumber  string                        `json:"objectNumber"`
	TrenovaMinor  int64                         `json:"trenovaMinor"`
	ProviderMinor int64                         `json:"providerMinor"`
}

// Documents Trenova sent to the accounting system that now differ there, with both values and how each was settled
type AccountingDriftFinding struct {
	keel.Table `keel:"accounting_drift_findings,report=Accounting,label=Drift Finding"`

	ID                 pulid.ID                      `json:"id"                 keel:"id,pk,autoid:acctdf_"`
	BusinessUnitID     pulid.ID                      `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID                      `json:"organizationId"     keel:"organization_id,pk,tenant"`
	ConnectionID       pulid.ID                      `json:"connectionId"       keel:"connection_id"`
	ObjectType         accountingsync.SyncObjectType `json:"objectType"         keel:"object_type,label=Document Type"`
	ObjectID           pulid.ID                      `json:"objectId"           keel:"object_id,label=Document ID"`
	ObjectNumber       string                        `json:"objectNumber"       keel:"object_number,nullzero,label=Document Number"`
	PartyID            pulid.ID                      `json:"partyId"            keel:"party_id,nullzero"`
	PartyName          string                        `json:"partyName"          keel:"party_name,nullzero"`
	ExternalID         string                        `json:"externalId"         keel:"external_id,nullzero,label=ID in the Books"`
	ExternalURL        string                        `json:"externalUrl"        keel:"external_url,nullzero,label=Link in the Books"`
	Kind               DriftKind                     `json:"kind"               keel:"kind,label=Difference"`
	CurrencyCode       string                        `json:"currencyCode"       keel:"currency_code,label=Currency"`
	TrenovaMinor       *int64                        `json:"trenovaMinor"       keel:"trenova_minor,label=Trenova Value (Minor)"`
	ProviderMinor      *int64                        `json:"providerMinor"      keel:"provider_minor,label=Books Value (Minor)"`
	DifferenceMinor    *int64                        `json:"differenceMinor"    keel:"difference_minor,label=Difference (Minor)"`
	TrenovaState       string                        `json:"trenovaState"       keel:"trenova_state,nullzero,label=State in Trenova"`
	ProviderState      string                        `json:"providerState"      keel:"provider_state,nullzero,label=State in the Books"`
	Detail             []DriftLine                   `json:"detail"             keel:"detail,jsonb,noreport"`
	ProviderModifiedAt *int64                        `json:"providerModifiedAt" keel:"provider_modified_at,label=Changed in the Books At"`
	ProviderModifiedBy string                        `json:"providerModifiedBy" keel:"provider_modified_by,nullzero,label=Changed in the Books By"`
	Status             DriftStatus                   `json:"status"             keel:"status"`
	Resolution         DriftResolution               `json:"resolution"         keel:"resolution,nullzero"`
	ResolutionNote     string                        `json:"resolutionNote"     keel:"resolution_note,nullzero,label=Note"`
	FixObjectType      DriftFixObject                `json:"fixObjectType"      keel:"fix_object_type,nullzero,label=Fixed With"`
	FixObjectID        pulid.ID                      `json:"fixObjectId"        keel:"fix_object_id,nullzero,label=Fix ID"`
	ResolvedByID       pulid.ID                      `json:"resolvedById"       keel:"resolved_by_id,nullzero"`
	ResolvedAt         *int64                        `json:"resolvedAt"         keel:"resolved_at"`
	DetectedAt         int64                         `json:"detectedAt"         keel:"detected_at"`
	LastSeenAt         int64                         `json:"lastSeenAt"         keel:"last_seen_at"`
	Version            int64                         `json:"version"            keel:"version,version"`
	CreatedAt          int64                         `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64                         `json:"updatedAt"          keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	ResolvedBy   *tenant.User         `json:"resolvedBy,omitempty"   keel:"belongs:resolved_by_id,noreport"`
}

func (f *AccountingDriftFinding) BeforeUpdate(_ context.Context) error {
	if f.Detail == nil {
		f.Detail = []DriftLine{}
	}

	return nil
}

type DriftObservation struct {
	TenantInfo         pagination.TenantInfo
	ConnectionID       pulid.ID
	ObjectType         accountingsync.SyncObjectType
	ObjectID           pulid.ID
	ObjectNumber       string
	PartyID            pulid.ID
	PartyName          string
	ExternalID         string
	ExternalURL        string
	Kind               DriftKind
	CurrencyCode       string
	TrenovaMinor       *int64
	ProviderMinor      *int64
	TrenovaState       string
	ProviderState      string
	Detail             []DriftLine
	ProviderModifiedAt *int64
	ProviderModifiedBy string
	At                 int64
}

func NewAccountingDriftFinding(o *DriftObservation) *AccountingDriftFinding {
	finding := &AccountingDriftFinding{
		ID:             pulid.MustNew(driftFindingIDPrefix),
		OrganizationID: o.TenantInfo.OrgID,
		BusinessUnitID: o.TenantInfo.BuID,
		ConnectionID:   o.ConnectionID,
		ObjectType:     o.ObjectType,
		ObjectID:       o.ObjectID,
		Kind:           o.Kind,
		Status:         DriftStatusOpen,
		DetectedAt:     o.At,
	}
	finding.observe(o)
	return finding
}

func (f *AccountingDriftFinding) observe(o *DriftObservation) {
	f.ObjectNumber = stringutils.TruncateRunes(o.ObjectNumber, 100)
	f.PartyID = o.PartyID
	f.PartyName = stringutils.TruncateRunes(o.PartyName, maxDriftPartyName)
	f.ExternalID = o.ExternalID
	if o.ExternalURL != "" {
		f.ExternalURL = o.ExternalURL
	}
	f.CurrencyCode = strings.ToUpper(o.CurrencyCode)
	f.TrenovaMinor = o.TrenovaMinor
	f.ProviderMinor = o.ProviderMinor
	f.DifferenceMinor = nil
	if o.TrenovaMinor != nil && o.ProviderMinor != nil {
		difference := *o.ProviderMinor - *o.TrenovaMinor
		f.DifferenceMinor = &difference
	}
	f.TrenovaState = stringutils.TruncateRunes(o.TrenovaState, maxDriftState)
	f.ProviderState = stringutils.TruncateRunes(o.ProviderState, maxDriftState)
	f.Detail = append([]DriftLine{}, o.Detail[:min(len(o.Detail), maxDriftDetailLines)]...)
	f.ProviderModifiedAt = o.ProviderModifiedAt
	f.ProviderModifiedBy = stringutils.TruncateRunes(o.ProviderModifiedBy, maxDriftModifiedBy)
	f.LastSeenAt = o.At
}

func (f *AccountingDriftFinding) Observe(o *DriftObservation) {
	if f.IsOpen() {
		f.observe(o)
	}
}

func (f *AccountingDriftFinding) IsOpen() bool {
	return f.Status == DriftStatusOpen
}

func (f *AccountingDriftFinding) Pushed() bool {
	return f.IsOpen() && f.FixObjectType == DriftFixSyncRecord
}

func (f *AccountingDriftFinding) Directions() []DriftDirection {
	directions := make([]DriftDirection, 0, len(AllDriftDirections()))
	if !f.Kind.IsBalance() {
		directions = append(directions, DriftPushTrenovaValue)
	}
	if f.adjustable() {
		directions = append(directions, DriftAdjustTrenova)
	}
	return directions
}

func (f *AccountingDriftFinding) adjustable() bool {
	if f.ObjectType == accountingsync.SyncObjectInvoice ||
		f.ObjectType == accountingsync.SyncObjectDebitMemo {
		return f.Kind == DriftAmountMismatch || f.Kind.Gone()
	}
	return f.ObjectType == accountingsync.SyncObjectCustomerPayment && f.Kind.Gone()
}

func (f *AccountingDriftFinding) CanFix(direction DriftDirection) error {
	if !f.IsOpen() {
		return ErrDriftClosed
	}
	if f.Kind == DriftTrialBalanceMismatch {
		return ErrDriftExplainOnly
	}
	if !slices.Contains(f.Directions(), direction) {
		return ErrDriftFixUnavailable
	}
	return nil
}

func (f *AccountingDriftFinding) WithinTolerance(toleranceMinor int64) bool {
	if !f.Kind.IsMoney() || f.DifferenceMinor == nil {
		return false
	}
	difference := *f.DifferenceMinor
	if difference < 0 {
		difference = -difference
	}
	return difference <= toleranceMinor
}

func (f *AccountingDriftFinding) MarkPushed(recordID, actorID pulid.ID) {
	f.FixObjectType = DriftFixSyncRecord
	f.FixObjectID = recordID
	f.ResolvedByID = actorID
}

func (f *AccountingDriftFinding) MarkAdjusted(
	fix DriftFixObject,
	fixID pulid.ID,
	actorID pulid.ID,
	at int64,
) {
	f.close(DriftStatusResolved, DriftAdjustedTrenova, at)
	f.FixObjectType = fix
	f.FixObjectID = fixID
	f.ResolvedByID = actorID
}

func (f *AccountingDriftFinding) Clear(at int64) bool {
	if !f.IsOpen() {
		return false
	}
	resolution := DriftNoLongerDiffers
	if f.Pushed() {
		resolution = DriftPushedTrenovaValue
	}
	f.close(DriftStatusResolved, resolution, at)
	return true
}

func (f *AccountingDriftFinding) Dismiss(actorID pulid.ID, note string, at int64) error {
	if !f.IsOpen() {
		return ErrDriftClosed
	}
	note = stringutils.TruncateRunes(stringutils.OneLine(note, maxDriftNote), maxDriftNote)
	if note == "" {
		return ErrDriftNoteRequired
	}
	f.close(DriftStatusDismissed, DriftDismissed, at)
	f.ResolutionNote = note
	f.ResolvedByID = actorID
	return nil
}

func (f *AccountingDriftFinding) close(status DriftStatus, resolution DriftResolution, at int64) {
	f.Status = status
	f.Resolution = resolution
	f.ResolvedAt = &at
}

func (f *AccountingDriftFinding) GetTableName() string { return "accounting_drift_findings" }

func (f *AccountingDriftFinding) GetID() pulid.ID { return f.ID }

func (f *AccountingDriftFinding) GetOrganizationID() pulid.ID { return f.OrganizationID }

func (f *AccountingDriftFinding) GetBusinessUnitID() pulid.ID { return f.BusinessUnitID }

func (f *AccountingDriftFinding) GetCreatedAt() int64 { return f.CreatedAt }

func (f *AccountingDriftFinding) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: f.OrganizationID, BuID: f.BusinessUnitID}
}

func (f *AccountingDriftFinding) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "acctdf",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "object_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "party_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "provider_modified_by",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (f *AccountingDriftFinding) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if f.Detail == nil {
		f.Detail = []DriftLine{}
	}
	f.stampSeen(now)
	f.stampSeen(timeutils.NowUnix())

	return nil
}

func (f *AccountingDriftFinding) BeforeSave(_ context.Context) error {
	if f.Detail == nil {
		f.Detail = []DriftLine{}
	}
	return nil
}

func (f *AccountingDriftFinding) stampSeen(now int64) {
	if f.DetectedAt == 0 {
		f.DetectedAt = now
	}
	if f.LastSeenAt == 0 {
		f.LastSeenAt = now
	}
}
