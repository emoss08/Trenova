package formulalibraryservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const stepFeeSource = `library "org.accessorials"

/// The fee for a load of the given weight.
fn fee(weight: Weight) -> Money = if weight > 999 lb { $40 } else { $50 }

test "a light load pays the base fee" {
  expect fee(100 lb) == $50
}

property "a heavier load never pays less" {
  vary weight in 1 lb..10_000 lb
  expect fee(weight) nondecreasing in weight
}
`

func TestSubmit_RefusesALibraryWhosePropertyFailsNamingItsCounterexample(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Source = stepFeeSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)

	_, err := deps.svc.Submit(t.Context(), &SubmitRequest{ApprovalActionRequest: ApprovalActionRequest{
		TenantInfo: newTenantInfo(),
		EntityID:   library.ID,
	}})

	require.Error(t, err)
	assert.Contains(t, fieldsOf(t, err), ReadinessCheckTests)
	assert.Contains(t, err.Error(),
		"a heavier load never pays less (fee(weight) is $50.00 at weight = 999 lb but $40.00 at 1000 lb)")
	deps.repo.AssertNotCalled(t, "Update")
}

func TestReadiness_ReportsALibrarysCoverageAsAWarning(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	library := newLibrary()
	library.Source = stepFeeSource
	deps.repo.On("GetByID", mock.Anything, mock.Anything).Return(library, nil)

	resp, err := deps.svc.Readiness(t.Context(), &ReadinessRequest{
		TenantInfo: newTenantInfo(),
		LibraryID:  library.ID,
	})

	require.NoError(t, err)
	assert.False(t, resp.CanSubmit)
	require.NotNil(t, resp.Coverage)
	assert.Positive(t, resp.Coverage.Total)
	var coverage *ReadinessCheck
	for i := range resp.Checks {
		if resp.Checks[i].Key == ReadinessCheckCoverage {
			coverage = &resp.Checks[i]
		}
	}
	require.NotNil(t, coverage)
	assert.NotEqual(t, ReadinessFail, coverage.Status)
}
