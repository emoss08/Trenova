package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*CustomerBillingProfile)(nil)

//nolint:revive // This is a valid struct name
type CustomerBillingProfile struct {
	bun.BaseModel `bun:"table:customer_billing_profiles,alias:cbr" json:"-"`

	ID                        pulid.ID           `json:"id"                        bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID            pulid.ID           `json:"businessUnitId"            bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID            pulid.ID           `json:"organizationId"            bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID                pulid.ID           `json:"customerId"                bun:"customer_id,pk,notnull,type:VARCHAR(100)"`
	BillingCycleType          BillingCycleType   `json:"billingCycleType"          bun:"billing_cycle_type,type:billing_cycle_type_enum,nullzero,default:'Immediate'"`
	PaymentTerm               tenant.PaymentTerm `json:"paymentTerm"               bun:"payment_term,type:payment_term_enum,nullzero,default:'Net30'"`
	HasOverrides              bool               `json:"hasOverrides"              bun:"has_overrides,type:BOOLEAN,notnull,default:false"`
	EnforceCustomerBillingReq bool               `json:"enforceCustomerBillingReq" bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"`
	ValidateCustomerRates     bool               `json:"validateCustomerRates"     bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`
	AutoTransfer              bool               `json:"autoTransfer"              bun:"auto_transfer,type:BOOLEAN,nullzero,default:true"`
	AutoMarkReadyToBill       bool               `json:"autoMarkReadyToBill"       bun:"auto_mark_ready_to_bill,type:BOOLEAN,nullzero,default:true"`
	AutoBill                  bool               `json:"autoBill"                  bun:"auto_bill,type:BOOLEAN,nullzero,default:true"`
	Version                   int64              `json:"version"                   bun:"version,type:BIGINT"`
	CreatedAt                 int64              `json:"createdAt"                 bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                 int64              `json:"updatedAt"                 bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Customer      *Customer                    `json:"-"             bun:"rel:belongs-to,join:customer_id=id"`
	BusinessUnit  *tenant.BusinessUnit         `json:"-"             bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization  *tenant.Organization         `json:"-"             bun:"rel:belongs-to,join:organization_id=id"`
	DocumentTypes []*documenttype.DocumentType `json:"documentTypes" bun:"m2m:customer_billing_profile_document_types,join:BillingProfile=DocumentType"`
}

func (b *CustomerBillingProfile) GetID() string {
	return b.ID.String()
}

func (b *CustomerBillingProfile) GetTableName() string {
	return "customer_billing_profiles"
}

func (b *CustomerBillingProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("cbr_")
		}

		b.CreatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

type BillingProfileDocumentType struct {
	bun.BaseModel `bun:"table:customer_billing_profile_document_types,alias:bpdt" json:"-"`

	OrganizationID   pulid.ID                   `json:"-"              bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	BusinessUnitID   pulid.ID                   `json:"-"              bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	BillingProfileID pulid.ID                   `json:"-"              bun:"billing_profile_id,pk,notnull,type:VARCHAR(100)"`
	DocumentTypeID   pulid.ID                   `json:"documentTypeId" bun:"document_type_id,pk,notnull,type:VARCHAR(100)"`
	BillingProfile   *CustomerBillingProfile    `json:"-"              bun:"rel:belongs-to,join:billing_profile_id=id"`
	DocumentType     *documenttype.DocumentType `json:"-"              bun:"rel:belongs-to,join:document_type_id=id"`
}
