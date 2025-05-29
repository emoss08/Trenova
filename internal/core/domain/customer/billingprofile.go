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

	ID                        pulid.ID            `json:"id"                        bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID            pulid.ID            `json:"businessUnitId"            bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID            pulid.ID            `json:"organizationId"            bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID                pulid.ID            `json:"customerId"                bun:"customer_id,pk,notnull,type:VARCHAR(100)"`
	BillingCycleType          BillingCycleType    `json:"billingCycleType"          bun:"billing_cycle_type,type:billing_cycle_type_enum,nullzero,default:'Immediate'"`
	PaymentTerm               billing.PaymentTerm `json:"paymentTerm"               bun:"payment_term,type:payment_term_enum,nullzero,default:'Net30'"`
	DocumentTypeIDs           []string            `json:"documentTypeIds"           bun:"document_type_ids,type:VARCHAR(100)[],notnull,default:{}"`
	HasOverrides              bool                `json:"hasOverrides"              bun:"has_overrides,type:BOOLEAN,notnull,default:false"`
	EnforceCustomerBillingReq bool                `json:"enforceCustomerBillingReq" bun:"enforce_customer_billing_req,type:BOOLEAN,notnull,default:true"`
	ValidateCustomerRates     bool                `json:"validateCustomerRates"     bun:"validate_customer_rates,type:BOOLEAN,notnull,default:true"`
	AutoTransfer              bool                `json:"autoTransfer"              bun:"auto_transfer,type:BOOLEAN,nullzero,default:true"`
	AutoMarkReadyToBill       bool                `json:"autoMarkReadyToBill"       bun:"auto_mark_ready_to_bill,type:BOOLEAN,nullzero,default:true"`
	AutoBill                  bool                `json:"autoBill"                  bun:"auto_bill,type:BOOLEAN,nullzero,default:true"`
	Version                   int64               `json:"version"                   bun:"version,type:BIGINT"`
	CreatedAt                 int64               `json:"createdAt"                 bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                 int64               `json:"updatedAt"                 bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (b *BillingProfile) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(
		ctx,
		b,
		// * Ensure Customer ID is set
		validation.Field(&b.CustomerID, validation.Required.Error("Customer ID is required")),
		validation.Field(
			&b.DocumentTypeIDs,
			validation.Required.Error("Document Type IDs are required"),
		),
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

func (b *BillingProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
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
