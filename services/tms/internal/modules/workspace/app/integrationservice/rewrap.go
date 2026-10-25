package integrationservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	i := integrationdb.Integrations
	integrationType := kernelports.EnvelopeColumnName(i.Type)
	return []kernelports.EnvelopeColumn{{
		Name:    "integration secrets",
		Value:   i.Configuration,
		Extra:   []keel.Expr{i.Type},
		JSONMap: true,
		Rebind:  true,
		Scopes: func(row *kernelports.EnvelopeRow, field string) []kernelports.SecretScope {
			typ := integration.Type(row.Extra[integrationType])
			scope := newSecretScope(row.Tenant, typ)
			return scope.SecretScopes(field)
		},
	}}
}
