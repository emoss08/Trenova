package agentdefinitionhandler

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A save says who may use the agent with both fields or with neither. Neither
keeps who may use it; both set it with the save; one without the other is
refused on the missing field rather than read as "no roles" or "Everyone".
*/
func TestSaveAgentRequest_AccessIsBothFieldsOrNeither(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	roles := []pulid.ID{pulid.MustNew("rol_")}
	restricted := agentdefinition.AccessRoles

	req, err := (&saveAgentRequest{Name: "Payroll"}).toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	assert.Nil(t, req.Access, "a body that says nothing keeps who may use the agent")

	req, err = (&saveAgentRequest{
		Name:          "Payroll",
		AccessMode:    &restricted,
		AccessRoleIDs: &roles,
	}).toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	require.NotNil(t, req.Access)
	assert.Equal(t, agentdefinition.AccessRoles, req.Access.Mode)
	assert.Equal(t, roles, req.Access.RoleIDs)

	var invalid *errortypes.Error
	_, err = (&saveAgentRequest{AccessMode: &restricted}).toSaveRequest(pulid.Nil, tenant)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "accessRoleIds", invalid.Field)

	_, err = (&saveAgentRequest{AccessRoleIDs: &roles}).toSaveRequest(pulid.Nil, tenant)
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "accessMode", invalid.Field)
}

func TestSaveAgentRequest_CarriesTheMemoryBudget(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	budget := 9000

	req, err := (&saveAgentRequest{Name: "Rates", MemoryTokenBudget: &budget}).
		toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	require.NotNil(t, req.MemoryTokenBudget)
	assert.Equal(t, 9000, *req.MemoryTokenBudget)

	req, err = (&saveAgentRequest{Name: "Rates"}).toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	assert.Nil(t, req.MemoryTokenBudget, "no budget is the default")
}

func TestSaveAgentRequest_CarriesTheDataAccess(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	req, err := (&saveAgentRequest{
		Name:              "Payroll",
		DataAccessCeiling: agentdefinition.DataAccessRestricted,
	}).toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	assert.Equal(t, agentdefinition.DataAccessRestricted, req.DataAccessCeiling)

	req, err = (&saveAgentRequest{Name: "Payroll"}).toSaveRequest(pulid.Nil, tenant)
	require.NoError(t, err)
	assert.Empty(t, req.DataAccessCeiling, "a body that says nothing keeps what the agent has")
}
