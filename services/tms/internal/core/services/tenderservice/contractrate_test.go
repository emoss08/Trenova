package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stubRateEngine answers one buy side rating per carrier.
type stubRateEngine struct {
	services.RateEngine

	byCarrier map[pulid.ID]*services.RatedShipment
	err       error
	calls     int
}

func (s *stubRateEngine) RateShipment(
	_ context.Context,
	req *services.RateShipmentRequest,
) (*services.RatedShipment, error) {
	s.calls++

	if s.err != nil {
		return nil, s.err
	}

	if rated, ok := s.byCarrier[req.PartyID]; ok {
		return rated, nil
	}

	return &services.RatedShipment{Outcome: ratequote.OutcomeNoRateFound}, nil
}

func contractRated(amount string) *services.RatedShipment {
	agreementID := pulid.MustNew("rag_")

	return &services.RatedShipment{
		Amount:      decimal.RequireFromString(amount),
		Currency:    "USD",
		Outcome:     ratequote.OutcomeRated,
		AgreementID: &agreementID,
		Quote: &ratequote.RateQuote{
			ID:      pulid.MustNew("rqt_"),
			Outcome: ratequote.OutcomeRated,
			Agreement: &rateagreement.RateAgreement{
				PartyType: rateagreement.PartyTypeCarrier,
			},
		},
	}
}

func contractEntry(carrierID pulid.ID, rank int16) *tender.RoutingGuideEntry {
	entry := guideEntry(carrierID, rank)
	entry.UseContractRate = true

	return entry
}

func expectNotifications(deps *createTestDeps) {
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Return(&notification.Notification{}, nil).Maybe()
}

// A rate frozen on a guide entry is the number somebody typed the day the guide
// was written. A guide entry that reads its carrier's contract offers what the
// contract actually says today, which is the whole point of having one.
func TestCreateWaterfall_ContractEntryOffersTheContractsRate(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)
	expectNotifications(deps)

	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:      pulid.MustNew("rgd_"),
		Status:  "Active",
		Entries: []*tender.RoutingGuideEntry{contractEntry(healthy.ID, 1)},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy}, nil)

	deps.svc.rateEngine = &stubRateEngine{
		byCarrier: map[pulid.ID]*services.RatedShipment{healthy.ID: contractRated("1875.50")},
	}

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.Len(t, created.Offers, 1)
	assert.True(t, created.Offers[0].Rate.Equal(decimal.RequireFromString("1875.50")),
		"offered %s", created.Offers[0].Rate)
	assert.Equal(t, "Flat", string(created.Offers[0].RateMethod))
}

// An entry that did not ask for contract pricing keeps the rate the guide
// carries. Reading contracts for every entry would change what existing guides
// offer on the day this shipped.
func TestCreateWaterfall_EntryWithoutContractPricingKeepsItsOwnRate(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)
	expectNotifications(deps)

	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:      pulid.MustNew("rgd_"),
		Status:  "Active",
		Entries: []*tender.RoutingGuideEntry{guideEntry(healthy.ID, 1)},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy}, nil)

	engine := &stubRateEngine{}
	deps.svc.rateEngine = engine

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.Len(t, created.Offers, 1)
	assert.True(t, created.Offers[0].Rate.Equal(decimal.NewFromInt(1200)))
	assert.Zero(t, engine.calls, "an entry that did not ask for it must not read a contract")
}

// Offering a carrier nothing because no contract covered the lane would send a
// tender for a load at zero dollars. The entry is skipped with the reason
// instead, which is what the screening report already exists to carry.
func TestCreateWaterfall_ContractEntryWithNoContractIsSkippedNotOfferedAtZero(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)
	expectNotifications(deps)

	uncovered := eligibleCarrier("uncovered")
	covered := eligibleCarrier("covered")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:     pulid.MustNew("rgd_"),
		Status: "Active",
		Entries: []*tender.RoutingGuideEntry{
			contractEntry(uncovered.ID, 1),
			contractEntry(covered.ID, 2),
		},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{uncovered, covered}, nil)

	deps.svc.rateEngine = &stubRateEngine{
		byCarrier: map[pulid.ID]*services.RatedShipment{covered.ID: contractRated("1500")},
	}

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.Len(t, created.Offers, 1)
	assert.Equal(t, covered.ID, created.Offers[0].CarrierID)

	require.NotNil(t, created.Screening)
	require.Len(t, created.Screening.Skipped, 1)
	assert.Equal(t, "uncovered", created.Screening.Skipped[0].CarrierName)
	assert.NotEmpty(t, created.Screening.Skipped[0].Reasons)
}

// An engine that is unreachable must not silently tender at zero either.
func TestCreateWaterfall_ContractEntryIsSkippedWhenTheEngineFails(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)
	expectNotifications(deps)

	healthy := eligibleCarrier("acme")
	other := eligibleCarrier("other")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:     pulid.MustNew("rgd_"),
		Status: "Active",
		Entries: []*tender.RoutingGuideEntry{
			contractEntry(healthy.ID, 1),
			guideEntry(other.ID, 2),
		},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy, other}, nil)

	deps.svc.rateEngine = &stubRateEngine{err: assert.AnError}

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.Len(t, created.Offers, 1)
	assert.Equal(t, other.ID, created.Offers[0].CarrierID)
	require.Len(t, created.Screening.Skipped, 1)
}

// A deployment without the engine wired keeps working: a contract-priced entry
// falls back to the rate the guide carries rather than refusing to tender.
func TestCreateWaterfall_ContractEntryFallsBackWhenNoEngineIsWired(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)
	expectNotifications(deps)

	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:      pulid.MustNew("rgd_"),
		Status:  "Active",
		Entries: []*tender.RoutingGuideEntry{contractEntry(healthy.ID, 1)},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{healthy}, nil)

	deps.svc.rateEngine = nil

	created, err := deps.svc.CreateWaterfall(t.Context(), &CreateWaterfallTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
	})

	require.NoError(t, err)
	require.Len(t, created.Offers, 1)
	assert.True(t, created.Offers[0].Rate.Equal(decimal.NewFromInt(1200)))
}
