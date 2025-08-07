/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/stretchr/testify/require"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/test/testutils"
)

func TestHazardousMaterialRepository(t *testing.T) {
	org := ts.Fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := ts.Fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)
	hm := ts.Fixture.MustRow("HazardousMaterial.test_hazardous_material").(*hazardousmaterial.HazardousMaterial)

	repo := repositories.NewHazardousMaterialRepository(
		repositories.HazardousMaterialRepositoryParams{
			Logger: logger.NewLogger(testutils.NewTestConfig()),
			DB:     ts.DB,
		},
	)

	t.Run("list", func(t *testing.T) {
		opts := &ports.LimitOffsetQueryOptions{
			Limit:  10,
			Offset: 0,
			TenantOpts: ports.TenantOptions{
				OrgID: org.ID,
				BuID:  bu.ID,
			},
		}

		testutils.TestRepoList(ctx, t, repo, opts)
	})

	t.Run("get by id", func(t *testing.T) {
		testutils.TestRepoGetByID(ctx, t, repo, repoports.GetHazardousMaterialByIDOptions{
			ID:    hm.ID,
			OrgID: org.ID,
			BuID:  bu.ID,
		})
	})

	t.Run("get with invalid id", func(t *testing.T) {
		hmaterial, err := repo.GetByID(ctx, repoports.GetHazardousMaterialByIDOptions{
			ID:    "invalid-id",
			OrgID: org.ID,
			BuID:  bu.ID,
		})

		require.Error(t, err, "hazardous material not found")
		require.Nil(t, hmaterial)
	})

	t.Run("create", func(t *testing.T) {
		newEntity := &hazardousmaterial.HazardousMaterial{
			Name:           "Test Hazardous Material",
			Description:    "Test Hazardous Material Description",
			Status:         domain.StatusActive,
			Class:          hazardousmaterial.HazardousClass1And1,
			PackingGroup:   hazardousmaterial.PackingGroupI,
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
		}

		testutils.TestRepoCreate(ctx, t, repo, newEntity)
	})

	t.Run("update", func(t *testing.T) {
		hm.Description = "Test Hazardous Material 2"
		testutils.TestRepoUpdate(ctx, t, repo, hm)
	})
}
