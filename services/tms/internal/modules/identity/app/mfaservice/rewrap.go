package mfaservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	mfa := iamdb.MFAAuthenticators
	user := kernelports.EnvelopeColumnName(mfa.UserID)
	return []kernelports.EnvelopeColumn{{
		Name:  "MFA authenticator secret",
		Value: mfa.SecretCipher,
		Extra: []keel.Expr{mfa.UserID},
		Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
			userID := pulid.ID(row.Extra[user])
			return []kernelports.SecretScope{secretAAD(row.Tenant.OrgID, userID).SecretScope()}
		},
	}}
}
