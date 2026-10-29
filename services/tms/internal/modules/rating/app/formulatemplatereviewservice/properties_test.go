package formulatemplatereviewservice

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const heavierBillsLessSource = `tariff "Per Mile"

/// What the contract pays per loaded mile.
input rate: Money/mi = $2/mi where $0.50/mi..=$15/mi

charge "Per Mile" (quantity: shipment.distance, rate: rate)
let weight = shipment.weight ?? 0 lb
if weight > 999 lb {
  charge "Heavy discount" = -$50
}

test "a short haul bills its miles" {
  given shipment.distance = 100 mi
  expect total == $200
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  given shipment.distance = 100 mi
  expect total nondecreasing in shipment.weight
}
`

const heavierBillsMoreSource = `tariff "Per Mile"

/// What the contract pays per loaded mile.
input rate: Money/mi = $2/mi where $0.50/mi..=$15/mi

charge "Per Mile" (quantity: shipment.distance, rate: rate)
let weight = shipment.weight ?? 0 lb
if weight > 30_000 lb {
  charge "Heavy" = $50
}
if weight > 45_000 lb {
  charge "Overweight" = $150
}

test "a short haul bills its miles" {
  given shipment.distance = 100 mi
  expect total == $200
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  given shipment.distance = 100 mi
  expect total nondecreasing in shipment.weight
}

property "the rate bounds keep the bill positive" {
  vary input.rate
  given shipment.distance = 100 mi
  expect total > $0
}
`

func draftWith(deps *testDeps, source string) *formulatemplate.FormulaTemplate {
	template := formulatemplatetest.NewTemplate()
	template.Status = formulatemplate.StatusDraft
	template.Source = source
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(template, nil)
	return template
}

func TestSubmit_AFailingPropertyIsRefusedNamingItsCounterexample(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := draftWith(deps, heavierBillsLessSource)

	_, err := deps.svc.Submit(
		t.Context(),
		&SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: formulatemplatetest.NewTenantInfo(),
			EntityID:   template.ID,
		}},
	)

	require.Error(t, err)
	fields := formulatemplatetest.ValidationFields(t, err)
	require.Contains(t, fields, ReadinessCheckTests)
	assert.Equal(t,
		"Only 1 of 2 tests pass (heavier never bills less (total is $200.00 at shipment.weight = "+
			"999 lb but $150.00 at 1000 lb)); fix the formula or the tests before submitting",
		errortypes.LocalizedText(t.Context(), fields[ReadinessCheckTests]),
	)
	deps.repo.AssertNotCalled(t, "Update")
}

func TestReadiness_AFailingPropertyBlocksSubmitAndCoverageOnlyWarns(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	template := draftWith(deps, heavierBillsLessSource)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{}, nil)
	deps.repo.On("CountUsages", mock.Anything, mock.Anything).
		Return(&formulatemplateports.GetTemplateUsageResponse{}, nil)

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.False(t, resp.CanSubmit)
	assert.Equal(t, 2, resp.TestTotal)
	assert.Equal(t, 1, resp.TestPassed)
	assert.Equal(t, []string{"heavier never bills less"}, resp.TestFailing)
	tests := readinessCheck(t, resp, ReadinessCheckTests)
	assert.Equal(t, ReadinessFail, tests.Status)

	require.NotNil(t, resp.Coverage)
	assert.Equal(t, resp.Coverage.Total, resp.Coverage.Reached, "every region ran")
	coverage := readinessCheck(t, resp, ReadinessCheckCoverage)
	assert.Equal(t, ReadinessPass, coverage.Status)
}

func TestReadiness_PassingPropertiesSubmitAndUnreachedRegionsWarn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	template := draftWith(deps, heavierBillsMoreSource)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).Return(nil, nil)
	deps.versionRepo.On("ListScheduled", mock.Anything, mock.Anything).
		Return([]*formulatemplate.FormulaTemplateVersion{}, nil)
	deps.repo.On("CountUsages", mock.Anything, mock.Anything).
		Return(&formulatemplateports.GetTemplateUsageResponse{}, nil)

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		TemplateID: template.ID,
	})

	require.NoError(t, err)
	assert.True(t, resp.CanSubmit, "%+v", resp.Checks)
	assert.Equal(t, 3, resp.TestTotal)
	assert.Equal(t, 3, resp.TestPassed)
	assert.Equal(t, ReadinessPass, readinessCheck(t, resp, ReadinessCheckTests).Status)

	require.NotNil(t, resp.Coverage)
	require.Len(t, resp.Coverage.Unreached, 2, "the overweight branch and its charge")
	unreached := resp.Coverage.Unreached[0]
	assert.Equal(t, 11, unreached.At.Line)
	assert.Equal(t, "if weight > 45_000 lb {", unreached.Excerpt)
	coverage := readinessCheck(t, resp, ReadinessCheckCoverage)
	assert.Equal(t, ReadinessWarn, coverage.Status)
	assert.Equal(t,
		"Tests reach "+itoa(resp.Coverage.Reached)+" of "+itoa(resp.Coverage.Total)+
			" regions; no test runs the rest",
		coverage.Detail,
	)
}

func TestSubmit_PassingPropertiesAreSubmitted(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	primeReviews(deps, nil)
	template := draftWith(deps, heavierBillsMoreSource)
	deps.repo.On("Update", mock.Anything, mock.Anything).Return(template, nil)
	deps.versionRepo.On("GetLatestByStatus", mock.Anything, mock.Anything).
		Return(&formulatemplate.FormulaTemplateVersion{VersionNumber: 1}, nil)
	deps.auditSvc.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := deps.svc.Submit(
		t.Context(),
		&SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
			TenantInfo: formulatemplatetest.NewTenantInfo(),
			EntityID:   template.ID,
		}},
	)

	require.NoError(t, err)
}

func readinessCheck(t *testing.T, resp *ReadinessResponse, key string) ReadinessCheck {
	t.Helper()
	for _, check := range resp.Checks {
		if check.Key == key {
			return check
		}
	}
	require.Failf(t, "no readiness check", "key %s in %+v", key, resp.Checks)
	return ReadinessCheck{}
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
