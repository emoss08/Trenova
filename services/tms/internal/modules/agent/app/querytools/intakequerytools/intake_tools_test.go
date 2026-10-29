package intakequerytools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDrafts struct {
	draft      *documentshipmentdraft.DocumentShipmentDraft
	lastTenant pagination.TenantInfo
	lastID     pulid.ID
}

func (f *fakeDrafts) GetShipmentDraft(
	_ context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	f.lastID = documentID
	f.lastTenant = tenantInfo

	return f.draft, nil
}

func TestGetShipmentDraft_ReadsFieldsStopsAndWhatNeedsAPerson(t *testing.T) {
	t.Parallel()

	docID := pulid.MustNew("doc_")
	drafts := &fakeDrafts{draft: &documentshipmentdraft.DocumentShipmentDraft{
		DocumentID:   docID,
		Status:       documentshipmentdraft.StatusReady,
		DocumentKind: "rate_confirmation",
		Confidence:   0.82,
		DraftData: map[string]any{
			"reviewStatus":  "NeedsReview",
			"missingFields": []any{"weight"},
			"fields": map[string]any{
				"bol": map[string]any{"label": "BOL", "value": "BOL-778", "confidence": 0.97},
				"rate": map[string]any{
					"label": "Rate", "value": "1450.00", "confidence": 0.41, "reviewRequired": true,
				},
			},
			"stops": []any{
				map[string]any{
					"sequence": 0, "role": "pickup", "city": "Dallas", "state": "TX", "date": "2026-10-02", "confidence": 0.9,
				},
				map[string]any{
					"sequence": 1, "role": "delivery", "city": "Houston", "state": "TX", "confidence": 0.7,
					"appointmentRequired": true,
				},
			},
		},
	}}
	tool := NewGetShipmentDraftTool(drafts)
	assert.Equal(t, permission.ResourceDocument, tool.Policy().Resource)

	params := querytooltest.TestParams(map[string]any{"documentId": docID.String()})
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	view, ok := result.(*draftView)
	require.True(t, ok)
	assert.Equal(t, docID, drafts.lastID)
	assert.Equal(t, params.OrganizationID, drafts.lastTenant.OrgID)
	assert.Equal(t, "Ready", view.Status)
	assert.Equal(t, "NeedsReview", view.ReviewStatus)
	assert.Equal(t, []string{"weight"}, view.MissingFields)
	assert.Equal(t, "BOL-778", view.Fields["bol"].Value)
	assert.True(t, view.Fields["rate"].ReviewRequired)
	require.Len(t, view.Stops, 2)
	assert.Equal(t, "delivery", view.Stops[1].Role)
	assert.True(t, view.Stops[1].AppointmentRequired)
	assert.Contains(t, view.Note, "confirm them")
}

func TestGetShipmentDraft_SaysWhenTheDraftWasAlreadyUsed(t *testing.T) {
	t.Parallel()

	attached := pulid.MustNew("shp_")
	drafts := &fakeDrafts{draft: &documentshipmentdraft.DocumentShipmentDraft{
		DocumentID:         pulid.MustNew("doc_"),
		Status:             documentshipmentdraft.StatusReady,
		AttachedShipmentID: &attached,
	}}
	tool := NewGetShipmentDraftTool(drafts)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": drafts.draft.DocumentID.String()}),
	)
	require.NoError(t, err)

	view := result.(*draftView)
	assert.Equal(t, attached.String(), view.AttachedShipmentID)
	assert.Contains(t, view.Note, "do not create it again")
	assert.NotNil(t, view.Fields, "an empty draft still answers with empty collections, not null")
	assert.NotNil(t, view.Stops)
}

func TestGetShipmentDraft_RejectsAMismatchedActor(t *testing.T) {
	t.Parallel()

	drafts := &fakeDrafts{}
	tool := NewGetShipmentDraftTool(drafts)
	params := querytooltest.TestParams(map[string]any{"documentId": pulid.MustNew("doc_").String()})
	params.Actor.OrganizationID = pulid.MustNew("org_")

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, querytool.ErrTenantMismatch)
	assert.True(t, drafts.lastID.IsNil())
}

type fakeQuoter struct {
	rated   *ratequoteports.RatedShipment
	lastReq *ratequoteservice.QuoteRequest
	miles   float64
	hydrate func(*shipment.Shipment)
}

func (f *fakeQuoter) Quote(
	_ context.Context,
	req *ratequoteservice.QuoteRequest,
) (*ratequoteports.RatedShipment, error) {
	f.lastReq = req
	if f.hydrate != nil && req.Shipment != nil {
		f.hydrate(req.Shipment)
	}
	if f.miles > 0 && req.Shipment != nil {
		for _, move := range req.Shipment.Moves {
			miles := f.miles
			move.Distance = &miles
		}
	}

	return f.rated, nil
}

type fakeLocations struct {
	byID map[pulid.ID]*location.Location
}

func (f *fakeLocations) GetByIDs(
	_ context.Context,
	req locationports.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	out := make([]*location.Location, 0, len(req.LocationIDs))
	for _, id := range req.LocationIDs {
		if loc, ok := f.byID[id]; ok {
			out = append(out, loc)
		}
	}

	return out, nil
}

func TestQuoteShipment_BuildsTheLaneFromLocationsAndNeverPersists(t *testing.T) {
	t.Parallel()

	origin := &location.Location{
		ID:         pulid.MustNew("loc_"),
		Name:       "Dallas Yard",
		City:       "Dallas",
		PostalCode: "75201",
		State:      &usstate.UsState{Abbreviation: "TX"},
	}
	destination := &location.Location{
		ID:         pulid.MustNew("loc_"),
		Name:       "Houston DC",
		City:       "Houston",
		PostalCode: "77001",
		State:      &usstate.UsState{Abbreviation: "TX"},
	}
	agreementID := pulid.MustNew("ragr_")
	quoter := &fakeQuoter{miles: 240, rated: &ratequoteports.RatedShipment{
		Amount: decimal.NewFromInt(1450), Total: decimal.NewFromInt(1450), Currency: "USD",
		Outcome: ratequote.OutcomeRated, AgreementID: &agreementID,
		TariffInputs: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.RequireFromString("2.9"), "$/mi"),
		},
	}}
	tool := NewQuoteShipmentTool(quoter, &fakeLocations{byID: map[pulid.ID]*location.Location{
		origin.ID: origin, destination.ID: destination,
	}})
	assert.Equal(t, permission.ResourceRateQuote, tool.Policy().Resource)

	customerID := pulid.MustNew("cust_")
	serviceTypeID := pulid.MustNew("st_")
	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParamsIn("America/Chicago", map[string]any{
			"customerId":    customerID.String(),
			"serviceTypeId": serviceTypeID.String(),
			"stops": []any{
				map[string]any{
					"locationId": origin.ID.String(),
					"type":       "Pickup",
					"date":       "2026-10-02",
				},
				map[string]any{"locationId": destination.ID.String(), "type": "Delivery"},
			},
			"pieces": 12,
			"weight": 18_000,
		}),
	)
	require.NoError(t, err)

	view := result.(*quoteView)
	assert.Equal(t, "Rated", view.Outcome)
	assert.Equal(t, "1450", view.Amount.String())
	assert.Equal(t, "1450", view.Total.String())
	assert.Equal(t, map[string]string{"rate": "$2.9/mi"}, view.Inputs)
	assert.Equal(t, agreementID.String(), view.AgreementID)
	assert.Equal(t, []string{"Dallas, TX, 75201", "Houston, TX, 77001"}, view.Lane)
	assert.InDelta(t, 240.0, view.TotalDistance, 0.001)
	assert.Empty(t, view.Note)

	require.NotNil(t, quoter.lastReq)
	assert.False(t, quoter.lastReq.Persist, "a quote from an agent is a look, not a record")
	sent := quoter.lastReq.Shipment
	assert.Equal(t, customerID, sent.CustomerID)
	assert.Equal(t, serviceTypeID, sent.ServiceTypeID)
	require.Len(t, sent.Moves, 1)
	require.Len(t, sent.Moves[0].Stops, 2)
	assert.Same(
		t,
		origin,
		sent.Moves[0].Stops[0].Location,
		"the engine reads the lane off the loaded locations",
	)
	assert.EqualValues(t, 18_000, *sent.Weight)
	assert.Equal(t, sent.Moves[0].Stops[0].ScheduledWindowStart, quoter.lastReq.AsOf,
		"rated as of the first stop's date, read in the organization's zone")
	assert.Equal(t, int64(1_790_000_000)/int64(1_790_000_000), int64(1))
}

func TestQuoteShipment_RefusesAnUnknownLocationAndAThinLane(t *testing.T) {
	t.Parallel()

	tool := NewQuoteShipmentTool(
		&fakeQuoter{},
		&fakeLocations{byID: map[pulid.ID]*location.Location{}},
	)
	base := map[string]any{
		"customerId":    pulid.MustNew("cust_").String(),
		"serviceTypeId": pulid.MustNew("st_").String(),
	}

	one := map[string]any{
		"stops": []any{
			map[string]any{"locationId": pulid.MustNew("loc_").String(), "type": "Pickup"},
		},
	}
	for k, v := range base {
		one[k] = v
	}
	_, err := tool.Query(t.Context(), querytooltest.TestParams(one))
	require.ErrorContains(t, err, "between 2 and")

	unknown := map[string]any{"stops": []any{
		map[string]any{"locationId": pulid.MustNew("loc_").String(), "type": "Pickup"},
		map[string]any{"locationId": pulid.MustNew("loc_").String(), "type": "Delivery"},
	}}
	for k, v := range base {
		unknown[k] = v
	}
	_, err = tool.Query(t.Context(), querytooltest.TestParams(unknown))
	require.ErrorContains(t, err, "not a location in this organization")
}

func TestQuoteShipment_SaysWhenNothingPricedTheLane(t *testing.T) {
	t.Parallel()

	formulaID := pulid.MustNew("ft_")
	fallback := quoteViewOf(&ratequoteports.RatedShipment{
		Outcome: ratequote.OutcomeFormulaFallback, Amount: decimal.NewFromInt(900), FormulaTemplateID: &formulaID,
	}, nil)
	assert.Contains(t, fallback.Note, "formula priced it")

	none := quoteViewOf(&ratequoteports.RatedShipment{Outcome: ratequote.OutcomeNoRateFound}, nil)
	assert.Contains(t, none.Note, "not a rate")
}

type fakeShopper struct {
	result  *ratequoteports.ShopResult
	lastReq *ratequoteservice.ShopRequest
}

func (f *fakeShopper) Shop(
	_ context.Context,
	req *ratequoteservice.ShopRequest,
) (*ratequoteports.ShopResult, error) {
	f.lastReq = req

	return f.result, nil
}

func shopResultWithMargin(carrierA, carrierB pulid.ID) *ratequoteports.ShopResult {
	return &ratequoteports.ShopResult{
		Strategy:  ratequoteports.ShopStrategyBestMargin,
		SellTotal: decimal.NewNullDecimal(decimal.NewFromInt(2000)),
		Options: []*ratequoteports.ShopOption{
			{
				Rank:        1,
				CarrierID:   carrierA,
				CarrierName: "Blue Ridge",
				Outcome:     ratequote.OutcomeRated,
				Cost:        decimal.NewFromInt(1500),
				Currency:    "USD",
				GuideRank:   2,
				Margin: &ratetypes.MarginVerdict{
					Amount:  decimal.NewFromInt(500),
					Percent: decimal.NewFromInt(25),
				},
			},
			{
				Rank: 2, CarrierID: carrierB, CarrierName: "No Contract Inc",
				Outcome: ratequote.OutcomeNoRateFound, Note: "no contract",
			},
		},
		Warnings: []string{"No Contract Inc has no agreement on this lane"},
	}
}

func TestShopCarriers_RanksOptionsWithMarginAndPassesTheShortlist(t *testing.T) {
	t.Parallel()

	carrierA := pulid.MustNew("carr_")
	carrierB := pulid.MustNew("carr_")
	shopper := &fakeShopper{result: shopResultWithMargin(carrierA, carrierB)}
	tool := NewShopCarriersTool(shopper, &shopCarriers{carriers: []*carrier.Carrier{
		{
			ID:               carrierA,
			Status:           carrier.StatusActive,
			ComplianceStatus: carrier.ComplianceStatusQualified,
		},
		{
			ID:               carrierB,
			Status:           carrier.StatusActive,
			ComplianceStatus: carrier.ComplianceStatusPending,
		},
	}}, querytooltest.MarginsReader())

	shipmentID := pulid.MustNew("shp_")
	result, err := tool.Query(t.Context(), querytooltest.ChatParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"strategy":   "BestMargin",
		"carrierIds": []any{carrierA.String(), carrierB.String()},
		"limit":      2,
	}, ""))
	require.NoError(t, err)

	view := result.(*shopView)
	require.Len(t, view.Options, 2)
	assert.True(t, view.Options[0].Priced)
	require.NotNil(t, view.Options[0].MarginPercent)
	assert.Equal(t, "25", view.Options[0].MarginPercent.String())
	assert.False(t, view.Options[1].Priced)
	assert.Nil(t, view.Options[1].MarginPercent, "an option nobody priced has no margin")
	assert.Equal(t, "Qualified", view.Options[0].Compliance)
	assert.Equal(t, "Active", view.Options[0].Status)
	assert.Equal(t, "Pending", view.Options[1].Compliance,
		"a carrier not yet qualified says so on its option")
	assert.Equal(t, "2000", view.SellTotal.String())
	assert.Len(t, view.Warnings, 1)
	assert.Contains(t, view.Note, "Flat baseRate",
		"a priced shop says its cost is the rate a carrier assignment takes")

	assert.Equal(t, shipmentID, shopper.lastReq.ShipmentID)
	assert.Equal(t, []pulid.ID{carrierA, carrierB}, shopper.lastReq.CarrierIDs)
	assert.Equal(t, 2, shopper.lastReq.Limit)
	assert.False(t, shopper.lastReq.Persist)
}

func TestShopCarriers_ShowsNoMarginToAPersonWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	carrierA := pulid.MustNew("carr_")
	carrierB := pulid.MustNew("carr_")
	shopper := &fakeShopper{result: shopResultWithMargin(carrierA, carrierB)}
	tool := NewShopCarriersTool(shopper, &shopCarriers{}, &querytooltest.FakePermissions{
		Readable: map[string]*permissionports.ResourcePermissionDetail{},
	})
	shipmentID := pulid.MustNew("shp_")

	_, err := tool.Query(t.Context(), querytooltest.ChatParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"strategy":   "BestMargin",
	}, ""))
	require.Error(t, err, "ranking by margin would tell them which carrier leaves the most")
	assert.Nil(t, shopper.lastReq, "nothing is shopped for a refused strategy")

	result, err := tool.Query(t.Context(), querytooltest.ChatParams(map[string]any{
		"shipmentId": shipmentID.String(),
	}, ""))
	require.NoError(t, err)
	view := result.(*shopView)
	assert.Nil(t, view.SellTotal, "the sell total beside each cost gives the margin away")
	require.Len(t, view.Options, 2)
	assert.Nil(t, view.Options[0].MarginAmount)
	assert.Nil(t, view.Options[0].MarginPercent)
	assert.Nil(t, view.Options[0].BelowFloor)
	assert.Equal(t, "1500", view.Options[0].Cost.String(), "the carrier's cost still shows")
}

func TestShopCarriers_HoldsAnAgentToItsDataAccess(t *testing.T) {
	t.Parallel()

	shopFor := func() agenttoolports.AgentQueryTool {
		shopper := &fakeShopper{
			result: shopResultWithMargin(pulid.MustNew("carr_"), pulid.MustNew("carr_")),
		}
		return NewShopCarriersTool(shopper, &shopCarriers{}, querytooltest.MarginsReader())
	}

	result, err := shopFor().Query(t.Context(), querytooltest.ChatParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
	}, permission.SensitivityInternal))
	require.NoError(t, err)
	view := result.(*shopView)
	assert.Nil(t, view.Options[0].MarginPercent,
		"an agent whose data access stops at Internal is shown no margin for its person")

	result, err = shopFor().Query(t.Context(), querytooltest.ChatParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	view = result.(*shopView)
	require.NotNil(t, view.Options[0].MarginPercent)
	assert.Equal(t, "25", view.Options[0].MarginPercent.String())
}

type shopCarriers struct{ carriers []*carrier.Carrier }

func (s *shopCarriers) GetByIDs(
	context.Context,
	carrierports.GetCarriersByIDsRequest,
) ([]*carrier.Carrier, error) {
	return s.carriers, nil
}

func TestQuoteShipment_LabelsTheLaneWithWhatTheQuoteLoaded(t *testing.T) {
	t.Parallel()

	texas := &usstate.UsState{ID: pulid.MustNew("st_"), Abbreviation: "TX"}
	origin := &location.Location{
		ID: pulid.MustNew("loc_"), City: "Dallas", PostalCode: "75201", StateID: texas.ID,
	}
	destination := &location.Location{
		ID: pulid.MustNew("loc_"), City: "Houston", PostalCode: "77001", StateID: texas.ID,
	}
	quoter := &fakeQuoter{
		rated: &ratequoteports.RatedShipment{Outcome: ratequote.OutcomeNoRateFound},
		hydrate: func(entity *shipment.Shipment) {
			for _, stop := range entity.Moves[0].Stops {
				stop.Location.State = texas
			}
		},
	}
	tool := NewQuoteShipmentTool(quoter, &fakeLocations{byID: map[pulid.ID]*location.Location{
		origin.ID: origin, destination.ID: destination,
	}})

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParamsIn("America/Chicago", map[string]any{
			"customerId":    pulid.MustNew("cust_").String(),
			"serviceTypeId": pulid.MustNew("st_").String(),
			"stops": []any{
				map[string]any{"locationId": origin.ID.String(), "type": "Pickup"},
				map[string]any{"locationId": destination.ID.String(), "type": "Delivery"},
			},
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"Dallas, TX, 75201", "Houston, TX, 77001"}, result.(*quoteView).Lane)
}
