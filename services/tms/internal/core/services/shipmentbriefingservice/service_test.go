package shipmentbriefingservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
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
		DeliveringToday: 14,
		Moving:          9,
		Late:            2,
		Uncovered:       3,
		LateReason:      "Shipper delay",
		OperationType:   tenant.OperationTypeAsset,
	})
	assert.Equal(
		t,
		"14 loads deliver today, 9 loads are moving on schedule and 2 loads are late, mostly shipper delay. 3 loads still need a driver.",
		joined(segments),
	)
	require.NotNil(t, segments[0].Filter)
	assert.Equal(t, shipment.QuickFilterDeliveringToday, *segments[0].Filter)

	single := Deterministic(Facts{
		DeliveringToday: 1,
		Late:            1,
		Uncovered:       1,
		OperationType:   tenant.OperationTypeBrokerage,
	})
	assert.Equal(
		t,
		"1 load delivers today and 1 load is late. 1 load still needs a carrier.",
		joined(single),
	)

	onlyUncovered := Deterministic(Facts{Uncovered: 2, OperationType: tenant.OperationTypeBoth})
	assert.Equal(t, "2 loads still need coverage.", joined(onlyUncovered))

	empty := Deterministic(Facts{})
	assert.Equal(t, "Nothing on the board needs you right now.", joined(empty))
	assert.Nil(t, empty[0].Filter)
}

func TestAcceptNarration(t *testing.T) {
	t.Parallel()

	facts := &Facts{DeliveringToday: 140, Late: 32, Uncovered: 3}
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
		[]services.ShipmentBriefingSegment{{Text: strings.Repeat("a", 500)}},
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
}

func (stubBoard) QuickFilterTotals(
	_ context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) ([]repositories.ShipmentQuickFilterTotal, error) {
	counts := map[shipment.QuickFilter]int{
		shipment.QuickFilterDeliveringToday: 30,
		shipment.QuickFilterMoving:          12,
		shipment.QuickFilterLate:            0,
		shipment.QuickFilterUncovered:       4,
	}
	out := make([]repositories.ShipmentQuickFilterTotal, len(req.Filters))
	for i, spec := range req.Filters {
		out[i].Count = counts[spec.Filter]
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

type stubCompletion struct {
	services.CompletionService
	text string
	err  error
}

func (s stubCompletion) CompleteStructured(
	context.Context,
	*services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &services.StructuredCompletionResult{Text: s.text}, nil
}

func newService(completion services.CompletionService) *Service {
	return NewWithDependencies(&Dependencies{
		Board:         stubBoard{},
		Briefing:      stubReasons{},
		QuickFilters:  stubBasis{},
		Organizations: stubOrgs{},
		Completion:    completion,
		Logger:        zap.NewNop(),
		Now:           func() time.Time { return time.Unix(1_791_000_000, 0) },
	})
}

func TestBriefingFallsBackWithoutAModel(t *testing.T) {
	t.Parallel()

	svc := newService(stubCompletion{err: services.ErrNoProviderConfigured})
	briefing, err := svc.Briefing(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
	assert.Equal(t, int64(1_791_000_000), briefing.GeneratedAt)
	assert.Equal(
		t,
		"30 loads deliver today and 12 loads are moving on schedule. 4 loads still need a driver.",
		joined(briefing.Segments),
	)

	broken := newService(stubCompletion{err: errors.New("timeout")})
	briefing, err = broken.Briefing(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
}

func TestBriefingUsesGuardedNarration(t *testing.T) {
	t.Parallel()

	svc := newService(stubCompletion{
		text: `{"segments":[{"text":"4 loads need a driver","filter":"Uncovered"},{"text":" while 30 deliver today.","filter":""}]}`,
	})
	briefing, err := svc.Briefing(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)
	assert.True(t, briefing.Narrated)
	require.Len(t, briefing.Segments, 2)
	assert.Equal(t, shipment.QuickFilterUncovered, *briefing.Segments[0].Filter)
	assert.Nil(t, briefing.Segments[1].Filter)

	lying := newService(stubCompletion{
		text: `{"segments":[{"text":"55 loads are late","filter":"Late"}]}`,
	})
	briefing, err = lying.Briefing(t.Context(), pagination.TenantInfo{}, "UTC")
	require.NoError(t, err)
	assert.False(t, briefing.Narrated)
}

func TestOutputSchemaAsksForSegmentsLinkedToAllowedFilters(t *testing.T) {
	t.Parallel()

	schema := outputSchema([]shipment.QuickFilter{shipment.QuickFilterLate, shipment.QuickFilterMoving})

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
