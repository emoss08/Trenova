package capabilitytest

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/mock"
)

type OrgRepoParams struct {
	TenantInfo             pagination.TenantInfo
	Lock                   tenantports.CapabilityLock
	BrokerageEnabled       bool
	AssetOperationsEnabled bool
	Optional               bool
}

func OrgRepo(t *testing.T, params OrgRepoParams) *tenantportsmock.MockOrganizationRepository {
	t.Helper()

	orgRepo := tenantportsmock.NewMockOrganizationRepository(t)
	call := orgRepo.EXPECT().
		GetCapabilities(mock.Anything, tenantports.GetOrganizationCapabilitiesRequest{
			TenantInfo: params.TenantInfo,
			Lock:       params.Lock,
		}).
		Return(&tenantports.OrganizationCapabilities{
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
