package reporting

import (
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type engineOptions struct {
	denied  []permission.Resource
	groups  permission.FieldGroupSet
	failure error
}

func engineService(t *testing.T, opts engineOptions) (*Service, *reportingtest.Engine) {
	t.Helper()
	catalog := reportingtest.ReportCatalog(t)
	reader := reportingtest.NewPermissionReader(opts.denied)
	registry := permissioncatalog.NewRegistry()
	access, err := reportaccess.New(reportaccess.Params{
		Catalog:  catalog,
		Engine:   reader,
		Registry: registry,
		Views:    refview.NewPort(refview.New(registry), reader),
		Fields:   reportingtest.FieldGroups{Groups: opts.groups},
	})
	require.NoError(t, err)
	engine := reportingtest.NewEngine(catalog, opts.failure)
	return &Service{
		l:       zap.NewNop(),
		engine:  engine,
		perms:   reader,
		access:  access,
		orgRepo: reportingtest.Organizations{},
		cfg:     &reportingconfig.Config{PreviewRowLimit: 40},
	}, engine
}

func storedRunDefinition() *ir.Definition {
	return &ir.Definition{
		Version: ir.Version,
		Entity:  "shipment",
		Columns: []ir.Column{{
			ID:        "c_customer",
			Dimension: &ir.Dimension{Field: "customer.name"},
		}},
	}
}

func testUserID() pulid.ID {
	return pulid.MustNew("usr_")
}
