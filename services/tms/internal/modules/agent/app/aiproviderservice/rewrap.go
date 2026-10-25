package aiproviderservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	p := aiproviderdb.Providers
	keyColumn := func(name string, value keel.Expr) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:   name,
			Value:  value,
			Rebind: true,
			Scopes: providerKeyScopes,
		}
	}
	return []kernelports.EnvelopeColumn{
		keyColumn("AI provider API key", p.APIKey),
		keyColumn("AI provider previous API key", p.PreviousAPIKey),
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
