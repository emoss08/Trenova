package tenantbootstrap

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func HasSystemDocumentTypes(ctx context.Context, db bun.IDB, orgID, buID pulid.ID) (bool, error) {
	count, err := db.NewSelect().
		Model((*documenttype.DocumentType)(nil)).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("is_system = true").
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("check existing system document types: %w", err)
	}

	return count > 0, nil
}

func CreateDocumentTypes(ctx context.Context, tx bun.IDB, scope Scope) (int, error) {
	orgID, buID := scope.OrganizationID, scope.BusinessUnitID
	docTypes := []documenttype.DocumentType{
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

	_, err := tx.NewInsert().
		Model(&docTypes).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("insert system document types: %w", err)
	}

	for i := range docTypes {
		if err = scope.record(ctx, "document_types", docTypes[i].ID); err != nil {
			return 0, fmt.Errorf("track document type: %w", err)
		}
	}

	return len(docTypes), nil
}
