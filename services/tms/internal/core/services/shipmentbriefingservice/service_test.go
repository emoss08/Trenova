package shipmentbriefingservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func joined(segments []services.ShipmentBriefingSegment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.Text)
	}
	return b.String()
}

func TestDeterministicWording(t *testing.T) {
	t.Parallel()

	segments := Deterministic(Facts{
		Facts: shipmentbrief.Facts{
			DeliveringToday: 14,
			Moving:          9,
			Late:            2,
			Uncovered:       3,
			LateReason:      "Shipper delay",
		},
		OperationType: tenant.OperationTypeAsset,
	})
	assert.Equal(
		t,
		"14 loads deliver today, 9 loads are moving on schedule and 2 loads are late, mostly shipper delay. 3 loads still need a driver.",
		joined(segments),
	)
	require.NotNil(t, segments[0].Filter)
	assert.Equal(t, shipment.QuickFilterDeliveringToday, *segments[0].Filter)

	single := Deterministic(Facts{
		Facts:         shipmentbrief.Facts{DeliveringToday: 1, Late: 1, Uncovered: 1},
		OperationType: tenant.OperationTypeBrokerage,
	})
	assert.Equal(
		t,
		"1 load delivers today and 1 load is late. 1 load still needs a carrier.",
		joined(single),
	)

	onlyUncovered := Deterministic(Facts{
		Facts:         shipmentbrief.Facts{Uncovered: 2},
		OperationType: tenant.OperationTypeBoth,
	})
	assert.Equal(t, "2 loads still need coverage.", joined(onlyUncovered))

	empty := Deterministic(Facts{})
	assert.Equal(t, "Nothing on the board needs you right now.", joined(empty))
	assert.Nil(t, empty[0].Filter)

	followUps := Deterministic(Facts{
		Facts: shipmentbrief.Facts{
			Moving:      4,
			Uncovered:   1,
			Detention:   2,
			ReadyToBill: 6,
			LowMargin:   3,
		},
		OperationType: tenant.OperationTypeAsset,
	})
	assert.Equal(
		t,
		"4 loads are moving on schedule. 1 load still needs a driver, 2 loads are accruing detention and 6 loads are ready to bill.",
		joined(followUps),
	)
	linkedTo := make([]shipment.QuickFilter, 0, len(followUps))
	for _, segment := range followUps {
		if segment.Filter != nil {
			linkedTo = append(linkedTo, *segment.Filter)
		}
	}
	assert.Equal(t, []shipment.QuickFilter{
		shipment.QuickFilterMoving,
		shipment.QuickFilterUncovered,
		shipment.QuickFilterDetention,
		shipment.QuickFilterReadyToBill,
	}, linkedTo)
}

func TestAcceptNarration(t *testing.T) {
	t.Parallel()

	facts := &Facts{Facts: shipmentbrief.Facts{DeliveringToday: 140, Late: 32, Uncovered: 3}}
	allowed := facts.Filters()
	late := shipment.QuickFilterLate
	moving := shipment.QuickFilterMoving

	good := []services.ShipmentBriefingSegment{
		{Text: "32 late loads", Filter: &late},
		{Text: " lead the day across 140 deliveries."},
	}
	accepted, ok := AcceptNarration(good, facts, allowed)
	require.True(t, ok)
	assert.Len(t, accepted, 2)

	invented := []services.ShipmentBriefingSegment{
		{Text: "Around 400 loads are late", Filter: &late},
	}
	_, ok = AcceptNarration(invented, facts, allowed)
	assert.False(t, ok)

	wrongFilter := []services.ShipmentBriefingSegment{{Text: "Loads are moving", Filter: &moving}}
	_, ok = AcceptNarration(wrongFilter, facts, allowed)
	assert.False(t, ok)

	_, ok = AcceptNarration(nil, facts, allowed)
	assert.False(t, ok)

	_, ok = AcceptNarration(
		[]services.ShipmentBriefingSegment{{Text: strings.Repeat("a", maxNarrationChars+1)}},
		facts,
		allowed,
	)
	assert.False(t, ok)
}

type stubBasis struct{}

func (stubBasis) Resolve(
	context.Context,
	*services.ResolveShipmentQuickFilterBasisRequest,
) (*repositories.ShipmentQuickFilterBasis, error) {
	return &repositories.ShipmentQuickFilterBasis{Now: time.Unix(0, 0), Location: time.UTC}, nil
}

func (stubBasis) Prepare(
	context.Context,
	pagination.TenantInfo,
	*repositories.ShipmentOptions,
) error {
	return nil
}

type stubBoard struct {
	repositories.ShipmentBoardRepository
	counts map[shipment.QuickFilter]int
}

func (b *stubBoard) QuickFilterTotals(
	_ context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) ([]repositories.ShipmentQuickFilterTotal, error) {
	out := make([]repositories.ShipmentQuickFilterTotal, len(req.Filters))
	for i, spec := range req.Filters {
		out[i].Count = b.counts[spec.Filter]
	}
	return out, nil
}

type stubReasons struct{}

func (stubReasons) LeadingLateReason(
	context.Context,
	pagination.TenantInfo,
) (*repositories.ShipmentLateReason, error) {
	return &repositories.ShipmentLateReason{Label: "Weather", Count: 1}, nil
}

type stubOrgs struct {
	repositories.OrganizationRepository
}

func (stubOrgs) GetCapabilities(
	context.Context,
	repositories.GetOrganizationCapabilitiesRequest,
) (*repositories.OrganizationCapabilities, error) {
	return &repositories.OrganizationCapabilities{AssetOperationsEnabled: true}, nil
}

type stubOrgCache struct{ timezone string }

func (c stubOrgCache) GetByID(context.Context, pulid.ID) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: c.timezone}, nil
}

type stubCandidates struct {
	items []*services.ShipmentSuggestion
}

func (c *stubCandidates) Candidates(
	context.Context,
	pagination.TenantInfo,
	string,
) ([]*services.ShipmentSuggestion, error) {
	out := make([]*services.ShipmentSuggestion, 0, len(c.items))
	for _, item := range c.items {
		copied := *item
		out = append(out, &copied)
	}
	return out, nil
}

type memoryBriefs struct {
	rows []*shipmentbrief.Brief
}

func (m *memoryBriefs) Latest(
	_ context.Context,
	req *repositories.GetLatestShipmentBriefRequest,
) (*shipmentbrief.Brief, error) {
	var latest *shipmentbrief.Brief
	for _, row := range m.rows {
		if row.BriefDate != req.BriefDate || row.OrganizationID != req.TenantInfo.OrgID {
			continue
		}
		if latest == nil || row.Generation > latest.Generation {
			latest = row
		}
	}
	return latest, nil
}

func (m *memoryBriefs) Insert(
	ctx context.Context,
	brief *shipmentbrief.Brief,
) (*shipmentbrief.Brief, error) {
	for _, row := range m.rows {
		if row.BriefDate == brief.BriefDate && row.OrganizationID == brief.OrganizationID &&
			row.Generation == brief.Generation {
			return m.Latest(ctx, &repositories.GetLatestShipmentBriefRequest{
				TenantInfo: pagination.TenantInfo{OrgID: brief.OrganizationID},
				BriefDate:  brief.BriefDate,
			})
		}
	}
	m.rows = append(m.rows, brief)
	return brief, nil
}

func (m *memoryBriefs) DeleteBefore(
	context.Context,
	*repositories.DeleteShipmentBriefsBeforeRequest,
) (int, error) {
	return 0, nil
}

type stubCompletion struct {
	services.CompletionService
	text  string
	err   error
	calls *int
}

func (s stubCompletion) CompleteStructured(
	_ context.Context,
	req *services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	if s.calls != nil {
		*s.calls++
	}
	if s.err != nil {
		return nil, s.err
	}
	if req.SchemaName != schemaName {
		return nil, errors.New("no wording in this test")
	}
	return &services.StructuredCompletionResult{Text: s.text, ModelIdentifier: "test-model"}, nil
}

type fixture struct {
	service    *Service
	board      *stubBoard
	briefs     *memoryBriefs
	candidates *stubCandidates
}

func defaultCounts() map[shipment.QuickFilter]int {
	return map[shipment.QuickFilter]int{
		shipment.QuickFilterDeliveringToday: 30,
		shipment.QuickFilterMoving:          12,
		shipment.QuickFilterLate:            0,
		shipment.QuickFilterUncovered:       4,
	}
}

func newFixture(completion services.CompletionService) *fixture {
	f := &fixture{
		board:      &stubBoard{counts: defaultCounts()},
		briefs:     &memoryBriefs{},
		candidates: &stubCandidates{},
	}
	f.service = NewWithDependencies(&Dependencies{
		Board:         f.board,
		Briefing:      stubReasons{},
		Briefs:        f.briefs,
		QuickFilters:  stubBasis{},
		Organizations: stubOrgs{},
		OrgCache:      stubOrgCache{timezone: "America/Chicago"},
		Suggestions:   f.candidates,
		Completion:    completion,
		Logger:        zap.NewNop(),
		Now:           func() time.Time { return time.Unix(1_791_000_000, 0) },
	})
	return f
}

func newService(completion services.CompletionService) *Service {
	return newFixture(completion).service
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
}

func TestBriefingFallsBackWithoutAModel(t *testing.T) {
	t.Parallel()

	svc := newService(stubCompletion{err: services.ErrNoProviderConfigured})
	briefing, err := svc.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
	assert.Equal(t, int64(1_791_000_000), briefing.GeneratedAt)
	assert.Equal(
		t,
		"30 loads deliver today and 12 loads are moving on schedule. 4 loads still need a driver.",
		joined(briefing.Segments),
	)

	broken := newService(stubCompletion{err: errors.New("timeout")})
	briefing, err = broken.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
}

func TestBriefingUsesGuardedNarration(t *testing.T) {
	t.Parallel()

	svc := newService(stubCompletion{
		text: `{"segments":[{"text":"4 loads need a driver","filter":"Uncovered"},{"text":" while 30 deliver today.","filter":""}]}`,
	})
	briefing, err := svc.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.True(t, briefing.Narrated)
	require.Len(t, briefing.Segments, 2)
	assert.Equal(t, shipment.QuickFilterUncovered, *briefing.Segments[0].Filter)
	assert.Nil(t, briefing.Segments[1].Filter)

	lying := newService(stubCompletion{
		text: `{"segments":[{"text":"55 loads are late","filter":"Late"}]}`,
	})
	briefing, err = lying.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
}

func TestBriefingIsWrittenOnceAndThenRead(t *testing.T) {
	t.Parallel()

	calls := 0
	f := newFixture(stubCompletion{
		text:  `{"segments":[{"text":"4 loads need a driver.","filter":"Uncovered"}]}`,
		calls: &calls,
	})

	first, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	require.Len(t, f.briefs.rows, 1)
	stored := f.briefs.rows[0]
	assert.Equal(t, shipmentbrief.TriggerOnDemand, stored.Trigger)
	assert.Equal(t, 1, stored.Generation)
	assert.Equal(t, "test-model", stored.ModelIdentifier)
	assert.Equal(t, "America/Chicago", stored.Facts.Timezone)
	assert.Equal(t, 4, stored.OpenIssues)
	assert.Equal(t, 1, first.Generation)
	modelCalls := calls

	f.board.counts[shipment.QuickFilterMoving] = 99
	second, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.Equal(t, modelCalls, calls, "reading the day's brief asks the model nothing")
	assert.Len(t, f.briefs.rows, 1)
	assert.Equal(t, joined(first.Segments), joined(second.Segments))
}

func TestBriefingIsRewrittenWhenTheBoardIsCleared(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: services.ErrNoProviderConfigured})
	_, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	require.Len(t, f.briefs.rows, 1)

	f.board.counts[shipment.QuickFilterLate] = 1
	f.board.counts[shipment.QuickFilterUncovered] = 0
	_, err = f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.Len(t, f.briefs.rows, 1, "a board with an issue still open keeps its brief")

	f.board.counts[shipment.QuickFilterLate] = 0
	cleared, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	require.Len(t, f.briefs.rows, 2)
	assert.Equal(t, shipmentbrief.TriggerCleared, f.briefs.rows[1].Trigger)
	assert.Equal(t, 2, cleared.Generation)
	assert.Zero(t, f.briefs.rows[1].OpenIssues)

	_, err = f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.Len(t, f.briefs.rows, 2, "a clear brief is not rewritten while the board stays clear")
}

func TestBriefingCountsOpenSuggestionsAsIssues(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: services.ErrNoProviderConfigured})
	f.board.counts[shipment.QuickFilterUncovered] = 0
	f.candidates.items = []*services.ShipmentSuggestion{{Key: "a", Title: "Approve detention"}}
	_, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	require.Len(t, f.briefs.rows, 1)
	assert.Equal(t, 1, f.briefs.rows[0].OpenIssues)

	f.candidates.items = nil
	_, err = f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	require.Len(t, f.briefs.rows, 2)
	assert.Equal(t, shipmentbrief.TriggerCleared, f.briefs.rows[1].Trigger)
}

func TestBriefingStopsRewritingAfterTheDaysLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: services.ErrNoProviderConfigured})
	_, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	f.briefs.rows[0].Generation = maxGenerationsPerDay

	f.board.counts[shipment.QuickFilterUncovered] = 0
	_, err = f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)
	assert.Len(t, f.briefs.rows, 1)
}

func TestWriteBriefForTheMorning(t *testing.T) {
	t.Parallel()

	f := newFixture(stubCompletion{err: services.ErrNoProviderConfigured})
	req := &services.WriteShipmentBriefRequest{
		TenantInfo: testTenant(),
		Trigger:    shipmentbrief.TriggerScheduled,
	}

	_, err := f.service.Briefing(t.Context(), testTenant(), "UTC")
	require.NoError(t, err)

	written, err := f.service.WriteBrief(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, 2, written.Generation, "the morning refreshes a brief written before it")
	assert.Equal(t, shipmentbrief.TriggerScheduled, written.Trigger)

	again, err := f.service.WriteBrief(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, written.ID, again.ID, "a retried morning keeps the brief it already wrote")
	assert.Len(t, f.briefs.rows, 2)
}

func TestWriteBriefStoresSuggestionWording(t *testing.T) {
	t.Parallel()

	item := &services.ShipmentSuggestion{
		Key:    "tender:smv_1",
		Title:  "Tender San Antonio → Dallas",
		Reason: "Pickup closes at 13:30.",
		Impact: []string{"$3590"},
	}
	f := newFixture(wordingCompletion{
		text: `{"items":[{"key":"tender:smv_1","title":"Send San Antonio out to tender","reason":"The $3590 load closes at 13:30."}]}`,
	})
	f.candidates.items = []*services.ShipmentSuggestion{item}

	written, err := f.service.WriteBrief(t.Context(), &services.WriteShipmentBriefRequest{
		TenantInfo: testTenant(),
		Trigger:    shipmentbrief.TriggerScheduled,
	})
	require.NoError(t, err)
	worded, ok := written.Wording["tender:smv_1"]
	require.True(t, ok)
	assert.Equal(t, "Send San Antonio out to tender", worded.Title)
	assert.True(t, worded.Fits(item.Title, item.Reason, item.Impact))
	assert.False(t, worded.Fits(item.Title, "Pickup closes at 14:30.", item.Impact))
}

type wordingCompletion struct {
	services.CompletionService
	text string
}

func (w wordingCompletion) CompleteStructured(
	_ context.Context,
	req *services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	if req.SchemaName != wordingSchemaName {
		return nil, services.ErrNoProviderConfigured
	}
	return &services.StructuredCompletionResult{Text: w.text}, nil
}

func TestAcceptedWordingKeepsOnlyRewordingsThatInventNothing(t *testing.T) {
	t.Parallel()

	honest := &services.ShipmentSuggestion{
		Key: "a", Title: "Assign Gwen Grant to San Antonio → Dallas",
		Reason: "5 mi out. Pickup closes at 13:30.", Impact: []string{"$3590"},
	}
	invented := &services.ShipmentSuggestion{
		Key: "b", Title: "Tender San Antonio → Dallas", Reason: "Pickup closes at 13:30.",
		Impact: []string{"$3590"},
	}

	wording := acceptedWording([]*services.ShipmentSuggestion{honest, invented}, []narratedItem{
		{
			Key:    "a",
			Title:  "Put Gwen on the San Antonio run",
			Reason: "She is 5 miles out and the $3590 load closes at 13:30.",
		},
		{Key: "b", Title: "Tender this one", Reason: "It pays $4200 and closes at 13:30."},
	})

	require.Len(t, wording, 1)
	assert.Equal(t, "Put Gwen on the San Antonio run", wording["a"].Title)
	assert.NotContains(t, wording, "b", "a made-up figure keeps the computed wording")
}

func TestOutputSchemaAsksForSegmentsLinkedToAllowedFilters(t *testing.T) {
	t.Parallel()

	schema := outputSchema(
		[]shipment.QuickFilter{shipment.QuickFilterLate, shipment.QuickFilterMoving},
	)

	assert.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"segments": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text": map[string]any{
							"type":        "string",
							"description": "A run of the sentence, with its own spacing and punctuation",
						},
						"filter": map[string]any{
							"type":        "string",
							"enum":        []string{"", "Late", "Moving"},
							"description": "The filter for the loads this run names, or empty",
						},
					},
					"required":             []string{"text", "filter"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"segments"},
		"additionalProperties": false,
	}, schema)
}
