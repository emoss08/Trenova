package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*BillingProfile)(nil)
	_ domain.Validatable        = (*BillingProfile)(nil)
)

type BillingProfile struct {
	bun.BaseModel `bun:"table:customer_billing_profiles,alias:cbr" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID     pulid.ID `json:"customerId" bun:"customer_id,pk,notnull,type:VARCHAR(100)"`

	// Core Fields
	BillingCycleType BillingCycleType `json:"billingCycleType" bun:"billing_cycle_type,type:billing_cycle_type_enum,nullzero,default:'Immediate'"`

	// Billing Control Overrides (If not set, the billing control will be used)
	HasOverrides              bool                     `json:"hasOverrides" bun:"has_overrides,type:BOOLEAN,notnull,default:false"`
	EnforceCustomerBillingReq bool                     `json:"enforceCustomerBillingReq" bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"`
	ValidateCustomerRates     bool                     `json:"validateCustomerRates" bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`
	PaymentTerm               billing.PaymentTerm      `json:"paymentTerm" bun:"payment_term,type:payment_term_enum,nullzero,default:'Net30'"`
	AutoTransfer              bool                     `json:"autoTransfer" bun:"auto_transfer,type:BOOLEAN,nullzero,default:true"`
	TransferCriteria          billing.TransferCriteria `json:"transferCriteria" bun:"transfer_criteria,type:transfer_criteria_enum,nullzero,default:'ReadyAndCompleted'"`
	AutoMarkReadyToBill       bool                     `json:"autoMarkReadyToBill" bun:"auto_mark_ready_to_bill,type:BOOLEAN,nullzero,default:true"`
	AutoBill                  bool                     `json:"autoBill" bun:"auto_bill,type:BOOLEAN,nullzero,default:true"`
	AutoBillCriteria          billing.AutoBillCriteria `json:"autoBillCriteria" bun:"auto_bill_criteria,type:auto_bill_criteria_enum,nullzero,default:'Delivered'"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`

	// Document Types that are required for this customer billing profile
	DocumentTypes []*billing.DocumentType `bun:"m2m:billing_profile_document_types,join:BillingProfile=DocumentType" json:"documentTypes,omitempty"`
}

func (b *BillingProfile) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, b,
		// * Ensure Customer ID is set
		validation.Field(&b.CustomerID, validation.Required.Error("Customer ID is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (b *BillingProfile) GetID() string {
	return b.ID.String()
}

func (b *BillingProfile) GetTableName() string {
	return "customer_billing_profiles"
}

func (b *BillingProfile) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

// BillingProfileDocumentType is a many-to-many relationship between BillingProfiles and DocumentTypes
type BillingProfileDocumentType struct {
	bun.BaseModel `bun:"table:billing_profile_document_types,alias:bpdt" json:"-"`

	// Primary keys matching your database schema
	BillingProfileID pulid.ID        `json:"billingProfileId" bun:"billing_profile_id,pk,type:VARCHAR(100)"`
	BillingProfile   *BillingProfile `bun:"rel:belongs-to,join:billing_profile_id=id"`

	DocumentTypeID pulid.ID              `json:"documentTypeId" bun:"document_type_id,pk,type:VARCHAR(100)"`
	DocumentType   *billing.DocumentType `bun:"rel:belongs-to,join:document_type_id=id"`

	// Other fields needed for referential integrity
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100)"`
	CustomerID     pulid.ID `json:"customerId" bun:"customer_id,type:VARCHAR(100)"`

	CreatedAt int64 `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (b *BillingProfileDocumentType) GetTableName() string {
	return "billing_profile_document_types"
}

func (b *BillingProfileDocumentType) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		b.CreatedAt = now
	}

	return nil
}
