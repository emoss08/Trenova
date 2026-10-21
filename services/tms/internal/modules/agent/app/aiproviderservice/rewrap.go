package aiproviderservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/aiprovidercol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := aiprovidercol.ProviderColumns
	keyColumn := func(name string, value buncolgen.Column) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:           name,
			Table:          aiprovidercol.ProviderTable,
			ID:             cols.ID,
			OrganizationID: cols.OrganizationID,
			BusinessUnitID: cols.BusinessUnitID,
			Value:          value,
			Rebind:         true,
			Scopes:         providerKeyScopes,
		}
	}
	return []kernelports.EnvelopeColumn{
		keyColumn("AI provider API key", cols.APIKey),
		keyColumn("AI provider previous API key", cols.PreviousAPIKey),
	}
}

func providerKeyScopes(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
	bound := credentialAAD(&aiprovider.Provider{
		ID:             pulid.ID(row.ID),
		OrganizationID: row.Tenant.OrgID,
		BusinessUnitID: row.Tenant.BuID,
	})
	return []kernelports.SecretScope{bound.SecretScope(), {}}
}
