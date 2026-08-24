package capabilitytest

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
)

type OrgRepoParams struct {
	TenantInfo             pagination.TenantInfo
	Lock                   repositories.CapabilityLock
	BrokerageEnabled       bool
	AssetOperationsEnabled bool
	Optional               bool
}

func OrgRepo(t *testing.T, params OrgRepoParams) *mocks.MockOrganizationRepository {
	t.Helper()

	orgRepo := mocks.NewMockOrganizationRepository(t)
	call := orgRepo.EXPECT().
		GetCapabilities(mock.Anything, repositories.GetOrganizationCapabilitiesRequest{
			TenantInfo: params.TenantInfo,
			Lock:       params.Lock,
		}).
		Return(&repositories.OrganizationCapabilities{
			BrokerageEnabled:       params.BrokerageEnabled,
			AssetOperationsEnabled: params.AssetOperationsEnabled,
		}, nil)

	if params.Optional {
		call.Maybe()
	} else {
		call.Once()
	}

	return orgRepo
}
