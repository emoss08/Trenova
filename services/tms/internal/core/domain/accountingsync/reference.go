package accountingsync

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*AccountingReferenceObject)(nil)

type AccountingReferenceObject struct {
	bun.BaseModel `bun:"table:accounting_reference_objects,alias:acctro" json:"-"`

	ID                      pulid.ID      `json:"id"                      bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID          pulid.ID      `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID      `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID            pulid.ID      `json:"connectionId"            bun:"connection_id,type:VARCHAR(100),notnull"`
	Kind                    ReferenceKind `json:"kind"                    bun:"kind,type:VARCHAR(30),notnull"`
	ExternalID              string        `json:"externalId"              bun:"external_id,type:VARCHAR(100),notnull"`
	Name                    string        `json:"name"                    bun:"name,type:VARCHAR(500),notnull"`
	SearchName              string        `json:"-"                       bun:"search_name,type:VARCHAR(500),notnull"`
	FullyQualifiedName      string        `json:"fullyQualifiedName"      bun:"fully_qualified_name,type:VARCHAR(1000),nullzero"`
	Number                  string        `json:"number"                  bun:"number,type:VARCHAR(100),nullzero"`
	Description             string        `json:"description"             bun:"description,type:TEXT,nullzero"`
	Classification          string        `json:"classification"          bun:"classification,type:VARCHAR(50),nullzero"`
	AccountType             string        `json:"accountType"             bun:"account_type,type:VARCHAR(100),nullzero"`
	AccountSubType          string        `json:"accountSubType"          bun:"account_sub_type,type:VARCHAR(100),nullzero"`
	ItemType                string        `json:"itemType"                bun:"item_type,type:VARCHAR(50),nullzero"`
	SubType                 string        `json:"subType"                 bun:"sub_type,type:VARCHAR(50),nullzero"`
	ParentExternalID        string        `json:"parentExternalId"        bun:"parent_external_id,type:VARCHAR(100),nullzero"`
	Active                  bool          `json:"active"                  bun:"active,type:BOOLEAN,notnull"`
	CurrencyCode            string        `json:"currencyCode"            bun:"currency_code,type:VARCHAR(3),nullzero"`
	SyncToken               string        `json:"-"                       bun:"sync_token,type:VARCHAR(50),nullzero"`
	CompanyName             string        `json:"companyName"             bun:"company_name,type:VARCHAR(500),nullzero"`
	Email                   string        `json:"email"                   bun:"email,type:VARCHAR(320),nullzero"`
	AddressLine1            string        `json:"addressLine1"            bun:"address_line1,type:VARCHAR(500),nullzero"`
	City                    string        `json:"city"                    bun:"city,type:VARCHAR(255),nullzero"`
	State                   string        `json:"state"                   bun:"state,type:VARCHAR(100),nullzero"`
	PostalCode              string        `json:"postalCode"              bun:"postal_code,type:VARCHAR(30),nullzero"`
	IncomeAccountExternalID string        `json:"incomeAccountExternalId" bun:"income_account_external_id,type:VARCHAR(100),nullzero"`
	DueDays                 *int          `json:"dueDays"                 bun:"due_days,type:INTEGER,nullzero"`
	Is1099                  bool          `json:"is1099"                  bun:"is_1099,type:BOOLEAN,notnull"`
	ProviderUpdatedAt       *int64        `json:"providerUpdatedAt"       bun:"provider_updated_at,type:BIGINT,nullzero"`
	LastSeenAt              int64         `json:"lastSeenAt"              bun:"last_seen_at,type:BIGINT,notnull"`
	RemovedAt               *int64        `json:"removedAt"               bun:"removed_at,type:BIGINT,nullzero"`
	CreatedAt               int64         `json:"createdAt"               bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64         `json:"updatedAt"               bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (r *AccountingReferenceObject) GetTableName() string { return "accounting_reference_objects" }

func (r *AccountingReferenceObject) GetID() pulid.ID { return r.ID }

func (r *AccountingReferenceObject) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *AccountingReferenceObject) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *AccountingReferenceObject) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	r.SearchName = stringutils.NormalizeName(r.Name)

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("acctro_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *AccountingReferenceObject) Usable() bool {
	if !r.Active || r.RemovedAt != nil {
		return false
	}
	if r.Kind == ReferenceKindItem {
		return r.ItemType != ItemTypeCategory && r.ItemType != ItemTypeGroup
	}
	return true
}

func (r *AccountingReferenceObject) Label() string {
	if r.FullyQualifiedName != "" {
		return r.FullyQualifiedName
	}
	return r.Name
}

const (
	ItemTypeCategory = "Category"
	ItemTypeGroup    = "Group"
)
