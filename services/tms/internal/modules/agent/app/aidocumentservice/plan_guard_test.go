package aidocumentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/require"
)

func TestRunStructuredRefusesWhenThePlanRestrictsDocumentIntelligence(t *testing.T) {
	t.Parallel()

	svc := &Service{features: plantest.Without(platformcatalog.FeatureDocumentIntelligence)}

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
			Meter:   platformcatalog.MeterDocumentIntelligencePages,
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
		features: plantest.AllFeatures(),
		quota:    quotaservice.NewManaged(stoppedGate{}, func() int64 { return 0 }),
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
		features: plantest.AllFeatures(),
		quota:    quotaservice.NewManaged(stoppedGate{}, func() int64 { return 0 }),
	}

	require.NoError(t, svc.admit(t.Context(), &structuredCall{
		tenant:  plantest.Tenant(),
		feature: aiusage.FeatureDocumentIntelligenceRoute,
	}))
}

type pagesGate struct {
	overage bool
}

func (g pagesGate) Decide(
	_ context.Context,
	meter platformcatalog.MeterKey,
	requested int64,
) (managedusage.Decision, error) {
	return managedusage.Decide(&managedusage.State{
		WindowStart: 1,
		WindowEnd:   2,
		Overage:     g.overage,
		Meters: []managedusage.Meter{{
			Meter:     platformcatalog.MeterDocumentIntelligencePages,
			Unit:      managedusage.UnitCount,
			Quantity:  995,
			Allowance: 1000,
		}},
	}, meter, requested), nil
}

func (pagesGate) State(context.Context) (*managedusage.State, error) {
	return &managedusage.State{}, nil
}

func pagedExtraction(pages int) *structuredCall {
	return newExtractCall(&documentaiextractionports.AIExtractRequest{
		TenantInfo: plantest.Tenant(),
		PageCount:  pages,
	})
}

func TestAnExtractionThatWouldCrossThePageAllowanceIsRefusedWithOverageOff(t *testing.T) {
	t.Parallel()

	svc := &Service{
		features: plantest.AllFeatures(),
		quota:    quotaservice.NewManaged(pagesGate{}, func() int64 { return 0 }),
	}

	require.NoError(t, svc.admit(t.Context(), pagedExtraction(5)))

	err := svc.admit(t.Context(), pagedExtraction(6))
	var quota *errortypes.QuotaExceededError
	require.ErrorAs(t, err, &quota)
	require.Equal(t, string(platformcatalog.MeterDocumentIntelligencePages), quota.Meter)
}

func TestAnExtractionPastThePageAllowanceRunsWithOverageOn(t *testing.T) {
	t.Parallel()

	svc := &Service{
		features: plantest.AllFeatures(),
		quota:    quotaservice.NewManaged(pagesGate{overage: true}, func() int64 { return 0 }),
	}

	require.NoError(t, svc.admit(t.Context(), pagedExtraction(500)))
}

func TestADocumentWithUnknownPagesIsAdmittedAndCountedAsOnePage(t *testing.T) {
	t.Parallel()

	call := pagedExtraction(0)
	svc := &Service{
		features: plantest.AllFeatures(),
		quota:    quotaservice.NewManaged(pagesGate{}, func() int64 { return 0 }),
	}

	require.NoError(t, svc.admit(t.Context(), call))
	require.Equal(t, 1, call.novaTask(1).Attribution.Pages)
}

func TestOnlyAnExtractionCarriesPagesOnItsUsageRecord(t *testing.T) {
	t.Parallel()

	extract := documentAttribution(plantest.Tenant(), "", aiusage.FeatureDocumentIntelligenceExtract, 7)
	route := documentAttribution(plantest.Tenant(), "", aiusage.FeatureDocumentIntelligenceRoute, 7)

	require.Equal(t, 7, extract.Pages)
	require.Zero(t, route.Pages)
}
