package organizationservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func brokerageErrorMessage(t *testing.T, err error) string {
	t.Helper()

	multiErr := new(errortypes.MultiError)
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "brokerageEnabled", multiErr.Errors[0].Field)
	assert.Equal(t, errortypes.ErrResourceInUse, multiErr.Errors[0].Code)

	return multiErr.Errors[0].Message
}

func TestUpdate_DisableBrokerage_BlockedByEachDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		counts   *repositories.BrokerageDependencyCounts
		expected string
	}{
		{
			name:     "active tenders",
			counts:   &repositories.BrokerageDependencyCounts{ActiveTenders: 1},
			expected: "Cannot disable brokerage: 1 active tender.",
		},
		{
			name:     "rate confirmations",
			counts:   &repositories.BrokerageDependencyCounts{ActiveRateConfirmations: 2},
			expected: "Cannot disable brokerage: 2 unvoided rate confirmations.",
		},
		{
			name:     "carrier settlements",
			counts:   &repositories.BrokerageDependencyCounts{UnpaidCarrierSettlements: 3},
			expected: "Cannot disable brokerage: 3 unpaid carrier settlements.",
		},
		{
			name:     "invoice matches",
			counts:   &repositories.BrokerageDependencyCounts{OpenCarrierInvoiceMatches: 1},
			expected: "Cannot disable brokerage: 1 open carrier invoice match.",
		},
		{
			name:     "carrier assignments",
			counts:   &repositories.BrokerageDependencyCounts{ActiveCarrierAssignments: 4},
			expected: "Cannot disable brokerage: 4 active carrier assignments.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTest(t)

			entity := newTestOrganization()
			entity.BrokerageEnabled = false

			stored := newTestOrganization()
			stored.ID = entity.ID
			stored.BusinessUnitID = entity.BusinessUnitID

			tenantInfo := pagination.TenantInfo{
				OrgID: entity.ID,
				BuID:  entity.BusinessUnitID,
			}

			deps.repo.On("GetByID", mock.Anything, repositories.GetOrganizationByIDRequest{
				TenantInfo: tenantInfo,
			}).Return(stored, nil)
			deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
				Return(tt.counts, nil)

			result, err := deps.svc.Update(t.Context(), entity)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, brokerageErrorMessage(t, err), tt.expected)
			deps.repo.AssertNotCalled(t, "Update")
			deps.repo.AssertExpectations(t)
		})
	}
}

func TestUpdate_DisableBrokerage_CombinesEveryBlocker(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	deps.repo.On("GetByID", mock.Anything, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	}).Return(stored, nil)
	deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
		Return(&repositories.BrokerageDependencyCounts{
			ActiveTenders:             1,
			ActiveRateConfirmations:   2,
			UnpaidCarrierSettlements:  3,
			OpenCarrierInvoiceMatches: 1,
			ActiveCarrierAssignments:  5,
		}, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t,
		"Cannot disable brokerage: 1 active tender, 2 unvoided rate confirmations, "+
			"3 unpaid carrier settlements, 1 open carrier invoice match, "+
			"5 active carrier assignments. Resolve or close this work before "+
			"turning brokerage off",
		brokerageErrorMessage(t, err),
	)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableBrokerage_AllowedWhenNothingOutstanding(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	deps.repo.On("GetByID", mock.Anything, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	}).Return(stored, nil)
	deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
		Return(&repositories.BrokerageDependencyCounts{}, nil)
	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.False(t, result.BrokerageEnabled)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_EnableBrokerage_IsAlwaysAllowed(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = true

	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.True(t, result.BrokerageEnabled)
	deps.repo.AssertNotCalled(t, "GetByID")
	deps.repo.AssertNotCalled(t, "CountBrokerageDependencies")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_BrokerageAlreadyDisabled_SkipsDependencyCount(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID
	stored.BrokerageEnabled = false

	deps.repo.On("GetByID", mock.Anything, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.ID,
			BuID:  entity.BusinessUnitID,
		},
	}).Return(stored, nil)
	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.False(t, result.BrokerageEnabled)
	deps.repo.AssertNotCalled(t, "CountBrokerageDependencies")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableBrokerage_PropagatesCountError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	countErr := errors.New("count failed")

	deps.repo.On("GetByID", mock.Anything, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	}).Return(stored, nil)
	deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
		Return(nil, countErr)

	result, err := deps.svc.Update(t.Context(), entity)

	require.ErrorIs(t, err, countErr)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_AssetOperationsFlag_IsNotGuarded(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.False(t, result.AssetOperationsEnabled)
	deps.repo.AssertNotCalled(t, "CountBrokerageDependencies")
	deps.repo.AssertExpectations(t)
}
