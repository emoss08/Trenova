package accountingconnectionservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	conn := accountingsyncdb.AccountingConnections
	app := accountingsyncdb.AccountingAppCredentials

	connectionColumn := func(value keel.Expr, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:  "accounting connection " + field,
			Value: value,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{connectionAAD(&accountingsync.AccountingConnection{
					ID:             pulid.ID(row.ID),
					OrganizationID: row.Tenant.OrgID,
					BusinessUnitID: row.Tenant.BuID,
				}, field).SecretScope()}
			},
		}
	}
	appColumn := func(value keel.Expr, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:  "accounting app " + field,
			Value: value,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{appAAD(&accountingsync.AccountingAppCredential{
					ID:             pulid.ID(row.ID),
					OrganizationID: row.Tenant.OrgID,
					BusinessUnitID: row.Tenant.BuID,
				}, field).SecretScope()}
			},
		}
	}

	return []kernelports.EnvelopeColumn{
		connectionColumn(conn.AccessTokenCiphertext, accessTokenField),
		connectionColumn(conn.RefreshTokenCiphertext, refreshTokenField),
		connectionColumn(conn.ClientKeyCiphertext, clientKeyField),
		appColumn(app.ClientSecretCiphertext, clientSecretField),
		appColumn(app.WebhookVerifierCiphertext, webhookVerifierField),
		{
			Name:  "accounting pending certificate key",
			Value: accountingsyncdb.AccountingPendingCertificates.PrivateKeyCiphertext,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					pendingKeyAAD(&accountingsync.AccountingPendingCertificate{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}).SecretScope(),
				}
			},
		},
		{
			Name:  "accounting webhook client state",
			Value: accountingsyncdb.AccountingWebhookSubscriptions.ClientStateCiphertext,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					clientStateAAD(&accountingsync.AccountingWebhookSubscription{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}).SecretScope(),
				}
			},
		},
	}
}
