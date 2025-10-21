package development

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// DocumentTypesSeed Creates document types data
type DocumentTypesSeed struct {
	seedhelpers.BaseSeed
}

// NewDocumentTypesSeed creates a new document_types seed
func NewDocumentTypesSeed() *DocumentTypesSeed {
	seed := &DocumentTypesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentTypes",
		"1.0.0",
		"Creates document types data",
		[]common.Environment{
			common.EnvDevelopment,
		},
	)

	// Development seeds typically depend on base seeds
	seed.SetDependencies("USStates", "AdminAccount", "Permissions")

	return seed
}

// Run executes the seed
func (s *DocumentTypesSeed) Run(ctx context.Context, db *bun.DB) error {
	return seedhelpers.RunInTransaction(
		ctx,
		db,
		s.Name(),
		func(ctx context.Context, tx bun.Tx, seedCtx *seedhelpers.SeedContext) error {
			var count int
			err := db.NewSelect().
				Model((*documenttype.DocumentType)(nil)).
				ColumnExpr("count(*)").
				Scan(ctx, &count)
			if err != nil {
				return err
			}

			if count > 0 {
				seedhelpers.LogSuccess("Document types already exist, skipping")
				return nil
			}

			buId, err := seedCtx.GetDefaultBusinessUnit()
			if err != nil {
				return err
			}

			orgId, err := seedCtx.GetDefaultOrganization()
			if err != nil {
				return err
			}

			return seedhelpers.RunInTransaction(
				ctx,
				db,
				s.Name(),
				func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
					docTypes := []documenttype.DocumentType{
						{
							BusinessUnitID:         buId.ID,
							OrganizationID:         orgId.ID,
							Code:                   "INVOICE",
							Name:                   "Invoice",
							Description:            "Invoice document",
							Color:                  "#C51D34",
							DocumentClassification: documenttype.ClassificationPublic,
							DocumentCategory:       documenttype.CategoryInvoice,
						},
						{
							BusinessUnitID:         buId.ID,
							OrganizationID:         orgId.ID,
							Code:                   "CM",
							Name:                   "Credit Memo",
							Description:            "Credit memo document",
							Color:                  "#646B63",
							DocumentClassification: documenttype.ClassificationPublic,
							DocumentCategory:       documenttype.CategoryInvoice,
						},
						{
							BusinessUnitID:         buId.ID,
							OrganizationID:         orgId.ID,
							Code:                   "BOL",
							Name:                   "Bill of Lading",
							Description:            "Bill of lading document",
							Color:                  "#5D9B9B",
							DocumentClassification: documenttype.ClassificationPublic,
							DocumentCategory:       documenttype.CategoryShipment,
						},
						{
							BusinessUnitID:         buId.ID,
							OrganizationID:         orgId.ID,
							Code:                   "POD",
							Name:                   "Proof of Delivery",
							Description:            "Proof of delivery document",
							Color:                  "#474A51",
							DocumentClassification: documenttype.ClassificationPublic,
							DocumentCategory:       documenttype.CategoryShipment,
						},
					}
					if _, err := tx.NewInsert().Model(&docTypes).Exec(ctx); err != nil {
						return err
					}

					seedhelpers.LogSuccess("Created document_types fixtures",
						"- 5 document types created",
					)
					return nil
				},
			)
		},
	)
}
