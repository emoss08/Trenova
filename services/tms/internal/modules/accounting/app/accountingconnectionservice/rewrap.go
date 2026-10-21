package accountingconnectionservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"
	"github.com/emoss08/trenova/shared/pulid"
)

func EnvelopeColumns() []kernelports.EnvelopeColumn {
	conn := accountingsynccol.AccountingConnectionColumns
	app := accountingsynccol.AccountingAppCredentialColumns
	pending := accountingsynccol.AccountingPendingCertificateColumns
	subscription := accountingsynccol.AccountingWebhookSubscriptionColumns

	connectionColumn := func(value buncolgen.Column, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:           "accounting connection " + field,
			Table:          accountingsynccol.AccountingConnectionTable,
			ID:             conn.ID,
			OrganizationID: conn.OrganizationID,
			BusinessUnitID: conn.BusinessUnitID,
			Value:          value,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{connectionAAD(&accountingsync.AccountingConnection{
					ID:             pulid.ID(row.ID),
					OrganizationID: row.Tenant.OrgID,
					BusinessUnitID: row.Tenant.BuID,
				}, field).SecretScope()}
			},
		}
	}
	appColumn := func(value buncolgen.Column, field string) kernelports.EnvelopeColumn {
		return kernelports.EnvelopeColumn{
			Name:           "accounting app " + field,
			Table:          accountingsynccol.AccountingAppCredentialTable,
			ID:             app.ID,
			OrganizationID: app.OrganizationID,
			BusinessUnitID: app.BusinessUnitID,
			Value:          value,
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
			Name:           "accounting pending certificate key",
			Table:          accountingsynccol.AccountingPendingCertificateTable,
			ID:             pending.ID,
			OrganizationID: pending.OrganizationID,
			BusinessUnitID: pending.BusinessUnitID,
			Value:          pending.PrivateKeyCiphertext,
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
			Name:           "accounting webhook client state",
			Table:          accountingsynccol.AccountingWebhookSubscriptionTable,
			ID:             subscription.ID,
			OrganizationID: subscription.OrganizationID,
			BusinessUnitID: subscription.BusinessUnitID,
			Value:          subscription.ClientStateCiphertext,
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
