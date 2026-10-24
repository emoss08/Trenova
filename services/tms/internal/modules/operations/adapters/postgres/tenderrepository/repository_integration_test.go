//go:build integration

package tenderrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/tenderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx    context.Context
	db     *keel.DB
	tenant pagination.TenantInfo
	repo   tenderports.TenderRepository
	graph  *testutil.ShipmentGraph
	now    int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-TENDER",
		ProNumber:    "PRO-TENDER",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	return &harness{
		ctx:    dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:     db,
		tenant: tenant,
		repo: tenderrepository.New(tenderrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		graph: graph,
		now:   timeutils.NowUnix(),
	}
}

func (h *harness) carrier(t *testing.T, code string) pulid.ID {
	t.Helper()
	id := pulid.MustNew("car_")
	_, err := keel.Exec(
		h.ctx,
		h.db,
		`INSERT INTO carriers (id, business_unit_id, organization_id, code, name) VALUES ($1, $2, $3, $4, $5)`,
		id.String(),
		h.tenant.BuID.String(),
		h.tenant.OrgID.String(),
		code,
		"Carrier "+code,
	)
	require.NoError(t, err)
	return id
}

func (h *harness) acceptedTender(t *testing.T) *tender.Tender {
	t.Helper()
	sentAt := h.now - 600
	statuses := []tender.OfferStatus{
		tender.OfferStatusDeclined,
		tender.OfferStatusAccepted,
		tender.OfferStatusSkipped,
	}
	entity := &tender.Tender{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ShipmentID:     h.graph.Shipment.ID,
		ShipmentMoveID: h.graph.Moves[0].ID,
		Mode:           tender.ModeSpotSequential,
		Status:         tender.StatusAccepted,
	}
	for idx, status := range statuses {
		offer := &tender.TenderOffer{
			CarrierID:       h.carrier(t, "C"+string(rune('A'+idx))),
			Rank:            int16(idx + 1),
			RateMethod:      shipment.CarrierRateMethodFlat,
			Rate:            decimal.NewFromInt(int64(1500 + 100*idx)),
			OfferTTLSeconds: 3600,
			Channel:         tender.ChannelEmail,
			RecipientEmail:  "dispatch@carrier.example",
			Status:          status,
		}
		if status != tender.OfferStatusSkipped {
			offer.SentAt = &sentAt
		}
		entity.Offers = append(entity.Offers, offer)
	}

	created, err := h.repo.Create(h.ctx, entity)
	require.NoError(t, err)
	accepted := created.Offers[1].ID
	tn := tenderdb.Tenders
	_, err = tn.Where(tn.ID.Eq(created.ID)).
		Update(h.ctx, h.db, tn.AcceptedOfferID.To(accepted), tn.AcceptedAt.To(sentAt))
	require.NoError(t, err)
	created.AcceptedOfferID = &accepted
	return created
}

func TestReopenRevokesTheAcceptanceAndInsertsTheReoffers(t *testing.T) {
	h := newHarness(t)
	entity := h.acceptedTender(t)
	revoked := entity.Offers[1]
	reoffers := entity.ReofferPlan(revoked)
	require.Len(t, reoffers, 1)

	reopened, err := h.repo.Reopen(h.ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     h.tenant,
		TenderID:       entity.ID,
		RevokedOfferID: revoked.ID,
		Reason:         "Truck broke down",
		At:             h.now,
		Reoffers:       reoffers,
	})
	require.NoError(t, err)
	require.True(t, reopened)

	read, err := h.repo.GetByID(h.ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    h.tenant,
		TenderID:      entity.ID,
		IncludeOffers: true,
	})
	require.NoError(t, err)
	assert.Equal(t, tender.StatusActive, read.Status)
	assert.Equal(t, int16(1), read.BounceCount)
	require.NotNil(t, read.LastBouncedAt)
	assert.Equal(t, h.now, *read.LastBouncedAt)
	assert.Nil(t, read.AcceptedOfferID)
	assert.Nil(t, read.AcceptedAt)
	require.Len(t, read.Offers, 4)

	gotRevoked := read.FindOffer(revoked.ID)
	require.NotNil(t, gotRevoked)
	assert.Equal(t, tender.OfferStatusRevoked, gotRevoked.Status)
	assert.Equal(t, "Truck broke down", gotRevoked.RevokeReason)
	require.NotNil(t, gotRevoked.RevokedAt)

	reoffer := read.FindOffer(reoffers[0].ID)
	require.NotNil(t, reoffer)
	assert.Equal(t, tender.OfferStatusPending, reoffer.Status)
	assert.Equal(t, int16(3), reoffer.Rank)
	require.NotNil(t, reoffer.ReofferedFromID)
	assert.Equal(t, entity.Offers[2].ID, *reoffer.ReofferedFromID)

	again, err := h.repo.Reopen(h.ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     h.tenant,
		TenderID:       entity.ID,
		RevokedOfferID: revoked.ID,
		Reason:         "Duplicate request",
		At:             h.now,
	})
	require.NoError(t, err)
	assert.False(t, again, "a tender no longer holding the acceptance is not reopened twice")
}

func TestReopenLeavesNothingBehindWhenTheOfferMovedFirst(t *testing.T) {
	h := newHarness(t)
	entity := h.acceptedTender(t)
	revoked := entity.Offers[1]
	of := tenderdb.TenderOffers
	_, err := of.Where(of.ID.Eq(revoked.ID)).
		Update(h.ctx, h.db, of.Status.To(tender.OfferStatusSuperseded))
	require.NoError(t, err)

	reopened, err := h.repo.Reopen(h.ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     h.tenant,
		TenderID:       entity.ID,
		RevokedOfferID: revoked.ID,
		Reason:         "Truck broke down",
		At:             h.now,
		Reoffers:       entity.ReofferPlan(revoked),
	})
	require.NoError(t, err)
	assert.False(t, reopened)

	read, err := h.repo.GetByID(h.ctx, tenderports.GetTenderByIDRequest{
		TenantInfo:    h.tenant,
		TenderID:      entity.ID,
		IncludeOffers: true,
	})
	require.NoError(t, err)
	assert.Equal(t, tender.StatusAccepted, read.Status, "the tender change rolled back")
	assert.Equal(t, int16(0), read.BounceCount)
	assert.Len(t, read.Offers, 3, "no re-offer was kept")
}

func TestACounterIsStoredOnTheOffer(t *testing.T) {
	h := newHarness(t)
	entity := h.acceptedTender(t)
	tn := tenderdb.Tenders
	_, err := tn.Where(tn.ID.Eq(entity.ID)).Update(
		h.ctx,
		h.db,
		tn.Status.To(tender.StatusActive),
		tn.AcceptedOfferID.ToNull(),
		tn.AcceptedAt.ToNull(),
	)
	require.NoError(t, err)
	offer := entity.Offers[1]
	of := tenderdb.TenderOffers
	_, err = of.Where(of.ID.Eq(offer.ID)).Update(h.ctx, h.db, of.Status.To(tender.OfferStatusSent))
	require.NoError(t, err)

	deadline := h.now + 3600
	moved, err := h.repo.UpdateOfferStatus(h.ctx, &tenderports.UpdateOfferStatusRequest{
		TenantInfo:     h.tenant,
		OfferID:        offer.ID,
		FromStatus:     []tender.OfferStatus{tender.OfferStatusSent},
		ToStatus:       tender.OfferStatusCountered,
		RespondedAt:    &h.now,
		ResponseSource: tender.ResponseSourceEmail,
		ExpiresAt:      &deadline,
		Counter: &tenderports.OfferCounter{
			Rate:          decimal.NewFromInt(1850),
			Note:          "Fuel is up",
			At:            h.now,
			MarginPercent: decimal.NewNullDecimal(decimal.NewFromFloat(7.5)),
			NeedsApproval: true,
		},
	})
	require.NoError(t, err)
	require.True(t, moved)

	read, err := h.repo.GetOfferByID(h.ctx, tenderports.GetTenderOfferByIDRequest{
		TenantInfo: h.tenant,
		OfferID:    offer.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, tender.OfferStatusCountered, read.Status)
	assert.True(t, decimal.NewFromInt(1850).Equal(read.CounterRate.Decimal))
	assert.Equal(t, "Fuel is up", read.CounterNote)
	assert.True(t, read.CounterNeedsApproval)
	assert.True(t, decimal.NewFromFloat(7.5).Equal(read.CounterMargin.Decimal))
	assert.Equal(t, deadline, *read.ExpiresAt)
	assert.True(t, decimal.NewFromInt(1850).Equal(read.AgreedRate()))

	tenders, err := h.repo.ListForSweep(h.ctx, tenderports.ListTendersForSweepRequest{
		TenantInfo: &h.tenant,
		ExpiredBy:  deadline + 1,
	})
	require.NoError(t, err)
	require.Len(t, tenders, 1, "a counter overdue for its decision is swept")

	tenders, err = h.repo.ListForSweep(h.ctx, tenderports.ListTendersForSweepRequest{
		TenantInfo: &h.tenant,
		ExpiredBy:  h.now - 1,
	})
	require.NoError(t, err)
	assert.Empty(t, tenders, "a counter still waiting is left alone")
}

func TestTheDatabaseRefusesACounterWithoutARate(t *testing.T) {
	h := newHarness(t)
	entity := h.acceptedTender(t)

	of := tenderdb.TenderOffers
	_, err := of.Where(of.ID.Eq(entity.Offers[0].ID)).
		Update(h.ctx, h.db, of.Status.To(tender.OfferStatusCountered))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ck_tender_offers_countered")
}

func TestASweepWaitsOutTheGraceAfterABounce(t *testing.T) {
	h := newHarness(t)
	entity := h.acceptedTender(t)
	tn := tenderdb.Tenders
	_, err := tn.Where(tn.ID.Eq(entity.ID)).Update(h.ctx, h.db, tn.CreatedAt.To(h.now-86400))
	require.NoError(t, err)
	reopened, err := h.repo.Reopen(h.ctx, &tenderports.ReopenTenderRequest{
		TenantInfo:     h.tenant,
		TenderID:       entity.ID,
		RevokedOfferID: entity.Offers[1].ID,
		Reason:         "Truck broke down",
		At:             h.now,
		Reoffers:       entity.ReofferPlan(entity.Offers[1]),
	})
	require.NoError(t, err)
	require.True(t, reopened)

	tenders, err := h.repo.ListForSweep(h.ctx, tenderports.ListTendersForSweepRequest{
		TenantInfo: &h.tenant,
		ExpiredBy:  h.now - 120,
	})
	require.NoError(t, err)
	assert.Empty(t, tenders, "a fresh round is not mistaken for a dead workflow")

	tenders, err = h.repo.ListForSweep(h.ctx, tenderports.ListTendersForSweepRequest{
		TenantInfo: &h.tenant,
		ExpiredBy:  h.now + 1,
	})
	require.NoError(t, err)
	require.Len(t, tenders, 1)
	assert.Equal(t, int16(1), tenders[0].BounceCount)
}
