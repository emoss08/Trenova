package mfaservice

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	cols := iamcol.MFAAuthenticatorColumns
	return []kernelports.EnvelopeColumn{{
		Name:           "MFA authenticator secret",
		Table:          iamcol.MFAAuthenticatorTable,
		ID:             cols.ID,
		OrganizationID: cols.OrganizationID,
		BusinessUnitID: buncolgen.Column{},
		Value:          cols.SecretCipher,
		Extra:          []buncolgen.Column{cols.UserID},
		Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
			userID := pulid.ID(row.Extra[cols.UserID.Name])
			return []kernelports.SecretScope{secretAAD(row.Tenant.OrgID, userID).SecretScope()}
		},
	}}
}
