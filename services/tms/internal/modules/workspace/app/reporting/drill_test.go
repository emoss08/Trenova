package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type confidentialReader struct {
	mocks.AllowAllPermissionEngine
}

func (*confidentialReader) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		DataScope:      permission.DataScopeOrganization,
		MaxSensitivity: permission.SensitivityConfidential,
	}, nil
}

type groupAccess struct {
	visible permission.FieldGroupSet
}

func (g groupAccess) Visible(
	context.Context,
	*permissionports.RequestActor,
) permission.FieldGroupSet {
	return g.visible
}

func TestDetailColumns_LeavesOutProtectedFieldsTheReaderCannotSee(t *testing.T) {
	t.Parallel()

	entity, ok := reportcatalog.Default.Entity("worker_profile")
	require.True(t, ok)
	req := &DrillThroughRequest{
		Request: Request{TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		}},
		Definition: &report.Definition{IRVersion: report.CurrentIRVersion, Entity: entity.Key},
	}
	keys := func(visible permission.FieldGroupSet) []string {
		svc := &Service{
			l:            zap.NewNop(),
			perms:        &confidentialReader{},
			fieldAccess:  groupAccess{visible: visible},
			permRegistry: permission.NewRegistry(),
		}
		columns, err := svc.detailColumns(t.Context(), req, entity)
		require.NoError(t, err)
		out := make([]string, 0, len(columns))
		for i := range columns {
			out = append(out, columns[i].Ref.Field)
		}
		return out
	}

	withheld := keys(0)
	for _, key := range withheld {
		field, _ := entity.Field(key)
		assert.Emptyf(t, field.Protected, "%s is protected and was offered without its group", key)
	}

	granted := keys(permission.AllFieldGroups())
	assert.Contains(t, granted, "dob", "a reader with personal details drills into them")
}
