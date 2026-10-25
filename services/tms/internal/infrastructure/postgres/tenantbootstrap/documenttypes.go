package tenantbootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/shared/pulid"
)

func CreateDocumentTypes(ctx context.Context, db *keel.DB, scope Scope) (int, error) {
	d := documenttypedb.DocumentTypes
	held, err := scoped(ctx, d.Pick(d.Code).Where(
		d.OrganizationID.Eq(scope.OrganizationID),
		d.BusinessUnitID.Eq(scope.BusinessUnitID),
	)).All(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("check existing document types: %w", err)
	}

	taken := make(map[string]struct{}, len(held))
	for _, row := range held {
		taken[strings.ToLower(row.Code)] = struct{}{}
	}

	missing := make([]*documenttype.DocumentType, 0)
	for _, docType := range systemDocumentTypes(scope) {
		if _, ok := taken[strings.ToLower(docType.Code)]; !ok {
			missing = append(missing, docType)
		}
	}

	if err = d.CreateMany(ctx, db, missing); err != nil {
		return 0, fmt.Errorf("insert system document types: %w", err)
	}

	return len(missing), nil
}

func systemDocumentTypes(scope Scope) []*documenttype.DocumentType {
	orgID, buID := scope.OrganizationID, scope.BusinessUnitID
	return []*documenttype.DocumentType{
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "INVOICE",
			Name:                   "Invoice",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryInvoice,
			Color:                  "#3b82f6",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "CREDITMEMO",
			Name:                   "Credit Memo",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryInvoice,
			Color:                  "#10b981",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "DEBITMEMO",
			Name:                   "Debit Memo",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryInvoice,
			Color:                  "#ef4444",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "POD",
			Name:                   "Proof of Delivery",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryShipment,
			Color:                  "#8b5cf6",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "BOL",
			Name:                   "Bill of Lading",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryShipment,
			Color:                  "#f59e0b",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   "RATECONF",
			Name:                   "Rate Confirmation",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryShipment,
			Color:                  "#0f766e",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   documenttype.CodeDetentionNotice,
			Name:                   "Detention Notice",
			Description:            "Customer detention notice rendered as a PDF and filed against the shipment",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryShipment,
			Color:                  "#f97316",
			IsSystem:               true,
		},
		{
			ID:                     pulid.MustNew("dt_"),
			BusinessUnitID:         buID,
			OrganizationID:         orgID,
			Code:                   documenttype.CodeRateConfirmation,
			Name:                   "Rate Confirmation",
			Description:            "Carrier rate confirmation rendered as a PDF and filed against the shipment",
			DocumentClassification: documenttype.ClassificationPublic,
			DocumentCategory:       documenttype.CategoryShipment,
			Color:                  "#0ea5e9",
			IsSystem:               true,
		},
	}
}
