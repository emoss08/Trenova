package authservice

import (
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

var knownSSOProviders = []tenant.SSOProvider{tenant.SSOProviderAzureAD, tenant.SSOProviderOkta}

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	idp := iamdb.IdentityProviders
	sso := tenantdb.SSOConfigs
	slug := kernelports.EnvelopeColumnName(idp.Slug)
	provider := kernelports.EnvelopeColumnName(sso.Provider)
	return []kernelports.EnvelopeColumn{
		{
			Name:  "identity provider client secret",
			Value: idp.OIDCClientSecret,
			Extra: []keel.Expr{idp.Slug},
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return ssoSecretScopes(row, identityProviderResourceIDs(row.Extra[slug]))
			},
		},
		{
			Name:  "SSO configuration client secret",
			Value: sso.OIDCClientSecret,
			Extra: []keel.Expr{sso.Provider},
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return ssoSecretScopes(
					row,
					ssoSecretAADResourceIDs(tenant.SSOProvider(row.Extra[provider])),
				)
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
