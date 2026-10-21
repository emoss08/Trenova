package integrationservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/integrationcol"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := integrationcol.IntegrationColumns
	return []kernelports.EnvelopeColumn{{
		Name:           "integration secrets",
		Table:          integrationcol.IntegrationTable,
		ID:             cols.ID,
		OrganizationID: cols.OrganizationID,
		BusinessUnitID: cols.BusinessUnitID,
		Value:          cols.Configuration,
		Extra:          []buncolgen.Column{cols.Type},
		JSONMap:        true,
		Rebind:         true,
		Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
			typ := integration.Type(row.Extra[cols.Type.Name])
			scope := newSecretScope(row.Tenant, typ)
			return scope.SecretScopes(field)
		},
	}}
}
