package accountingsync

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type AccountingReferenceObject struct {
	keel.Table `keel:"accounting_reference_objects"`

	ID                      pulid.ID      `json:"id"                      keel:"id,pk,autoid:acctro_"`
	BusinessUnitID          pulid.ID      `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID      `json:"organizationId"          keel:"organization_id,pk,tenant"`
	ConnectionID            pulid.ID      `json:"connectionId"            keel:"connection_id"`
	Kind                    ReferenceKind `json:"kind"                    keel:"kind"`
	ExternalID              string        `json:"externalId"              keel:"external_id"`
	Name                    string        `json:"name"                    keel:"name"`
	SearchName              string        `json:"-"                       keel:"search_name"`
	FullyQualifiedName      string        `json:"fullyQualifiedName"      keel:"fully_qualified_name,nullzero"`
	Number                  string        `json:"number"                  keel:"number,nullzero"`
	Description             string        `json:"description"             keel:"description,nullzero"`
	Classification          string        `json:"classification"          keel:"classification,nullzero"`
	AccountType             string        `json:"accountType"             keel:"account_type,nullzero"`
	AccountSubType          string        `json:"accountSubType"          keel:"account_sub_type,nullzero"`
	AccountClass            AccountClass  `json:"accountClass"            keel:"account_class,nullzero"`
	ItemType                string        `json:"itemType"                keel:"item_type,nullzero"`
	SubType                 string        `json:"subType"                 keel:"sub_type,nullzero"`
	ParentExternalID        string        `json:"parentExternalId"        keel:"parent_external_id,nullzero"`
	Active                  bool          `json:"active"                  keel:"active"`
	CurrencyCode            string        `json:"currencyCode"            keel:"currency_code,nullzero"`
	SyncToken               string        `json:"-"                       keel:"sync_token,nullzero"`
	CompanyName             string        `json:"companyName"             keel:"company_name,nullzero"`
	Email                   string        `json:"email"                   keel:"email,nullzero"`
	AddressLine1            string        `json:"addressLine1"            keel:"address_line1,nullzero"`
	City                    string        `json:"city"                    keel:"city,nullzero"`
	State                   string        `json:"state"                   keel:"state,nullzero"`
	PostalCode              string        `json:"postalCode"              keel:"postal_code,nullzero"`
	IncomeAccountExternalID string        `json:"incomeAccountExternalId" keel:"income_account_external_id,nullzero"`
	DueDays                 *int          `json:"dueDays"                 keel:"due_days"`
	Is1099                  bool          `json:"is1099"                  keel:"is_1099"`
	ProviderUpdatedAt       *int64        `json:"providerUpdatedAt"       keel:"provider_updated_at"`
	LastSeenAt              int64         `json:"lastSeenAt"              keel:"last_seen_at"`
	RemovedAt               *int64        `json:"removedAt"               keel:"removed_at"`
	CreatedAt               int64         `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64         `json:"updatedAt"               keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

func (r *AccountingReferenceObject) BeforeUpdate(_ context.Context) error {
	r.SearchName = stringutils.NormalizeName(r.Name)

	return nil
}

func (r *AccountingReferenceObject) BeforeCreate(_ context.Context) error {
	r.SearchName = stringutils.NormalizeName(r.Name)

	return nil
}

func (r *AccountingReferenceObject) GetTableName() string { return "accounting_reference_objects" }

func (r *AccountingReferenceObject) GetID() pulid.ID { return r.ID }

func (r *AccountingReferenceObject) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *AccountingReferenceObject) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *AccountingReferenceObject) BeforeSave(_ context.Context) error {
	r.SearchName = stringutils.NormalizeName(r.Name)
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

type AccountClass string

const (
	AccountClassReceivable       = AccountClass("Receivable")
	AccountClassPayable          = AccountClass("Payable")
	AccountClassBank             = AccountClass("Bank")
	AccountClassUndepositedFunds = AccountClass("UndepositedFunds")
	AccountClassIncome           = AccountClass("Income")
	AccountClassOtherIncome      = AccountClass("OtherIncome")
	AccountClassCostOfSales      = AccountClass("CostOfSales")
	AccountClassExpense          = AccountClass("Expense")
	AccountClassOtherExpense     = AccountClass("OtherExpense")
	AccountClassAsset            = AccountClass("Asset")
	AccountClassLiability        = AccountClass("Liability")
	AccountClassEquity           = AccountClass("Equity")
)

func (c AccountClass) String() string { return string(c) }

func (c AccountClass) IsValid() bool {
	switch c {
	case AccountClassReceivable,
		AccountClassPayable,
		AccountClassBank,
		AccountClassUndepositedFunds,
		AccountClassIncome,
		AccountClassOtherIncome,
		AccountClassCostOfSales,
		AccountClassExpense,
		AccountClassOtherExpense,
		AccountClassAsset,
		AccountClassLiability,
		AccountClassEquity:
		return true
	default:
		return false
	}
}

func AllAccountClasses() []AccountClass {
	return []AccountClass{
		AccountClassReceivable,
		AccountClassPayable,
		AccountClassBank,
		AccountClassUndepositedFunds,
		AccountClassIncome,
		AccountClassOtherIncome,
		AccountClassCostOfSales,
		AccountClassExpense,
		AccountClassOtherExpense,
		AccountClassAsset,
		AccountClassLiability,
		AccountClassEquity,
	}
}

type LedgerPartyNeed string

const (
	LedgerPartyNone     = LedgerPartyNeed("")
	LedgerPartyCustomer = LedgerPartyNeed("Customer")
	LedgerPartyVendor   = LedgerPartyNeed("Vendor")
)

func (r *AccountingReferenceObject) LedgerParty() LedgerPartyNeed {
	if r == nil || r.Kind != ReferenceKindAccount {
		return LedgerPartyNone
	}
	switch r.AccountClass { //nolint:exhaustive // only receivable and payable accounts need a party
	case AccountClassReceivable:
		return LedgerPartyCustomer
	case AccountClassPayable:
		return LedgerPartyVendor
	default:
		return LedgerPartyNone
	}
}
