package shipmentimportassistantservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"errors"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func importTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func expectCheck(
	engine *permissionportsmock.MockPermissionEngine,
	tenant pagination.TenantInfo,
	resource permission.Resource,
	operation permission.Operation,
	allowed bool,
) {
	engine.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.PrincipalType == permissionports.PrincipalTypeUser &&
				req.PrincipalID == tenant.UserID &&
				req.UserID == tenant.UserID &&
				req.OrganizationID == tenant.OrgID &&
				req.BusinessUnitID == tenant.BuID &&
				req.Resource == resource.String() &&
				req.Operation == operation
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: allowed}, nil).
		Once()
}

func toolError(t *testing.T, outcome ToolOutcome) string {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, sonic.UnmarshalString(outcome.Output, &decoded))
	message, _ := decoded["error"].(string)

	return message
}

func TestRunTool_RefusesAddLocationWithoutLocationCreate(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceLocation, permission.OpCreate, false)
	service := &Service{logger: zap.NewNop(), permissions: engine}

	outcome := service.RunTool(t.Context(), tenant, &agentports.ToolCall{
		ID:   "call_1",
		Name: "add_location",
		Arguments: map[string]any{
			"name":          "Acme Dock",
			"address_line1": "1 Main St",
			"city":          "Reno",
			"state_abbrev":  "NV",
			"postal_code":   "89501",
		},
	})

	assert.Equal(t, toolStatusError, outcome.Status)
	assert.Empty(t, outcome.Actions)
	message := toolError(t, outcome)
	assert.Contains(t, message, "add_location")
	assert.Contains(t, message, "create")
	assert.Contains(t, message, "location")
}

func TestRunTool_RefusesCustomerReadsWithoutCustomerRead(t *testing.T) {
	t.Parallel()

	for _, call := range []agentports.ToolCall{
		{ID: "call_1", Name: "search_customers", Arguments: map[string]any{"query": "Acme"}},
		{
			ID:        "call_2",
			Name:      "get_customer_requirements",
			Arguments: map[string]any{"customer_id": pulid.MustNew("cus_").String()},
		},
	} {
		t.Run(call.Name, func(t *testing.T) {
			t.Parallel()

			tenant := importTenant()
			engine := permissionportsmock.NewMockPermissionEngine(t)
			expectCheck(engine, tenant, permission.ResourceCustomer, permission.OpRead, false)
			service := &Service{logger: zap.NewNop(), permissions: engine}

			outcome := service.RunTool(t.Context(), tenant, &call)

			assert.Equal(t, toolStatusError, outcome.Status)
			message := toolError(t, outcome)
			assert.Contains(t, message, call.Name)
			assert.Contains(t, message, "customer")
		})
	}
}

func TestRunTool_RefusesLocationSearchWithoutLocationRead(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceLocation, permission.OpRead, false)
	service := &Service{logger: zap.NewNop(), permissions: engine}

	outcome := service.RunTool(t.Context(), tenant, &agentports.ToolCall{
		ID:        "call_1",
		Name:      "search_locations",
		Arguments: map[string]any{"query": "Reno"},
	})

	assert.Equal(t, toolStatusError, outcome.Status)
	assert.Contains(t, toolError(t, outcome), "search_locations")
}

func TestRunTool_SearchesCustomersWhenThePersonMayReadThem(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceCustomer, permission.OpRead, true)
	customers := customerportsmock.NewMockCustomerRepository(t)
	customerID := pulid.MustNew("cus_")
	customers.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *customerports.CustomerSelectOptionsRequest) bool {
			return req.SelectQueryRequest.TenantInfo == tenant && req.SelectQueryRequest.Query == "Acme"
		})).
		Return(&pagination.ListResult[*customer.Customer]{
			Items: []*customer.Customer{{ID: customerID, Name: "Acme Freight"}},
			Total: 1,
		}, nil).
		Once()
	service := &Service{logger: zap.NewNop(), permissions: engine, customerRepo: customers}

	outcome := service.RunTool(t.Context(), tenant, &agentports.ToolCall{
		ID:        "call_1",
		Name:      "search_customers",
		Arguments: map[string]any{"query": "Acme"},
	})

	assert.Equal(t, toolStatusCompleted, outcome.Status)
	assert.Contains(t, outcome.Output, customerID.String())
}

func TestRunTool_ReportsAFailedPermissionCheckWithoutFailingTheTurn(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.EXPECT().
		Check(mock.Anything, mock.Anything).
		Return(nil, errors.New("engine unavailable")).
		Once()
	service := &Service{logger: zap.NewNop(), permissions: engine}

	outcome := service.RunTool(t.Context(), tenant, &agentports.ToolCall{
		ID:        "call_1",
		Name:      "search_locations",
		Arguments: map[string]any{"query": "Reno"},
	})

	assert.Equal(t, toolStatusError, outcome.Status)
	assert.Contains(t, toolError(t, outcome), "search_locations")
}

func TestRunTool_RefusesAGatedToolWhenNobodyIsActing(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	tenant.UserID = pulid.Nil
	service := &Service{logger: zap.NewNop(), permissions: permissionportsmock.NewMockPermissionEngine(t)}

	outcome := service.RunTool(t.Context(), tenant, &agentports.ToolCall{
		ID:        "call_1",
		Name:      "search_customers",
		Arguments: map[string]any{"query": "Acme"},
	})

	assert.Equal(t, toolStatusError, outcome.Status)
}

func TestRunTool_RefusesAGatedToolWithoutAPermissionEngine(t *testing.T) {
	t.Parallel()

	service := &Service{logger: zap.NewNop()}

	outcome := service.RunTool(t.Context(), importTenant(), &agentports.ToolCall{
		ID:        "call_1",
		Name:      "search_customers",
		Arguments: map[string]any{"query": "Acme"},
	})

	assert.Equal(t, toolStatusError, outcome.Status)
}

func TestRunTool_DraftEditsNeedNoPermissionCheck(t *testing.T) {
	t.Parallel()

	service := &Service{logger: zap.NewNop(), permissions: permissionportsmock.NewMockPermissionEngine(t)}

	outcome := service.RunTool(t.Context(), importTenant(), &agentports.ToolCall{
		ID:        "call_1",
		Name:      "accept_field",
		Arguments: map[string]any{"field_key": "bol"},
	})

	assert.Equal(t, toolStatusCompleted, outcome.Status)
	require.Len(t, outcome.Actions, 1)
	assert.Equal(t, "accept_field", outcome.Actions[0].Type)
}
