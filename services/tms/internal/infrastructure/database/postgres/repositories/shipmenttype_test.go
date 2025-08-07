/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestShipmentTypeRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	stype := ts.Fixture.MustRow("ShipmentType.ftl_shipment_type").(*shipmenttype.ShipmentType)

	repo := repositories.NewShipmentTypeRepository(repositories.ShipmentTypeRepositoryParams{
		Logger: logger.NewLogger(testutils.NewTestConfig()),
		DB:     ts.DB,
	})

	t.Run("list", func(t *testing.T) {
		opts := &repoports.ListShipmentTypeRequest{
			Filter: &ports.LimitOffsetQueryOptions{
				Limit:  10,
				Offset: 0,
				TenantOpts: ports.TenantOptions{
					OrgID: org.ID,
					BuID:  bu.ID,
				},
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetShipmentTypeByIDOptions{
			ID:    stype.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		entity, err := repo.GetByID(ctx, repoports.GetShipmentTypeByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "entity not found")
		require.Nil(t, entity)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &shipmenttype.ShipmentType{
			Code:           "TEST",
			Description:    "Test Service Type",
			Color:          "#000000",
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		stype.Description = "Test Service Type 2"
		testutils.TestRepoUpdate(ctx, t, repo, stype)
	})
}
