//go:build integration

package edicommunicationprofilerepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newAS2Profile(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	code string,
) *edi.EDICommunicationProfile {
	t.Helper()

	partner := &edi.EDIPartner{
		Kind: edi.PartnerKindExternal,
		Code: code,
		Name: "AS2 Partner " + code,
	}
	require.NoError(t, edidb.EDIPartners.Create(ctx, db, partner))

	return &edi.EDICommunicationProfile{
		EDIPartnerID: partner.ID,
		Method:       edi.ConnectionMethodAS2,
		Status:       domaintypes.StatusActive,
		Name:         "AS2 " + code,
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
	ctxA, ctxB := tenantA.Ctx(ctx), tenantB.Ctx(ctx)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	owned, err := repo.CreateProfile(ctxA, newAS2Profile(t, ctxA, db, "AS2-A"))
	require.NoError(t, err)

	_, err = repo.CreateProfile(ctxB, newAS2Profile(t, ctxB, db, "AS2-B"))
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))
	assert.Equal(
		t,
		"uq_edi_communication_profiles_active_as2_identifiers",
		dberror.ExtractConstraintName(err),
	)

	routed, err := repo.GetActiveAS2ProfileByIdentifiers(
		ctx,
		edipartnerports.GetActiveAS2ProfileByIdentifiersRequest{
			LocalAS2ID:   "SHIPPER-AS2",
			PartnerAS2ID: "CARRIER-AS2",
		},
	)
	require.NoError(t, err)
	assert.Equal(t, owned.ID, routed.ID)
	assert.Equal(t, tenantA.Organization.ID, routed.OrganizationID)

	inactive := newAS2Profile(t, ctxB, db, "AS2-C")
	inactive.Status = domaintypes.StatusInactive
	_, err = repo.CreateProfile(ctxB, inactive)
	require.NoError(t, err)

	_, err = repo.GetProfileByID(
		ctxB,
		edipartnerports.GetEDICommunicationProfileByIDRequest{ID: owned.ID},
	)
	require.True(t, errortypes.IsNotFoundError(err))
}

func TestPollOutcomeKeepsTheProfileVersion(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenant := seedtest.SeedFullTestData(t, ctx, db)
	tctx := tenant.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	profile := newAS2Profile(t, tctx, db, "AS2-P")
	profile.Method = edi.ConnectionMethodSFTP
	profile.Config = map[string]any{"inboundDirectory": "/in"}
	created, err := repo.CreateProfile(tctx, profile)
	require.NoError(t, err)

	require.NoError(
		t,
		repo.RecordInboundPollOutcome(tctx, edipartnerports.RecordEDIProfilePollOutcomeRequest{
			ProfileID: created.ID,
			PolledAt:  100,
			Error:     "timeout",
		}),
	)
	require.NoError(
		t,
		repo.RecordInboundPollOutcome(tctx, edipartnerports.RecordEDIProfilePollOutcomeRequest{
			ProfileID: created.ID,
			PolledAt:  200,
			Success:   true,
		}),
	)

	polled, err := repo.GetProfileByID(
		tctx,
		edipartnerports.GetEDICommunicationProfileByIDRequest{ID: created.ID},
	)
	require.NoError(t, err)
	assert.Equal(t, created.Version, polled.Version)
	require.NotNil(t, polled.LastPollSuccessAt)
	assert.Equal(t, int64(200), *polled.LastPollSuccessAt)
	assert.Empty(t, polled.LastPollError)

	profiles, err := repo.ListInboundPollingProfiles(ctx)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, created.ID, profiles[0].ID)

	stale, err := repo.CountStaleInboundPollingProfiles(ctx, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stale)

	polled.Name = "Renamed"
	updated, err := repo.UpdateProfile(tctx, polled)
	require.NoError(t, err)
	assert.Equal(t, created.Version+1, updated.Version)
}
