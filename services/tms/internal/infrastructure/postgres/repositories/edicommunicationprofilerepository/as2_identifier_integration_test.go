//go:build integration

package edicommunicationprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func newAS2Profile(
	t *testing.T,
	db *bun.DB,
	tenant *seedtest.TestData,
	code string,
) *edi.EDICommunicationProfile {
	t.Helper()

	partner := &edi.EDIPartner{
		BusinessUnitID: tenant.BusinessUnit.ID,
		OrganizationID: tenant.Organization.ID,
		Kind:           edi.PartnerKindExternal,
		Code:           code,
		Name:           "AS2 Partner " + code,
	}
	_, err := db.NewInsert().Model(partner).Exec(t.Context())
	require.NoError(t, err)

	return &edi.EDICommunicationProfile{
		ID:             pulid.MustNew("ecp_"),
		BusinessUnitID: tenant.BusinessUnit.ID,
		OrganizationID: tenant.Organization.ID,
		EDIPartnerID:   partner.ID,
		Method:         edi.ConnectionMethodAS2,
		Status:         domaintypes.StatusActive,
		Name:           "AS2 " + code,
		Config: map[string]any{
			"localAS2Id":   "SHIPPER-AS2",
			"partnerAS2Id": " CARRIER-AS2 ",
			"endpointUrl":  "https://as2.example.com/inbound",
			"mdnMode":      "sync",
		},
		EncryptedSecrets: map[string]string{},
	}
}

func TestAS2IdentifierPairsAreUniqueAcrossTenants(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "AS")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	owned, err := repo.CreateProfile(ctx, newAS2Profile(t, db, tenantA, "AS2-A"))
	require.NoError(t, err)

	_, err = repo.CreateProfile(ctx, newAS2Profile(t, db, tenantB, "AS2-B"))
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))
	assert.Equal(
		t,
		"uq_edi_communication_profiles_active_as2_identifiers",
		dberror.ExtractConstraintName(err),
	)

	routed, err := repo.GetActiveAS2ProfileByIdentifiers(
		ctx,
		repositories.GetActiveAS2ProfileByIdentifiersRequest{
			LocalAS2ID:   "SHIPPER-AS2",
			PartnerAS2ID: "CARRIER-AS2",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, owned.ID, routed.ID)
	assert.Equal(t, tenantA.Organization.ID, routed.OrganizationID)

	inactive := newAS2Profile(t, db, tenantB, "AS2-C")
	inactive.Status = domaintypes.StatusInactive
	_, err = repo.CreateProfile(ctx, inactive)
	require.NoError(t, err)
}
