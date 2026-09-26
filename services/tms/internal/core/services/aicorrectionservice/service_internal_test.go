package aicorrectionservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testOrg      = pulid.ID("org_test")
	testBU       = pulid.ID("bu_test")
	testDraft    = pulid.ID("dsd_test")
	testDocument = pulid.ID("doc_test")
	testShipment = pulid.ID("shp_test")
	testUser     = pulid.ID("usr_test")
	testProvider = pulid.ID("aip_test")
	capturedAt   = int64(1_790_000_000)
)

type fakeStore struct {
	saved       *aicorrection.Correction
	upsertErr   error
	purgeCalls  []repositories.PurgeAICorrectionsRequest
	purgeCounts []int64
}

func (f *fakeStore) Upsert(
	_ context.Context,
	entity *aicorrection.Correction,
) (*aicorrection.Correction, error) {
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	f.saved = entity
	return entity, nil
}

func (f *fakeStore) PurgeBefore(
	_ context.Context,
	req repositories.PurgeAICorrectionsRequest,
) (int64, error) {
	f.purgeCalls = append(f.purgeCalls, req)
	if len(f.purgeCounts) == 0 {
		return 0, nil
	}
	next := f.purgeCounts[0]
	f.purgeCounts = f.purgeCounts[1:]
	return next, nil
}

type fakeShipments struct {
	shipment *shipment.Shipment
	err      error
	req      *repositories.GetShipmentByIDRequest
}

func (f *fakeShipments) GetByID(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.req = req
	return f.shipment, f.err
}

type fakeContents struct {
	content *documentcontent.Content
	err     error
}

func (f *fakeContents) GetByDocumentID(
	_ context.Context,
	_ pulid.ID,
	_ pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return f.content, f.err
}

type fakeExtractions struct {
	extraction *documentaiextraction.Extraction
	err        error
	req        repositories.GetDocumentAIExtractionRequest
}

func (f *fakeExtractions) GetByDocumentExtractedAt(
	_ context.Context,
	req repositories.GetDocumentAIExtractionRequest,
) (*documentaiextraction.Extraction, error) {
	f.req = req
	return f.extraction, f.err
}

type fakeRetention struct {
	settings *tenant.DataRetention
	err      error
}

func (f *fakeRetention) Get(
	_ context.Context,
	_ repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return f.settings, f.err
}

type harness struct {
	svc         *Service
	store       *fakeStore
	shipments   *fakeShipments
	contents    *fakeContents
	extractions *fakeExtractions
	retention   *fakeRetention
}

func newHarness(shp *shipment.Shipment) *harness {
	extractedAt := int64(1_789_999_000)
	h := &harness{
		store:     &fakeStore{},
		shipments: &fakeShipments{shipment: shp},
		contents: &fakeContents{
			content: &documentcontent.Content{LastExtractedAt: &extractedAt},
		},
		extractions: &fakeExtractions{
			extraction: &documentaiextraction.Extraction{
				Status:     documentaiextraction.StatusApplied,
				Model:      "open-weights-extractor",
				ProviderID: testProvider,
			},
		},
		retention: &fakeRetention{},
	}
	h.svc = &Service{
		l:           zap.NewNop(),
		repo:        h.store,
		shipments:   h.shipments,
		contents:    h.contents,
		extractions: h.extractions,
		retention:   h.retention,
		now:         func() int64 { return capturedAt },
	}
	return h
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrg, BuID: testBU}
}

func field(value, source string, confidence float64) map[string]any {
	return map[string]any{"value": value, "source": source, "confidence": confidence}
}

func rateConfirmationDraft() *documentshipmentdraft.DocumentShipmentDraft {
	return &documentshipmentdraft.DocumentShipmentDraft{
		ID:             testDraft,
		DocumentID:     testDocument,
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		Status:         documentshipmentdraft.StatusReady,
		DocumentKind:   "RateConfirmation",
		Confidence:     0.86,
		DraftData: map[string]any{
			"kind":                "RateConfirmation",
			"overallConfidence":   0.86,
			"providerFingerprint": "Acme Logistics",
			"fields": map[string]any{
				"referenceNumber": field("LD-10442", "ai", 0.95),
				"rate":            field("$2,450.00", "ai", 0.93),
				"weight":          field("42,000 lbs", "deterministic", 0.8),
				"pieceCount":      field("24", "ai", 0.7),
				"shipper":         field("Northwind Foods", "ai", 0.9),
				"consignee":       field("Contoso Grocery", "ai", 0.88),
				"pickupWindow":    field("03/15/2026 08:00-10:00", "ai", 0.9),
				"deliveryWindow":  field("03/17/2026", "ai", 0.9),
				"commodity":       field("Frozen vegetables", "ai", 0.6),
				"signature":       field("J. Smith", "ai", 0.5),
			},
			"stops": []any{
				map[string]any{
					"sequence": 0, "role": "pickup", "name": "Northwind Foods",
					"addressLine1": "100 Industrial Pkwy", "city": "Dallas", "state": "TX",
					"postalCode": "75201", "date": "03/15/2026", "appointmentRequired": true,
					"source": "ai", "confidence": 0.9,
				},
				map[string]any{
					"sequence": 1, "role": "delivery", "name": "Contoso Grocery",
					"addressLine1": "9 Market St", "city": "Tulsa", "state": "OK",
					"postalCode": "74103", "date": "03/17/2026", "appointmentRequired": false,
					"source": "ai", "confidence": 0.85,
				},
			},
		},
	}
}

func stopAt(
	stopType shipment.StopType,
	seq int64,
	loc *location.Location,
	at time.Time,
	appointment bool,
) *shipment.Stop {
	scheduleType := shipment.StopScheduleTypeOpen
	if appointment {
		scheduleType = shipment.StopScheduleTypeAppointment
	}
	return &shipment.Stop{
		Type:                 stopType,
		Sequence:             seq,
		ScheduleType:         scheduleType,
		ScheduledWindowStart: at.Unix(),
		Location:             loc,
	}
}

func confirmedShipment() *shipment.Shipment {
	chicago := time.FixedZone("CST", -6*60*60)
	weight := int64(42000)
	pieces := int64(26)
	return &shipment.Shipment{
		ID:                  testShipment,
		BOL:                 "LD-10442",
		FreightChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("2450")),
		Weight:              &weight,
		Pieces:              &pieces,
		Commodities: []*shipment.ShipmentCommodity{
			{Commodity: &commodity.Commodity{Name: "Frozen Vegetables"}},
		},
		Moves: []*shipment.ShipmentMove{
			{
				Sequence: 0,
				Stops: []*shipment.Stop{
					stopAt(shipment.StopTypeDelivery, 1, &location.Location{
						Name: "Contoso Grocery DC", AddressLine1: "9 Market Street",
						City: "Tulsa", PostalCode: "74103-2201",
						State: &usstate.UsState{Abbreviation: "OK"},
					}, time.Date(2026, time.March, 18, 9, 0, 0, 0, chicago), false),
					stopAt(shipment.StopTypePickup, 0, &location.Location{
						Name: "Northwind Foods", AddressLine1: "100 Industrial Pkwy",
						City: "Dallas", PostalCode: "75201",
						State:    &usstate.UsState{Abbreviation: "TX"},
						Timezone: "America/Chicago",
					}, time.Date(2026, time.March, 15, 8, 0, 0, 0, chicago), true),
				},
			},
		},
	}
}

func draftFields(
	t *testing.T,
	draft *documentshipmentdraft.DocumentShipmentDraft,
) map[string]any {
	t.Helper()
	fields, ok := draft.DraftData["fields"].(map[string]any)
	require.True(t, ok)
	return fields
}

func resultsByKey(results []aicorrection.FieldResult) map[string]aicorrection.FieldResult {
	out := make(map[string]aicorrection.FieldResult, len(results))
	for i := range results {
		out[results[i].Key] = results[i]
	}
	return out
}

func capture(t *testing.T, h *harness, draft *documentshipmentdraft.DocumentShipmentDraft) *aicorrection.Correction {
	t.Helper()
	got, err := h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:        draft,
		ShipmentID:   testShipment,
		CapturedByID: testUser,
		TenantInfo:   tenantInfo(),
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	return got
}

func TestCaptureShipmentDraft_ScoresEachFieldAgainstTheConfirmedShipment(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())
	got := capture(t, h, rateConfirmationDraft())

	require.Same(t, got, h.store.saved)
	require.NotNil(t, h.shipments.req)
	assert.True(t, h.shipments.req.ExpandShipmentDetails)

	results := resultsByKey(got.FieldResults)
	cases := map[string]aicorrection.Outcome{
		"referenceNumber":                     aicorrection.OutcomeCorrect,
		"rate":                                aicorrection.OutcomeCorrect,
		"weight":                              aicorrection.OutcomeCorrect,
		"pieceCount":                          aicorrection.OutcomeCorrected,
		"commodity":                           aicorrection.OutcomeCorrect,
		"shipper":                             aicorrection.OutcomeCorrect,
		"consignee":                           aicorrection.OutcomeCorrect,
		"pickupWindow":                        aicorrection.OutcomeCorrect,
		"deliveryWindow":                      aicorrection.OutcomeCorrected,
		"stops.pickup[0].name":                aicorrection.OutcomeCorrect,
		"stops.pickup[0].addressLine1":        aicorrection.OutcomeCorrect,
		"stops.pickup[0].date":                aicorrection.OutcomeCorrect,
		"stops.pickup[0].appointmentRequired": aicorrection.OutcomeCorrect,
		"stops.delivery[0].addressLine1":      aicorrection.OutcomeCorrected,
		"stops.delivery[0].postalCode":        aicorrection.OutcomeCorrect,
		"stops.delivery[0].state":             aicorrection.OutcomeCorrect,
		"stops.delivery[0].date":              aicorrection.OutcomeCorrected,
	}
	for key, want := range cases {
		result, ok := results[key]
		if assert.True(t, ok, key) {
			assert.Equal(t, want, result.Outcome, key)
		}
	}
	_, scoredSignature := results["signature"]
	assert.False(t, scoredSignature, "a field with nothing to confirm it is not scored")
	assert.Equal(t, "J. Smith", got.Predicted.Fields["signature"])

	assert.Equal(t, "24", results["pieceCount"].Predicted)
	assert.Equal(t, "26", results["pieceCount"].Confirmed)
	assert.Equal(t, "ai", results["pieceCount"].Source)
	assert.InDelta(t, 0.7, results["pieceCount"].Confidence, 1e-9)
	assert.Equal(t, "2026-03-15", got.Confirmed.Fields["pickupWindow"])
	assert.Equal(t, "America/Chicago", got.Confirmed.Stops[0].Timezone)
}

func TestCaptureShipmentDraft_RecordsProvenanceAndTallies(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())
	got := capture(t, h, rateConfirmationDraft())

	assert.Equal(t, aicorrection.TaskShipmentDraftExtraction, got.Task)
	assert.Equal(t, aicorrection.SourceDocumentShipmentDraft, got.SourceType)
	assert.Equal(t, testDraft, got.SourceID)
	require.NotNil(t, got.DocumentID)
	assert.Equal(t, testDocument, *got.DocumentID)
	assert.Equal(t, aicorrection.SubjectShipment, got.SubjectType)
	assert.Equal(t, testShipment, got.SubjectID)
	assert.Equal(t, testUser, got.CapturedByID)
	assert.Equal(t, testOrg, got.OrganizationID)
	assert.Equal(t, testBU, got.BusinessUnitID)
	assert.Equal(t, "RateConfirmation", got.DocumentKind)
	assert.Equal(t, "Acme Logistics", got.DocumentFingerprint)
	assert.Equal(t, "open-weights-extractor", got.ExtractionModel)
	require.NotNil(t, got.ExtractionProviderID)
	assert.Equal(t, testProvider, *got.ExtractionProviderID)
	assert.InDelta(t, 0.86, got.PredictedConfidence, 1e-9)
	assert.Equal(t, capturedAt, got.CapturedAt)
	assert.Equal(t, testDocument, h.extractions.req.DocumentID)

	tally := aicorrection.TallyResults(got.FieldResults)
	assert.Equal(t, tally.Correct, got.CorrectCount)
	assert.Equal(t, tally.Corrected, got.CorrectedCount)
	assert.Equal(t, tally.Missed, got.MissedCount)
	assert.Equal(t, got.CorrectCount+got.CorrectedCount+got.MissedCount, got.ScoredCount)
	assert.Positive(t, got.CorrectCount)
	assert.Positive(t, got.CorrectedCount)
}

func TestCaptureShipmentDraft_MissedAndUnconfirmedFields(t *testing.T) {
	t.Parallel()

	shp := confirmedShipment()
	shp.BOL = ""
	shp.Moves[0].Stops = append(shp.Moves[0].Stops, stopAt(
		shipment.StopTypeDelivery, 2,
		&location.Location{Name: "Second Drop", City: "Wichita"},
		time.Date(2026, time.March, 19, 9, 0, 0, 0, time.UTC), false,
	))
	draft := rateConfirmationDraft()
	fields := draftFields(t, draft)
	delete(fields, "weight")

	got := capture(t, newHarness(shp), draft)
	results := resultsByKey(got.FieldResults)

	assert.Equal(t, aicorrection.OutcomeUnconfirmed, results["referenceNumber"].Outcome)
	assert.Equal(t, aicorrection.OutcomeMissed, results["weight"].Outcome)
	assert.Equal(t, "42000", results["weight"].Confirmed)
	assert.Equal(t, aicorrection.OutcomeMissed, results["stops.delivery[1].name"].Outcome)
	assert.Equal(t, aicorrection.OutcomeMissed, results["stops.delivery[1].city"].Outcome)
	assert.Equal(t, "Second Drop", results["consignee"].Confirmed)
	assert.Equal(t, aicorrection.OutcomeCorrected, results["consignee"].Outcome)
}

func TestCaptureShipmentDraft_UnreadableValuesAreUnscored(t *testing.T) {
	t.Parallel()

	draft := rateConfirmationDraft()
	fields := draftFields(t, draft)
	fields["rate"] = field("call for rate", "ai", 0.4)
	fields["pickupWindow"] = field("FCFS", "ai", 0.4)

	got := capture(t, newHarness(confirmedShipment()), draft)
	results := resultsByKey(got.FieldResults)

	assert.Equal(t, aicorrection.OutcomeUnscored, results["rate"].Outcome)
	assert.Equal(t, aicorrection.OutcomeUnscored, results["pickupWindow"].Outcome)
	assert.Equal(t, 2, got.UnscoredCount)
}

func TestCaptureShipmentDraft_WithoutAnAIExtractionLeavesTheModelEmpty(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())
	h.extractions.extraction.Status = documentaiextraction.StatusFailed

	got := capture(t, h, rateConfirmationDraft())
	assert.Empty(t, got.ExtractionModel)
	assert.Nil(t, got.ExtractionProviderID)

	h = newHarness(confirmedShipment())
	h.contents.err = errortypes.NewNotFoundError("content not found")
	got = capture(t, h, rateConfirmationDraft())
	assert.Empty(t, got.ExtractionModel)

	h = newHarness(confirmedShipment())
	h.contents.content.LastExtractedAt = nil
	got = capture(t, h, rateConfirmationDraft())
	assert.Empty(t, got.ExtractionModel)
}

func TestCaptureShipmentDraft_SkipsADraftWithNothingPredicted(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())
	draft := rateConfirmationDraft()
	draft.DraftData = map[string]any{"fields": map[string]any{"rate": field("  ", "ai", 0.1)}}

	got, err := h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:        draft,
		ShipmentID:   testShipment,
		CapturedByID: testUser,
		TenantInfo:   tenantInfo(),
	})
	require.ErrorIs(t, err, aicorrection.ErrNothingPredicted)
	assert.Nil(t, got)
	assert.Nil(t, h.store.saved)
	assert.Nil(t, h.shipments.req, "the shipment is not read when there is nothing to compare")
}

func TestCaptureShipmentDraft_RefusesADraftFromAnotherTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())
	draft := rateConfirmationDraft()
	draft.OrganizationID = pulid.ID("org_other")

	_, err := h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:        draft,
		ShipmentID:   testShipment,
		CapturedByID: testUser,
		TenantInfo:   tenantInfo(),
	})
	require.Error(t, err)
	assert.Nil(t, h.store.saved)
	assert.Nil(t, h.shipments.req)
}

func TestCaptureShipmentDraft_RequiresADraftAndAShipment(t *testing.T) {
	t.Parallel()

	h := newHarness(confirmedShipment())

	_, err := h.svc.CaptureShipmentDraft(t.Context(), nil)
	require.Error(t, err)

	_, err = h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:      rateConfirmationDraft(),
		TenantInfo: tenantInfo(),
	})
	require.Error(t, err)
}

func TestCaptureShipmentDraft_PropagatesShipmentAndStoreFailures(t *testing.T) {
	t.Parallel()

	h := newHarness(nil)
	h.shipments.err = errors.New("database unavailable")
	_, err := h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:        rateConfirmationDraft(),
		ShipmentID:   testShipment,
		CapturedByID: testUser,
		TenantInfo:   tenantInfo(),
	})
	require.ErrorContains(t, err, "database unavailable")

	h = newHarness(confirmedShipment())
	h.store.upsertErr = errors.New("write failed")
	_, err = h.svc.CaptureShipmentDraft(t.Context(), &services.CaptureShipmentDraftCorrectionRequest{
		Draft:        rateConfirmationDraft(),
		ShipmentID:   testShipment,
		CapturedByID: testUser,
		TenantInfo:   tenantInfo(),
	})
	require.ErrorContains(t, err, "write failed")
}

func TestPurgeExpired_UsesTheOrganizationsRetentionInBatches(t *testing.T) {
	t.Parallel()

	h := newHarness(nil)
	h.retention.settings = &tenant.DataRetention{AICorrectionRetentionPeriod: 60}
	h.store.purgeCounts = []int64{purgeBatchSize, 12}

	purged, err := h.svc.PurgeExpired(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: tenantInfo(),
		Now:        capturedAt,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(purgeBatchSize+12), purged)
	require.Len(t, h.store.purgeCalls, 2)
	assert.Equal(t, capturedAt-60*86400, h.store.purgeCalls[0].Before)
	assert.Equal(t, tenantInfo(), h.store.purgeCalls[0].TenantInfo)
}

func TestPurgeExpired_DefaultsWhenNoRetentionIsSaved(t *testing.T) {
	t.Parallel()

	h := newHarness(nil)
	h.retention.err = errortypes.NewNotFoundError("data retention not found")

	_, err := h.svc.PurgeExpired(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: tenantInfo(),
	})
	require.NoError(t, err)
	require.Len(t, h.store.purgeCalls, 1)
	assert.Equal(
		t,
		capturedAt-int64(tenant.DefaultAICorrectionRetentionDays)*86400,
		h.store.purgeCalls[0].Before,
	)

	h = newHarness(nil)
	h.retention.err = errors.New("database unavailable")
	_, err = h.svc.PurgeExpired(t.Context(), services.PurgeExpiredAICorrectionsRequest{
		TenantInfo: tenantInfo(),
	})
	require.Error(t, err)
	assert.Empty(t, h.store.purgeCalls)
}
