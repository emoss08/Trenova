package fieldgate

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrievalAccessChecksEachRecordOnceAndHonoursTheCeiling(t *testing.T) {
	t.Parallel()

	permissions := &countingPermissions{FakePermissions: querytooltest.FakePermissions{
		Readable: map[string]*permissionports.ResourcePermissionDetail{
			permission.ResourceDocument.String(): {
				Resource:       permission.ResourceDocument.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
			permission.ResourceShipment.String(): {
				Resource:       permission.ResourceShipment.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
			permission.ResourceWorker.String(): {
				Resource:       permission.ResourceWorker.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
		},
	}}
	access := NewAccess(permissions).ForRetrieval(querytooltest.TestParams(nil))
	shipment := pulid.MustNew("shp_").String()

	assert.True(t, access.MayReadRecord(t.Context(), permission.ResourceShipment, shipment))
	assert.True(t, access.MayReadRecord(t.Context(), permission.ResourceShipment, shipment))
	assert.False(t, access.MayReadRecord(t.Context(), permission.ResourceCustomer, "cus_1"))
	require.Len(t, permissions.requests, 3, "one resource check and one record check, remembered")
	assert.Nil(t, permissions.requests[0].ResourceID)
	require.NotNil(t, permissions.requests[1].ResourceID)
	assert.Equal(t, shipment, permissions.requests[1].ResourceID.String())
	assert.Equal(t, permission.ResourceCustomer.String(), permissions.requests[2].Resource)

	assert.True(t, access.ShowsRecordText(t.Context(), permission.ResourceShipment))
	assert.False(t, access.ShowsRecordText(t.Context(), permission.ResourceWorker),
		"a worker's records are Restricted and the ceiling is Internal")
	assert.False(t, access.ShowsRecordText(t.Context(), permission.ResourceUser))
	assert.True(t, access.ShowsField(t.Context(), permission.ResourceDocument, "contentText"))
}

type countingPermissions struct {
	querytooltest.FakePermissions

	requests []*permissionports.PermissionCheckRequest
}

func (c *countingPermissions) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	c.requests = append(c.requests, req)

	return c.FakePermissions.Check(ctx, req)
}
