package latechargeservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanAssessIsAPreviewThatSaysWhetherTheMemosPost(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{
		LateChargeAssessmentMode: tenant.LateChargeAssessmentModePreview,
		InvoicePostingMode:       tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
	})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf, candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000))

	result, err := f.svc.PlanAssess(t.Context(), &servicesports.LateChargeAssessmentRequest{
		TenantInfo: f.tenantInfo, AsOfDate: asOf,
	}, f.actor)
	require.NoError(t, err)

	assert.True(t, result.Preview)
	assert.True(t, result.AutoPost)
	assert.Equal(t, int64(750), result.TotalChargeMinor)
	f.repo.AssertNotCalled(t, "InsertAssessments", mock.Anything, mock.Anything)
	f.invoiceService.AssertNotCalled(t, "CreateMemo", mock.Anything, mock.Anything, mock.Anything)
}

func TestPlanAssessRefusesWhatARealRunRefuses(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{
		LateChargeAssessmentMode: tenant.LateChargeAssessmentModeDisabled,
	})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf)

	_, err := f.svc.PlanAssess(t.Context(), &servicesports.LateChargeAssessmentRequest{
		TenantInfo: f.tenantInfo, AsOfDate: asOf,
	}, f.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
