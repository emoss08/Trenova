package organizationservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingUniquenessChecker struct {
	system bool
	req    *validationframework.UniquenessRequest
}

func (c *capturingUniquenessChecker) CheckUniqueness(
	ctx context.Context,
	req *validationframework.UniquenessRequest,
) (bool, error) {
	c.system = dbscope.IsSystem(ctx)
	c.req = req
	return true, nil
}

func TestBusinessUnitWideCheckerSearchesEveryOrganizationOfTheCallersBusinessUnit(t *testing.T) {
	t.Parallel()

	tenantScope := dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}
	inner := &capturingUniquenessChecker{}
	req := &validationframework.UniquenessRequest{
		TableName: "organizations",
		Fields:    []validationframework.FieldCheck{{Column: "name", Value: "Acme"}},
	}

	exists, err := businessUnitWideChecker{checker: inner}.CheckUniqueness(
		dbscope.WithTenant(t.Context(), tenantScope),
		req,
	)

	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, inner.system)
	require.NotNil(t, inner.req)
	assert.Equal(t, tenantScope.BusinessUnitID, inner.req.BusinessUnitID)
	assert.True(t, inner.req.OrganizationID.IsNil())
	assert.True(t, req.BusinessUnitID.IsNil())
}

func TestBusinessUnitWideCheckerKeepsTheRequestUnderASystemScope(t *testing.T) {
	t.Parallel()

	inner := &capturingUniquenessChecker{}
	req := &validationframework.UniquenessRequest{TableName: "organizations"}

	_, err := businessUnitWideChecker{checker: inner}.CheckUniqueness(
		dbscope.WithSystem(t.Context(), "test"),
		req,
	)

	require.NoError(t, err)
	assert.True(t, inner.system)
	assert.True(t, inner.req.BusinessUnitID.IsNil())
}
