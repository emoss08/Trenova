package reporting

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/dbtype"
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

func (c *confidentialReader) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, c, userID, orgID, resources)
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
		engine := &confidentialReader{}
		svc := &Service{
			l:     zap.NewNop(),
			perms: engine,
			access: reportaccess.New(reportaccess.Params{
				Engine: engine,
				Fields: groupAccess{visible: visible},
			}),
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

func TestDrillDimensionFilters_ABucketCoversItsLocalPeriod(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	cases := []struct {
		bucket report.DateBucket
		start  time.Time
		end    time.Time
	}{
		{
			bucket: report.DateBucketMonth,
			start:  time.Date(2026, 3, 1, 0, 0, 0, 0, chicago),
			end:    time.Date(2026, 4, 1, 0, 0, 0, 0, chicago),
		},
		{
			bucket: report.DateBucketDay,
			start:  time.Date(2026, 3, 8, 0, 0, 0, 0, chicago),
			end:    time.Date(2026, 3, 9, 0, 0, 0, 0, chicago),
		},
		{
			bucket: report.DateBucketWeek,
			start:  time.Date(2026, 10, 26, 0, 0, 0, 0, chicago),
			end:    time.Date(2026, 11, 2, 0, 0, 0, 0, chicago),
		},
		{
			bucket: report.DateBucketYear,
			start:  time.Date(2026, 1, 1, 0, 0, 0, 0, chicago),
			end:    time.Date(2027, 1, 1, 0, 0, 0, 0, chicago),
		},
	}

	for _, tc := range cases {
		def := &report.Definition{
			Columns: []report.ColumnSpec{{
				ID:     "c_period",
				Ref:    report.FieldRef{Field: "createdAt"},
				Kind:   report.ColumnKindDimension,
				Bucket: tc.bucket,
			}},
		}

		filters, filterErr := drillDimensionFilters(def, &DrillValue{
			ColumnID: "c_period",
			Value:    float64(tc.start.Unix()),
		}, chicago)
		require.NoError(t, filterErr, tc.bucket)
		require.Len(t, filters, 2, tc.bucket)
		assert.Equal(t, dbtype.OpGreaterThanOrEqual, filters[0].Operator, tc.bucket)
		assert.Equal(t, tc.start.Unix(), filters[0].Value, tc.bucket)
		assert.Equal(t, dbtype.OpLessThan, filters[1].Operator, tc.bucket)
		assert.Equal(t, tc.end.Unix(), filters[1].Value, tc.bucket)
	}
}
