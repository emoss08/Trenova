package accountingconnectionservice

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection/accountingconnectiondb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	conn := accountingconnectiondb.AccountingConnections
	app := accountingconnectiondb.AccountingAppCredentials

	connectionColumn := func(value keel.Expr, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:  "accounting connection " + field,
			Value: value,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					connectionAAD(&accountingconnection.AccountingConnection{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}, field).SecretScope(),
				}
			},
		}
	}
	appColumn := func(value keel.Expr, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:  "accounting app " + field,
			Value: value,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					appAAD(&accountingconnection.AccountingAppCredential{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}, field).SecretScope(),
				}
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
			Value: accountingconnectiondb.AccountingPendingCertificates.PrivateKeyCiphertext,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					pendingKeyAAD(&accountingconnection.AccountingPendingCertificate{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}).SecretScope(),
				}
			},
		},
		{
			Name:  "accounting webhook client state",
			Value: accountingconnectiondb.AccountingWebhookSubscriptions.ClientStateCiphertext,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					clientStateAAD(&accountingconnection.AccountingWebhookSubscription{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}).SecretScope(),
				}
			},
		},
	}
}
