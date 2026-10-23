package reporting

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCatalog_DecidesEveryEntityForTheCaller(t *testing.T) {
	t.Parallel()

	engine := permtest.AllowAll()
	engine.Overrides["customer"] = permtest.Grant{MaxSensitivity: permission.SensitivityInternal}
	registry := permission.NewRegistry()
	svc := &Service{
		l: zap.NewNop(),
		access: reportaccess.New(reportaccess.Params{
			Engine:   engine,
			Registry: registry,
			Views:    refview.NewPort(refview.New(registry), engine),
			Fields:   groupAccess{visible: 0},
		}),
	}

	catalog, err := svc.Catalog(t.Context(), &Request{TenantInfo: pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}})
	require.NoError(t, err)

	assert.Equal(t, reportcatalog.Default.Version, catalog.Version)
	require.Len(t, catalog.Entities, len(reportcatalog.Default.Entities))
	byKey := make(map[string]reportaccess.Entity, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		byKey[entity.Entity.Key] = entity
	}

	assert.True(t, byKey["shipment"].Readable)
	assert.True(t, byKey["shipment"].Exportable)
	customer := byKey["customer"]
	assert.False(t, customer.Readable)
	assert.True(t, customer.Usable())
	require.Len(t, customer.Fields, len(customer.Entity.Fields))
	require.Len(t, customer.Edges, len(customer.Entity.Edges))
}
