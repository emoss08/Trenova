package encryption

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvelopeSourcesAreComplete(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	sources := envelopeSources(true)
	columns := make([]kernelports.EnvelopeColumn, 0, len(sources.Columns)+len(sources.Objects))
	columns = append(columns, sources.Columns...)
	for _, object := range sources.Objects {
		columns = append(columns, object.Column)
		if object.Version != nil {
			assert.Contains(t, object.Column.Extra, *object.Version, object.Column.Name)
		}
	}

	names := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		require.NotEmpty(t, column.Name)
		_, duplicate := names[column.Name]
		assert.False(t, duplicate, "duplicate source %q", column.Name)
		names[column.Name] = struct{}{}

		assert.NotEmpty(t, column.Table.Name, column.Name)
		assert.NotEmpty(t, column.ID.Name, column.Name)
		assert.NotEmpty(t, column.OrganizationID.Name, column.Name)
		assert.NotEmpty(t, column.Value.Name, column.Name)
		require.NotNil(t, column.Scopes, column.Name)

		row := &kernelports.EnvelopeRow{
			ID:     pulid.MustNew("row_").String(),
			Tenant: tenant,
			Extra:  map[string]string{},
		}
		for _, extra := range column.Extra {
			row.Extra[extra.Name] = "value"
		}
		scopes := column.Scopes(row, "field")
		require.NotEmpty(t, scopes, column.Name)
		if scopes[0].Purpose != "" {
			assert.Equal(t, tenant.OrgID, scopes[0].Tenant.OrgID, column.Name)
		}
	}

	assert.Contains(t, names, "AI provider API key")
	assert.Contains(t, names, "AI provider previous API key")
	assert.Contains(t, names, "document files")
	assert.Len(t, envelopeSources(false).Objects, 0)
}
