package authservice

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
)

var knownSSOProviders = []tenant.SSOProvider{tenant.SSOProviderAzureAD, tenant.SSOProviderOkta}

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	idp := iamcol.IdentityProviderColumns
	sso := tenantcol.SSOConfigColumns
	return []kernelports.EnvelopeColumn{
		{
			Name:           "identity provider client secret",
			Table:          iamcol.IdentityProviderTable,
			ID:             idp.ID,
			OrganizationID: idp.OrganizationID,
			BusinessUnitID: idp.BusinessUnitID,
			Value:          idp.OIDCClientSecret,
			Extra:          []buncolgen.Column{idp.Slug},
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return ssoSecretScopes(row, identityProviderResourceIDs(row.Extra[idp.Slug.Name]))
			},
		},
		{
			Name:           "SSO configuration client secret",
			Table:          tenantcol.SSOConfigTable,
			ID:             sso.ID,
			OrganizationID: sso.OrganizationID,
			BusinessUnitID: sso.BusinessUnitID,
			Value:          sso.OIDCClientSecret,
			Extra:          []buncolgen.Column{sso.Provider},
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				provider := tenant.SSOProvider(row.Extra[sso.Provider.Name])
				return ssoSecretScopes(row, ssoSecretAADResourceIDs(provider))
			},
		},
	}
}

func identityProviderResourceIDs(slug string) []string {
	ids := make([]string, 0, 3)
	for _, provider := range knownSSOProviders {
		if strings.EqualFold(string(provider), strings.TrimSpace(slug)) {
			ids = append(ids, ssoSecretAADResourceIDs(provider)...)
		}
	}
	if !slices.Contains(ids, slug) {
		ids = append(ids, slug)
	}
	return ids
}

func ssoSecretScopes(row *kernelports.EnvelopeRow, resourceIDs []string) []kernelports.SecretScope {
	scopes := make([]kernelports.SecretScope, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		scopes = append(scopes, ssoSecretAAD(row.Tenant, resourceID).SecretScope())
	}
	return scopes
}
