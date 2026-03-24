package customer

import (
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type CustomerBillingProfileDocumentType struct {
	bun.BaseModel `bun:"table:customer_billing_profile_document_types,alias:cbpdt" json:"-"`

	OrganizationID   pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	BusinessUnitID   pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	BillingProfileID pulid.ID `bun:"billing_profile_id,pk,notnull,type:VARCHAR(100)"`
	DocumentTypeID   pulid.ID `bun:"document_type_id,pk,notnull,type:VARCHAR(100)"`

	BillingProfile *CustomerBillingProfile    `bun:"rel:belongs-to,join:billing_profile_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DocumentType   *documenttype.DocumentType `bun:"rel:belongs-to,join:document_type_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}
