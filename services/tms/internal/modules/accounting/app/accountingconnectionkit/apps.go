package accountingconnectionkit

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	ClientSecretField    = "client_secret"
	WebhookVerifierField = "webhook_verifier"
)

var ErrAppChanged = errors.New(
	"the app this company was connected through was changed or removed; reconnect to resume",
)

func (k *Kit) Provider(
	typ integration.Type,
) (accountingconnectorports.AccountingProvider, error) {
	provider, ok := k.connectors.For(typ)
	if !ok || !accountingprovider.SupportsAccountingSync(typ) {
		return nil, errortypes.NewValidationError(
			"integrationType",
			errortypes.ErrInvalid,
			"{0} is not an accounting system Trenova can sync with",
			string(typ),
		)
	}
	return provider, nil
}

func errAppNotSetUp(provider string) error {
	return errortypes.NewBusinessError(
		"{0} has no app to connect through. Add your own {0} app's keys on this page, or ask the server's administrator to configure one for the whole server.",
		provider,
	)
}

func ErrNoRedirect(provider string) error {
	return errortypes.NewBusinessError(
		"This Trenova server has no web address configured (app.webBaseUrl), so {0} cannot send people back after they sign in. Ask the server's administrator to set it.",
		provider,
	)
}

func AppAAD(
	cred *accountingconnection.AccountingAppCredential,
	field string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingAppSecret,
		OrganizationID: cred.OrganizationID,
		BusinessUnitID: cred.BusinessUnitID,
		ResourceID:     cred.ID.String() + ":" + field,
	}
}

func (k *Kit) TenantCredential(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnection.AccountingAppCredential, bool, error) {
	cred, err := k.apps.GetByType(ctx, accountingconnectionports.GetAccountingAppCredentialRequest{
		TenantInfo:      tenantInfo,
		IntegrationType: typ,
	})
	switch {
	case err == nil:
		return cred, true, nil
	case errortypes.IsNotFoundError(err):
		return nil, false, nil
	default:
		return nil, false, err
	}
}

func (k *Kit) OpenCredential(
	cred *accountingconnection.AccountingAppCredential,
) (*accountingconnectorports.AccountingApp, error) {
	secret, err := k.encryption.DecryptStringWithAAD(
		cred.ClientSecretCiphertext,
		AppAAD(cred, ClientSecretField),
	)
	if err != nil {
		return nil, err
	}
	app := &accountingconnectorports.AccountingApp{
		Source:       accountingconnection.AppSourceTenant,
		Environment:  cred.Environment,
		ClientID:     cred.ClientID,
		ClientSecret: secret,
	}
	if cred.HasWebhookVerifier() {
		verifier, openErr := k.encryption.DecryptStringWithAAD(
			cred.WebhookVerifierCiphertext,
			AppAAD(cred, WebhookVerifierField),
		)
		if openErr != nil {
			return nil, openErr
		}
		app.WebhookVerifierToken = verifier
	}
	return app, nil
}

func (k *Kit) AppForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider accountingconnectorports.AccountingProvider,
) (*accountingconnectorports.AccountingApp, error) {
	name := accountingprovider.ProviderName(provider.IntegrationType())
	cred, found, err := k.TenantCredential(ctx, tenantInfo, provider.IntegrationType())
	if err != nil {
		return nil, err
	}
	if found {
		app, openErr := k.OpenCredential(cred)
		if openErr != nil {
			return nil, errortypes.NewBusinessError(
				"The saved {0} app keys could not be read. Enter them again.",
				name,
			).WithInternal(openErr)
		}
		return app, nil
	}
	if app, ok := provider.InstanceApp(); ok {
		return app, nil
	}
	return nil, errAppNotSetUp(name)
}

func (k *Kit) AppForConnection(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	conn *accountingconnection.AccountingConnection,
) (*accountingconnectorports.AccountingApp, error) {
	var app *accountingconnectorports.AccountingApp
	switch conn.AppSource {
	case accountingconnection.AppSourceTenant:
		cred, found, err := k.TenantCredential(
			ctx,
			pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
			conn.IntegrationType,
		)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrAppChanged
		}
		if app, err = k.OpenCredential(cred); err != nil {
			return nil, err
		}
	case accountingconnection.AppSourceInstance:
		instance, ok := provider.InstanceApp()
		if !ok {
			return nil, ErrAppChanged
		}
		app = instance
	default:
		return nil, ErrAppChanged
	}
	if !conn.ConnectedThrough(app.Identity()) {
		return nil, ErrAppChanged
	}
	return app, nil
}

func (k *Kit) ConnectorFor(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	conn *accountingconnection.AccountingConnection,
) (accountingconnectorports.AccountingConnector, error) {
	app, err := k.AppForConnection(ctx, provider, conn)
	if err != nil {
		return nil, err
	}
	return provider.Bind(app)
}

func (k *Kit) AppSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider accountingconnectorports.AccountingProvider,
) (*accountingconnectionports.AccountingAppSettings, error) {
	typ := provider.IntegrationType()
	profile := accountingprovider.MustProfile(typ)
	settings := &accountingconnectionports.AccountingAppSettings{
		RedirectURL: provider.RedirectURL(),
		WebhookPath: profile.WebhookPath(),
	}
	if instance, ok := provider.InstanceApp(); ok {
		settings.InstanceAppAvailable = true
		settings.InstanceEnvironment = instance.Environment
		settings.ActiveSource = accountingconnection.AppSourceInstance
	}

	cred, found, err := k.TenantCredential(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if found {
		settings.TenantApp = cred
		settings.ActiveSource = accountingconnection.AppSourceTenant
		settings.WebhookPath = profile.AppWebhookPath(cred.ID.String())
	}
	return settings, nil
}
