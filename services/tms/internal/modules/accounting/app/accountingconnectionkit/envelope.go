package accountingconnectionkit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection/accountingconnectiondb"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	clientStateField = "client_state"
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
					ConnectionAAD(&accountingconnection.AccountingConnection{
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
					AppAAD(&accountingconnection.AccountingAppCredential{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}, field).SecretScope(),
				}
			},
		}
	}

	return []kernelports.EnvelopeColumn{
		connectionColumn(conn.AccessTokenCiphertext, AccessTokenField),
		connectionColumn(conn.RefreshTokenCiphertext, RefreshTokenField),
		connectionColumn(conn.ClientKeyCiphertext, ClientKeyField),
		appColumn(app.ClientSecretCiphertext, ClientSecretField),
		appColumn(app.WebhookVerifierCiphertext, WebhookVerifierField),
		{
			Name:  "accounting pending certificate key",
			Value: accountingconnectiondb.AccountingPendingCertificates.PrivateKeyCiphertext,
			Scopes: func(row *kernelports.EnvelopeRow, _ string) []kernelports.SecretScope {
				return []kernelports.SecretScope{
					PendingKeyAAD(&accountingconnection.AccountingPendingCertificate{
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
					ClientStateAAD(&accountingconnection.AccountingWebhookSubscription{
						ID:             pulid.ID(row.ID),
						OrganizationID: row.Tenant.OrgID,
						BusinessUnitID: row.Tenant.BuID,
					}).SecretScope(),
				}
			},
		},
	}
}

func ClientStateAAD(row *accountingconnection.AccountingWebhookSubscription) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingWebhookState,
		OrganizationID: row.OrganizationID,
		BusinessUnitID: row.BusinessUnitID,
		ResourceID:     row.ID.String() + ":" + clientStateField,
	}
}

func ConnectionScope(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) context.Context {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
	})
}
