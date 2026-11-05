package aidocumentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestRunStructuredRefusesWhenThePlanRestrictsDocumentIntelligence(t *testing.T) {
	t.Parallel()

	svc := &Service{plans: plantest.Restricting(t, platformplan.CapabilityDocumentIntelligence)}

	result, err := svc.runStructured(t.Context(), &structuredCall{tenant: plantest.Tenant()}, nil)

	require.Nil(t, result)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

type stoppedGate struct{}

func (stoppedGate) Decide(
	_ context.Context,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	return managedusage.Decide(&managedusage.State{
		WindowStart: 1,
		WindowEnd:   2,
		Meters: []managedusage.Meter{{
			Meter:   platformcatalog.MeterDocumentAIExtractions,
			Unit:    managedusage.UnitCount,
			Stopped: true,
		}},
	}, meter, requested), nil
}

func (stoppedGate) State(context.Context) (*managedusage.State, error) {
	return &managedusage.State{}, nil
}

func TestAnExtractionStopsWhenItsAllowanceIsStopped(t *testing.T) {
	t.Parallel()

	svc := &Service{
		plans: plantest.Allowing(t, platformplan.CapabilityDocumentIntelligence),
		quota: quotaservice.NewManaged(stoppedGate{}, func() int64 { return 0 }),
	}

	result, err := svc.runStructured(t.Context(), newExtractCall(
		&documentaiextractionports.AIExtractRequest{TenantInfo: plantest.Tenant()},
	), nil)

	require.Nil(t, result)
	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota)
}

func TestRoutingIsNotMeteredAsAnExtraction(t *testing.T) {
	t.Parallel()

	svc := &Service{
		plans: plantest.Allowing(t, platformplan.CapabilityDocumentIntelligence),
		quota: quotaservice.NewManaged(stoppedGate{}, func() int64 { return 0 }),
	}

	require.NoError(t, svc.admit(t.Context(), &structuredCall{
		tenant:  plantest.Tenant(),
		feature: aiusage.FeatureDocumentIntelligenceRoute,
	}))
}
