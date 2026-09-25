package accountingconnectionservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	clientSecretField    = "client_secret"
	webhookVerifierField = "webhook_verifier"
)

var errAppChanged = errors.New(
	"the app this company was connected through was changed or removed; reconnect to resume",
)

func (s *Service) provider(typ integration.Type) (services.AccountingProvider, error) {
	provider, ok := s.connectors.For(typ)
	if !ok || !accountingsync.SupportsAccountingSync(typ) {
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

func errNoRedirect(provider string) error {
	return errortypes.NewBusinessError(
		"This Trenova server has no web address configured (app.webBaseUrl), so {0} cannot send people back after they sign in. Ask the server's administrator to set it.",
		provider,
	)
}

func (s *Service) appAAD(
	cred *accountingsync.AccountingAppCredential,
	field string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingAppSecret,
		OrganizationID: cred.OrganizationID,
		BusinessUnitID: cred.BusinessUnitID,
		ResourceID:     cred.ID.String() + ":" + field,
	}
}

func (s *Service) tenantCredential(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingAppCredential, bool, error) {
	cred, err := s.apps.GetByType(ctx, repositories.GetAccountingAppCredentialRequest{
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

func (s *Service) openCredential(
	cred *accountingsync.AccountingAppCredential,
) (*services.AccountingApp, error) {
	secret, err := s.encryption.DecryptStringWithAAD(
		cred.ClientSecretCiphertext,
		s.appAAD(cred, clientSecretField),
	)
	if err != nil {
		return nil, err
	}
	app := &services.AccountingApp{
		Source:       accountingsync.AppSourceTenant,
		Environment:  cred.Environment,
		ClientID:     cred.ClientID,
		ClientSecret: secret,
	}
	if cred.HasWebhookVerifier() {
		verifier, openErr := s.encryption.DecryptStringWithAAD(
			cred.WebhookVerifierCiphertext,
			s.appAAD(cred, webhookVerifierField),
		)
		if openErr != nil {
			return nil, openErr
		}
		app.WebhookVerifierToken = verifier
	}
	return app, nil
}

func (s *Service) appForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.AccountingProvider,
) (*services.AccountingApp, error) {
	name := accountingsync.ProviderName(provider.IntegrationType())
	cred, found, err := s.tenantCredential(ctx, tenantInfo, provider.IntegrationType())
	if err != nil {
		return nil, err
	}
	if found {
		app, openErr := s.openCredential(cred)
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

func (s *Service) appForConnection(
	ctx context.Context,
	provider services.AccountingProvider,
	conn *accountingsync.AccountingConnection,
) (*services.AccountingApp, error) {
	var app *services.AccountingApp
	switch conn.AppSource {
	case accountingsync.AppSourceTenant:
		cred, found, err := s.tenantCredential(
			ctx,
			pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
			conn.IntegrationType,
		)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errAppChanged
		}
		if app, err = s.openCredential(cred); err != nil {
			return nil, err
		}
	case accountingsync.AppSourceInstance:
		instance, ok := provider.InstanceApp()
		if !ok {
			return nil, errAppChanged
		}
		app = instance
	default:
		return nil, errAppChanged
	}
	if !conn.ConnectedThrough(app.Identity()) {
		return nil, errAppChanged
	}
	return app, nil
}

func (s *Service) connectorFor(
	ctx context.Context,
	provider services.AccountingProvider,
	conn *accountingsync.AccountingConnection,
) (services.AccountingConnector, error) {
	app, err := s.appForConnection(ctx, provider, conn)
	if err != nil {
		return nil, err
	}
	return provider.Bind(app)
}

func (s *Service) appSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.AccountingProvider,
) (*services.AccountingAppSettings, error) {
	typ := provider.IntegrationType()
	settings := &services.AccountingAppSettings{
		RedirectURL: provider.RedirectURL(),
		WebhookPath: accountingsync.WebhookPath(typ),
	}
	if instance, ok := provider.InstanceApp(); ok {
		settings.InstanceAppAvailable = true
		settings.InstanceEnvironment = instance.Environment
		settings.ActiveSource = accountingsync.AppSourceInstance
	}

	cred, found, err := s.tenantCredential(ctx, tenantInfo, typ)
	if err != nil {
		return nil, err
	}
	if found {
		settings.TenantApp = cred
		settings.ActiveSource = accountingsync.AppSourceTenant
	}
	return settings, nil
}

func validateSaveApp(req *services.SaveAccountingAppRequest) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.ClientID) == "" {
		multiErr.Add("clientId", errortypes.ErrRequired, "Client ID is required")
	}
	if !req.Environment.IsValid() {
		multiErr.Add(
			"environment",
			errortypes.ErrInvalid,
			"Environment must be Sandbox or Production",
		)
	}
	if req.ClearWebhookVerifierToken && strings.TrimSpace(req.WebhookVerifierToken) != "" {
		multiErr.Add(
			"webhookVerifierToken",
			errortypes.ErrInvalid,
			"Enter a verifier token or clear it, not both",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type appChange struct {
	req      *services.SaveAccountingAppRequest
	existing *accountingsync.AccountingAppCredential
	conn     *accountingsync.AccountingConnection
	provider string
}

func (c *appChange) guardActiveConnection() error {
	if c.conn == nil || !c.conn.IsActive() {
		return nil
	}
	if c.conn.AppSource != accountingsync.AppSourceTenant {
		return errortypes.NewBusinessError(
			"{0} company {1} is connected through this server's app. Disconnect it before switching to your own app.",
			c.provider,
			c.conn.ExternalCompanyName,
		)
	}
	fingerprint := accountingsync.AppFingerprint(c.req.Environment, c.req.ClientID)
	if c.conn.AppFingerprint != fingerprint {
		return errortypes.NewBusinessError(
			"Disconnect {0} company {1} before changing the client ID or environment. Its authorization belongs to the app it was connected through; you can still replace the client secret or the webhook verifier token.",
			c.provider,
			c.conn.ExternalCompanyName,
		)
	}
	return nil
}

func (s *Service) activeConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.connections.GetByType(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenantInfo,
		IntegrationType: typ,
	})
	switch {
	case err == nil:
		return conn, nil
	case errortypes.IsNotFoundError(err):
		return nil, nil //nolint:nilnil // no connection is a valid answer, not an error
	default:
		return nil, err
	}
}

func (s *Service) sealCredential(
	change *appChange,
) (*accountingsync.AccountingAppCredential, error) {
	req := change.req
	cred := change.existing
	if cred == nil {
		cred = &accountingsync.AccountingAppCredential{
			ID:              pulid.MustNew("acctapp_"),
			OrganizationID:  req.TenantInfo.OrgID,
			BusinessUnitID:  req.TenantInfo.BuID,
			IntegrationType: req.IntegrationType,
		}
	}
	previousFingerprint := cred.Fingerprint
	cred.Environment = req.Environment
	cred.ClientID = req.ClientID
	cred.Stamp()
	cred.UpdatedByID = req.UserID
	appChanged := previousFingerprint != cred.Fingerprint

	switch secret := strings.TrimSpace(req.ClientSecret); {
	case secret != "":
		sealed, err := s.encryption.EncryptStringWithAAD(
			secret,
			s.appAAD(cred, clientSecretField),
		)
		if err != nil {
			return nil, err
		}
		cred.ClientSecretCiphertext = sealed
	case change.existing == nil || appChanged:
		return nil, errortypes.NewValidationError(
			"clientSecret",
			errortypes.ErrRequired,
			"Enter the client secret that goes with this client ID",
		)
	}

	switch verifier := strings.TrimSpace(req.WebhookVerifierToken); {
	case verifier != "":
		sealed, err := s.encryption.EncryptStringWithAAD(
			verifier,
			s.appAAD(cred, webhookVerifierField),
		)
		if err != nil {
			return nil, err
		}
		cred.WebhookVerifierCiphertext = sealed
	case req.ClearWebhookVerifierToken || appChanged:
		cred.WebhookVerifierCiphertext = ""
	}

	multiErr := errortypes.NewMultiError()
	cred.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return cred, nil
}

func (s *Service) verifyCredential(
	ctx context.Context,
	provider services.AccountingProvider,
	cred *accountingsync.AccountingAppCredential,
) error {
	name := accountingsync.ProviderName(provider.IntegrationType())
	if provider.RedirectURL() == "" {
		return errNoRedirect(name)
	}
	app, err := s.openCredential(cred)
	if err != nil {
		return err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return err
	}
	switch err = connector.VerifyApp(ctx); {
	case err == nil:
		return nil
	case errors.Is(err, services.ErrAccountingAppRejected):
		return errortypes.NewValidationError(
			"clientSecret",
			errortypes.ErrInvalid,
			"{0} did not accept this client ID and secret. Check that both come from the same app and the environment you chose.",
			name,
		)
	default:
		return errortypes.NewBusinessError(
			"Could not reach {0} to check the app's keys. Try again.",
			name,
		).WithInternal(err)
	}
}

func (s *Service) prepareAppChange(
	ctx context.Context,
	req *services.SaveAccountingAppRequest,
) (*appChange, error) {
	change := &appChange{req: req, provider: accountingsync.ProviderName(req.IntegrationType)}
	existing, found, err := s.tenantCredential(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if found {
		change.existing = existing
	}
	if change.conn, err = s.activeConnection(ctx, req.TenantInfo, req.IntegrationType); err != nil {
		return nil, err
	}
	if err = change.guardActiveConnection(); err != nil {
		return nil, err
	}
	return change, nil
}

func (s *Service) SaveApp(
	ctx context.Context,
	req *services.SaveAccountingAppRequest,
) (*services.AccountingSyncStatus, error) {
	req.ClientID = strings.TrimSpace(req.ClientID)
	if err := validateSaveApp(req); err != nil {
		return nil, err
	}
	provider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingsync.ProviderName(req.IntegrationType)

	checked, err := s.prepareAppChange(ctx, req)
	if err != nil {
		return nil, err
	}
	cred, err := s.sealCredential(checked)
	if err != nil {
		return nil, err
	}
	if err = s.verifyCredential(ctx, provider, cred); err != nil {
		return nil, err
	}

	var saved *accountingsync.AccountingAppCredential
	var previous map[string]any
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		current, prepErr := s.prepareAppChange(txCtx, req)
		if prepErr != nil {
			return prepErr
		}
		if !sameCredentialVersion(checked.existing, current.existing) {
			return errortypes.NewBusinessError(
				"The {0} app keys were changed by someone else while you were saving. Review them and save again.",
				name,
			)
		}
		if current.existing == nil {
			saved, prepErr = s.apps.Create(txCtx, cred)
			return prepErr
		}
		previous = jsonutils.MustToJSON(current.existing)
		saved, prepErr = s.apps.Update(txCtx, cred)
		return prepErr
	})
	if err != nil {
		return nil, err
	}

	s.logAppAudit(saved, req.UserID, previous, "Saved the "+name+" app keys ("+
		string(saved.Environment)+", client "+saved.ClientID+")")
	s.publishAppInvalidation(ctx, saved.OrganizationID, saved.BusinessUnitID, saved.ID, req.UserID)
	return s.Status(ctx, req.TenantInfo, req.IntegrationType)
}

func sameCredentialVersion(before, after *accountingsync.AccountingAppCredential) bool {
	if before == nil || after == nil {
		return before == nil && after == nil
	}
	return before.ID == after.ID && before.Version == after.Version
}

func (s *Service) RemoveApp(
	ctx context.Context,
	req *services.RemoveAccountingAppRequest,
) (*services.AccountingSyncStatus, error) {
	if _, err := s.provider(req.IntegrationType); err != nil {
		return nil, err
	}
	name := accountingsync.ProviderName(req.IntegrationType)

	var removed *accountingsync.AccountingAppCredential
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, found, getErr := s.tenantCredential(txCtx, req.TenantInfo, req.IntegrationType)
		if getErr != nil || !found {
			return getErr
		}
		conn, connErr := s.activeConnection(txCtx, req.TenantInfo, req.IntegrationType)
		if connErr != nil {
			return connErr
		}
		if conn != nil && conn.IsActive() && conn.AppSource == accountingsync.AppSourceTenant {
			return errortypes.NewBusinessError(
				"Disconnect {0} company {1} before removing the app it was connected through.",
				name,
				conn.ExternalCompanyName,
			)
		}
		removed = existing
		return s.apps.Delete(txCtx, repositories.GetAccountingAppCredentialRequest{
			TenantInfo:      req.TenantInfo,
			IntegrationType: req.IntegrationType,
		})
	})
	if err != nil {
		return nil, err
	}

	if removed != nil {
		s.logAppAudit(removed, req.UserID, jsonutils.MustToJSON(removed),
			"Removed the "+name+" app keys (client "+removed.ClientID+")")
		s.publishAppInvalidation(
			ctx,
			removed.OrganizationID,
			removed.BusinessUnitID,
			removed.ID,
			req.UserID,
		)
	}
	return s.Status(ctx, req.TenantInfo, req.IntegrationType)
}

func (s *Service) logAppAudit(
	cred *accountingsync.AccountingAppCredential,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     cred.ID.String(),
		Operation:      permission.OpManage,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(cred),
		PreviousState:  previous,
		OrganizationID: cred.OrganizationID,
		BusinessUnitID: cred.BusinessUnitID,
		Critical:       true,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log accounting app audit", zap.Error(err))
	}
}

func (s *Service) publishAppInvalidation(
	ctx context.Context,
	orgID, buID, recordID, userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish accounting app invalidation", zap.Error(err))
	}
}
