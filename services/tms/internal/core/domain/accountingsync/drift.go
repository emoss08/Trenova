package accountingsync

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	driftFindingIDPrefix = "acctdf_"
	maxDriftNote         = 500
	maxDriftPartyName    = 200
	maxDriftModifiedBy   = 200
	maxDriftState        = 50
	maxDriftDetailLines  = 50
)

var (
	ErrDriftClosed         = errors.New("the finding was already resolved or dismissed")
	ErrDriftNoteRequired   = errors.New("a note is required to dismiss a finding")
	ErrDriftFixUnavailable = errors.New("that fix is not offered for this finding")
)

type DriftLine struct {
	ObjectType    SyncObjectType `json:"objectType"`
	ObjectID      pulid.ID       `json:"objectId"`
	ObjectNumber  string         `json:"objectNumber"`
	TrenovaMinor  int64          `json:"trenovaMinor"`
	ProviderMinor int64          `json:"providerMinor"`
}

type AccountingDriftFinding struct {
	bun.BaseModel             `bun:"table:accounting_drift_findings,alias:acctdf" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                       json:"-"`

	ID                 pulid.ID        `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID        `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID        `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID       pulid.ID        `json:"connectionId"       bun:"connection_id,type:VARCHAR(100),notnull"`
	ObjectType         SyncObjectType  `json:"objectType"         bun:"object_type,type:VARCHAR(30),notnull"`
	ObjectID           pulid.ID        `json:"objectId"           bun:"object_id,type:VARCHAR(100),notnull"`
	ObjectNumber       string          `json:"objectNumber"       bun:"object_number,type:VARCHAR(100),nullzero"`
	PartyID            pulid.ID        `json:"partyId"            bun:"party_id,type:VARCHAR(100),nullzero"`
	PartyName          string          `json:"partyName"          bun:"party_name,type:VARCHAR(200),nullzero"`
	ExternalID         string          `json:"externalId"         bun:"external_id,type:VARCHAR(100),nullzero"`
	ExternalURL        string          `json:"externalUrl"        bun:"external_url,type:TEXT,nullzero"`
	Kind               DriftKind       `json:"kind"               bun:"kind,type:VARCHAR(30),notnull"`
	CurrencyCode       string          `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull"`
	TrenovaMinor       *int64          `json:"trenovaMinor"       bun:"trenova_minor,type:BIGINT,nullzero"`
	ProviderMinor      *int64          `json:"providerMinor"      bun:"provider_minor,type:BIGINT,nullzero"`
	DifferenceMinor    *int64          `json:"differenceMinor"    bun:"difference_minor,type:BIGINT,nullzero"`
	TrenovaState       string          `json:"trenovaState"       bun:"trenova_state,type:VARCHAR(50),nullzero"`
	ProviderState      string          `json:"providerState"      bun:"provider_state,type:VARCHAR(50),nullzero"`
	Detail             []DriftLine     `json:"detail"             bun:"detail,type:JSONB,notnull"`
	ProviderModifiedAt *int64          `json:"providerModifiedAt" bun:"provider_modified_at,type:BIGINT,nullzero"`
	ProviderModifiedBy string          `json:"providerModifiedBy" bun:"provider_modified_by,type:VARCHAR(200),nullzero"`
	Status             DriftStatus     `json:"status"             bun:"status,type:VARCHAR(20),notnull"`
	Resolution         DriftResolution `json:"resolution"         bun:"resolution,type:VARCHAR(30),nullzero"`
	ResolutionNote     string          `json:"resolutionNote"     bun:"resolution_note,type:TEXT,nullzero"`
	FixObjectType      DriftFixObject  `json:"fixObjectType"      bun:"fix_object_type,type:VARCHAR(40),nullzero"`
	FixObjectID        pulid.ID        `json:"fixObjectId"        bun:"fix_object_id,type:VARCHAR(100),nullzero"`
	ResolvedByID       pulid.ID        `json:"resolvedById"       bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt         *int64          `json:"resolvedAt"         bun:"resolved_at,type:BIGINT,nullzero"`
	DetectedAt         int64           `json:"detectedAt"         bun:"detected_at,type:BIGINT,notnull"`
	LastSeenAt         int64           `json:"lastSeenAt"         bun:"last_seen_at,type:BIGINT,notnull"`
	Version            int64           `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64           `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64           `json:"updatedAt"          bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	ResolvedBy   *tenant.User         `json:"resolvedBy,omitempty"   bun:"rel:belongs-to,join:resolved_by_id=id"`
}

type DriftObservation struct {
	TenantInfo         pagination.TenantInfo
	ConnectionID       pulid.ID
	ObjectType         SyncObjectType
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
	if f.Kind != DriftCustomerBalanceMismatch {
		directions = append(directions, DriftPushTrenovaValue)
	}
	if f.adjustable() {
		directions = append(directions, DriftAdjustTrenova)
	}
	return directions
}

func (f *AccountingDriftFinding) adjustable() bool {
	if f.ObjectType == SyncObjectInvoice || f.ObjectType == SyncObjectDebitMemo {
		return f.Kind == DriftAmountMismatch || f.Kind.Gone()
	}
	return f.ObjectType == SyncObjectCustomerPayment && f.Kind.Gone()
}

func (f *AccountingDriftFinding) CanFix(direction DriftDirection) error {
	if !f.IsOpen() {
		return ErrDriftClosed
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

func (f *AccountingDriftFinding) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if f.Detail == nil {
		f.Detail = []DriftLine{}
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew(driftFindingIDPrefix)
		}
		if f.DetectedAt == 0 {
			f.DetectedAt = now
		}
		if f.LastSeenAt == 0 {
			f.LastSeenAt = now
		}
		f.CreatedAt = now
		f.UpdatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}
	return nil
}
