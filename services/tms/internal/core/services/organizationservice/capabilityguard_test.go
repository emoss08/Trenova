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

func capabilityErrorMessage(t *testing.T, err error, field string) string {
	t.Helper()

	multiErr := new(errortypes.MultiError)
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, field, multiErr.Errors[0].Field)
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

			deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
				TenantInfo: tenantInfo,
				Lock:       repositories.CapabilityLockUpdate,
			}).Return(&repositories.OrganizationCapabilities{
				BrokerageEnabled:       stored.BrokerageEnabled,
				AssetOperationsEnabled: stored.AssetOperationsEnabled,
			}, nil)
			deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
				Return(tt.counts, nil)

			result, err := deps.svc.Update(t.Context(), entity)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, capabilityErrorMessage(t, err, "brokerageEnabled"), tt.expected)
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

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
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
		capabilityErrorMessage(t, err, "brokerageEnabled"),
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

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
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
	deps.repo.AssertNotCalled(t, "GetCapabilities")
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

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.ID,
			BuID:  entity.BusinessUnitID,
		},
		Lock: repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
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

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
	deps.repo.On("CountBrokerageDependencies", mock.Anything, tenantInfo).
		Return(nil, countErr)

	result, err := deps.svc.Update(t.Context(), entity)

	require.ErrorIs(t, err, countErr)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableAssetOperations_BlockedByEachDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		counts   *repositories.AssetDependencyCounts
		expected string
	}{
		{
			name:     "driver assignments",
			counts:   &repositories.AssetDependencyCounts{ActiveDriverAssignments: 1},
			expected: "Cannot disable asset operations: 1 active driver assignment.",
		},
		{
			name:     "driver settlements",
			counts:   &repositories.AssetDependencyCounts{UnpaidDriverSettlements: 2},
			expected: "Cannot disable asset operations: 2 unpaid driver settlements.",
		},
		{
			name:     "portal users",
			counts:   &repositories.AssetDependencyCounts{LinkedPortalUsers: 3},
			expected: "Cannot disable asset operations: 3 linked driver portal users.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTest(t)

			entity := newTestOrganization()
			entity.AssetOperationsEnabled = false

			stored := newTestOrganization()
			stored.ID = entity.ID
			stored.BusinessUnitID = entity.BusinessUnitID

			tenantInfo := pagination.TenantInfo{
				OrgID: entity.ID,
				BuID:  entity.BusinessUnitID,
			}

			deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
				TenantInfo: tenantInfo,
				Lock:       repositories.CapabilityLockUpdate,
			}).Return(&repositories.OrganizationCapabilities{
				BrokerageEnabled:       stored.BrokerageEnabled,
				AssetOperationsEnabled: stored.AssetOperationsEnabled,
			}, nil)
			deps.repo.On("CountAssetDependencies", mock.Anything, tenantInfo).
				Return(tt.counts, nil)

			result, err := deps.svc.Update(t.Context(), entity)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(
				t,
				capabilityErrorMessage(t, err, "assetOperationsEnabled"),
				tt.expected,
			)
			deps.repo.AssertNotCalled(t, "Update")
			deps.repo.AssertNotCalled(t, "CountBrokerageDependencies")
			deps.repo.AssertExpectations(t)
		})
	}
}

func TestUpdate_DisableAssetOperations_CombinesEveryBlocker(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
	deps.repo.On("CountAssetDependencies", mock.Anything, tenantInfo).
		Return(&repositories.AssetDependencyCounts{
			ActiveDriverAssignments: 4,
			UnpaidDriverSettlements: 1,
			LinkedPortalUsers:       2,
		}, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t,
		"Cannot disable asset operations: 4 active driver assignments, "+
			"1 unpaid driver settlement, 2 linked driver portal users. "+
			"Resolve or close this work before turning asset operations off",
		capabilityErrorMessage(t, err, "assetOperationsEnabled"),
	)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableAssetOperations_AllowedWhenNothingOutstanding(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
	deps.repo.On("CountAssetDependencies", mock.Anything, tenantInfo).
		Return(&repositories.AssetDependencyCounts{}, nil)
	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.False(t, result.AssetOperationsEnabled)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_EnableAssetOperations_IsAlwaysAllowed(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = true

	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.True(t, result.AssetOperationsEnabled)
	deps.repo.AssertNotCalled(t, "GetCapabilities")
	deps.repo.AssertNotCalled(t, "CountAssetDependencies")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_AssetOperationsAlreadyDisabled_SkipsDependencyCount(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID
	stored.AssetOperationsEnabled = false

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.ID,
			BuID:  entity.BusinessUnitID,
		},
		Lock: repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.False(t, result.AssetOperationsEnabled)
	deps.repo.AssertNotCalled(t, "CountAssetDependencies")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableAssetOperations_PropagatesCountError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.ID,
		BuID:  entity.BusinessUnitID,
	}

	countErr := errors.New("count failed")

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: tenantInfo,
		Lock:       repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)
	deps.repo.On("CountAssetDependencies", mock.Anything, tenantInfo).
		Return(nil, countErr)

	result, err := deps.svc.Update(t.Context(), entity)

	require.ErrorIs(t, err, countErr)
	assert.Nil(t, result)
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DisableBothCapabilities_IsRejected(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.BrokerageEnabled = false
	entity.AssetOperationsEnabled = false

	result, err := deps.svc.Update(t.Context(), entity)

	require.Error(t, err)
	assert.Nil(t, result)

	multiErr := new(errortypes.MultiError)
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 2)

	const message = "At least one operating capability must be enabled. " +
		"An organization with neither brokerage nor asset operations cannot move freight"

	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
		assert.Equal(t, errortypes.ErrInvalid, fieldErr.Code)
		assert.Equal(t, message, fieldErr.Message)
	}
	assert.ElementsMatch(t, []string{"brokerageEnabled", "assetOperationsEnabled"}, fields)

	deps.repo.AssertNotCalled(t, "GetCapabilities")
	deps.repo.AssertNotCalled(t, "Update")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_AssetOperationsEnabled_IsUnaffectedByBrokerageGuard(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()

	deps.repo.On("Update", mock.Anything, entity).Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	assert.True(t, result.AssetOperationsEnabled)
	assert.True(t, result.BrokerageEnabled)
	deps.repo.AssertNotCalled(t, "GetCapabilities")
	deps.repo.AssertNotCalled(t, "CountAssetDependencies")
	deps.repo.AssertNotCalled(t, "CountBrokerageDependencies")
	deps.repo.AssertExpectations(t)
}

func TestUpdate_ReadsCapabilitiesUnderRowLockInsideTheWriteTransaction(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	var order []string

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.ID,
			BuID:  entity.BusinessUnitID,
		},
		Lock: repositories.CapabilityLockUpdate,
	}).Run(func(mock.Arguments) {
		order = append(order, "getCapabilities")
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)

	deps.repo.On("CountAssetDependencies", mock.Anything, mock.Anything).
		Run(func(mock.Arguments) {
			order = append(order, "countAssetDependencies")
		}).
		Return(&repositories.AssetDependencyCounts{}, nil)

	deps.repo.On("Update", mock.Anything, entity).
		Run(func(mock.Arguments) {
			order = append(order, "update")
		}).
		Return(entity, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(
		t,
		[]string{"getCapabilities", "countAssetDependencies", "update"},
		order,
		"the capability row must be locked before dependencies are counted and before the write",
	)
	deps.repo.AssertExpectations(t)
}

func TestUpdate_DoesNotWriteWhenTheGuardRefusesInsideTheTransaction(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestOrganization()
	entity.AssetOperationsEnabled = false

	stored := newTestOrganization()
	stored.ID = entity.ID
	stored.BusinessUnitID = entity.BusinessUnitID

	deps.repo.On("GetCapabilities", mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.ID,
			BuID:  entity.BusinessUnitID,
		},
		Lock: repositories.CapabilityLockUpdate,
	}).Return(&repositories.OrganizationCapabilities{
		BrokerageEnabled:       stored.BrokerageEnabled,
		AssetOperationsEnabled: stored.AssetOperationsEnabled,
	}, nil)

	deps.repo.On("CountAssetDependencies", mock.Anything, mock.Anything).
		Return(&repositories.AssetDependencyCounts{ActiveDriverAssignments: 1}, nil)

	result, err := deps.svc.Update(t.Context(), entity)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(
		t,
		capabilityErrorMessage(t, err, "assetOperationsEnabled"),
		"1 active driver assignment",
	)
	deps.repo.AssertNotCalled(t, "Update")
}
