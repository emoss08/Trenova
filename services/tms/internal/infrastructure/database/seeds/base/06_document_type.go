package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type DocumentTypeSeed struct {
	seedhelpers.BaseSeed
}

func NewDocumentTypeSeed() *DocumentTypeSeed {
	seed := &DocumentTypeSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"DocumentType",
		"1.0.0",
		"Creates default system document types for transportation operations",
		[]common.Environment{
			common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest,
		},
	)

	seed.SetDependencies(seedhelpers.SeedAdminAccount)

	return seed
}

func (s *DocumentTypeSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetOrganization("default_org")
			if err != nil {
				org, err = sc.GetDefaultOrganization(ctx)
				if err != nil {
					return fmt.Errorf("get default organization: %w", err)
				}
			}

			count, err := tx.NewSelect().
				Model((*documenttype.DocumentType)(nil)).
				Where("organization_id = ?", org.ID).
				Where("business_unit_id = ?", org.BusinessUnitID).
				Where("is_system = true").
				Count(ctx)
			if err != nil {
				return fmt.Errorf("check existing system document types: %w", err)
			}

			if count > 0 {
				return nil
			}

			if err = s.createSystemDocumentTypes(ctx, tx, sc, org.ID, org.BusinessUnitID); err != nil {
				return fmt.Errorf("create system document types: %w", err)
			}

			seedhelpers.LogSuccess(
				"Created system document type fixtures",
				"- Created 5 system document types",
			)

			return nil
		},
	)
}

func (s *DocumentTypeSeed) createSystemDocumentTypes(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
) error {
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
	}

	_, err := tx.NewInsert().
		Model(&docTypes).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("insert system document types: %w", err)
	}

	for i := range docTypes {
		if err = sc.TrackCreated(ctx, "document_types", docTypes[i].ID, s.Name()); err != nil {
			return fmt.Errorf("track document type: %w", err)
		}
	}

	return nil
}

func (s *DocumentTypeSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *DocumentTypeSeed) CanRollback() bool {
	return true
}
