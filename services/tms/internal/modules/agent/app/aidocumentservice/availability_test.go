package aidocumentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// novaRunner reports whether Nova runs for the tenant, and counts the asking.
type novaRunner struct {
	llmports.TaskRunner

	servable bool
	err      error
	asked    int
}

func (n *novaRunner) Available(_ context.Context, _ pagination.TenantInfo) (bool, error) {
	n.asked++
	return n.servable, n.err
}

func TestExtractionAvailabilityNamesWhatStandsInTheWay(t *testing.T) {
	t.Parallel()

	off := false

	tests := []struct {
		name         string
		cfg          aiconfig.Config
		plans        func(t *testing.T) platformplanports.PlanService
		availability *novaRunner
		want         documentaiextractionports.ExtractionAvailability
	}{
		{
			name:         "document extraction switched off on the server",
			cfg:          aiconfig.Config{DocumentExtraction: false},
			plans:        func(*testing.T) platformplanports.PlanService { return nil },
			availability: &novaRunner{servable: true},
			want:         documentaiextractionports.ExtractionOffOnServer,
		},
		{
			name:         "all AI switched off on the server",
			cfg:          aiconfig.Config{Enabled: &off, DocumentExtraction: true},
			plans:        func(*testing.T) platformplanports.PlanService { return nil },
			availability: &novaRunner{servable: true},
			want:         documentaiextractionports.ExtractionOffOnServer,
		},
		{
			name: "plan without document intelligence",
			cfg:  aiconfig.Config{DocumentExtraction: true},
			plans: func(t *testing.T) platformplanports.PlanService {
				return plantest.Restricting(t, platformplan.CapabilityDocumentIntelligence)
			},
			availability: &novaRunner{servable: true},
			want:         documentaiextractionports.ExtractionNotInPlan,
		},
		{
			name:         "no provider chosen for Nova",
			cfg:          aiconfig.Config{DocumentExtraction: true},
			plans:        func(*testing.T) platformplanports.PlanService { return nil },
			availability: &novaRunner{servable: false},
			want:         documentaiextractionports.ExtractionNoProviderAssigned,
		},
		{
			name: "Nova runs",
			cfg:  aiconfig.Config{DocumentExtraction: true},
			plans: func(t *testing.T) platformplanports.PlanService {
				return plantest.Allowing(t, platformplan.CapabilityDocumentIntelligence)
			},
			availability: &novaRunner{servable: true},
			want:         documentaiextractionports.ExtractionAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.cfg
			svc := &Service{cfg: &cfg, plans: tt.plans(t), nova: tt.availability}

			got, err := svc.ExtractionAvailability(t.Context(), plantest.Tenant())

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(
				t,
				tt.want.Available(),
				got == documentaiextractionports.ExtractionAvailable,
			)
		})
	}
}

func TestExtractionAvailabilityAsksWhetherNovaRuns(t *testing.T) {
	t.Parallel()

	nova := &novaRunner{servable: true}
	svc := &Service{
		cfg:  &aiconfig.Config{DocumentExtraction: true},
		nova: nova,
	}

	got, err := svc.ExtractionAvailability(t.Context(), plantest.Tenant())

	require.NoError(t, err)
	assert.Equal(t, documentaiextractionports.ExtractionAvailable, got)
	assert.Equal(t, 1, nova.asked)
}

func TestExtractionAvailabilityReturnsTheRouterError(t *testing.T) {
	t.Parallel()

	failure := errors.New("provider lookup failed")
	svc := &Service{
		cfg:  &aiconfig.Config{DocumentExtraction: true},
		nova: &novaRunner{err: failure},
	}

	_, err := svc.ExtractionAvailability(t.Context(), plantest.Tenant())

	require.ErrorIs(t, err, failure)
}

var _ llmports.TaskRunner = (*novaRunner)(nil)
