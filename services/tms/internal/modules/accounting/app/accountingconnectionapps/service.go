package accountingconnectionapps

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionstatus"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	DB          kernelports.Transactor
	Apps        accountingconnectionports.AccountingAppCredentialRepository
	Connections accountingconnectionports.AccountingConnectionRepository
	Encryption  *encryptionservice.Service
	Audit       auditports.AuditService
	Realtime    realtimeports.RealtimeService `optional:"true"`
	Kit         *accountingconnectionkit.Kit
	Status      *accountingconnectionstatus.Service
}

type Service struct {
	l           *zap.Logger
	db          kernelports.Transactor
	apps        accountingconnectionports.AccountingAppCredentialRepository
	connections accountingconnectionports.AccountingConnectionRepository
	encryption  *encryptionservice.Service
	audit       auditports.AuditService
	realtime    realtimeports.RealtimeService
	kit         *accountingconnectionkit.Kit
	status      *accountingconnectionstatus.Service
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.accounting-connection"),
		db:          p.DB,
		apps:        p.Apps,
		connections: p.Connections,
		encryption:  p.Encryption,
		audit:       p.Audit,
		realtime:    p.Realtime,
		kit:         p.Kit,
		status:      p.Status,
	}
}

var _ accountingconnectionports.AccountingAppService = (*Service)(nil)

func (s *Service) activeConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.connections.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
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
) (*accountingconnection.AccountingAppCredential, error) {
	req := change.req
	cred := change.existing
	if cred == nil {
		cred = &accountingconnection.AccountingAppCredential{
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
			accountingconnectionkit.AppAAD(cred, accountingconnectionkit.ClientSecretField),
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
			accountingconnectionkit.AppAAD(cred, accountingconnectionkit.WebhookVerifierField),
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
	provider accountingconnectorports.AccountingProvider,
	cred *accountingconnection.AccountingAppCredential,
) error {
	name := accountingprovider.ProviderName(provider.IntegrationType())
	if provider.RedirectURL() == "" {
		return accountingconnectionkit.ErrNoRedirect(name)
	}
	app, err := s.kit.OpenCredential(cred)
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
	case errors.Is(err, accountingconnectorports.ErrAccountingAppRejected):
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
	req *accountingconnectionports.SaveAccountingAppRequest,
) (*appChange, error) {
	change := &appChange{req: req, provider: accountingprovider.ProviderName(req.IntegrationType)}
	existing, found, err := s.kit.TenantCredential(ctx, req.TenantInfo, req.IntegrationType)
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
	req *accountingconnectionports.SaveAccountingAppRequest,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	req.ClientID = strings.TrimSpace(req.ClientID)
	if err := validateSaveApp(req); err != nil {
		return nil, err
	}
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(req.IntegrationType)

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

	var saved *accountingconnection.AccountingAppCredential
	var previous map[string]any
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
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
	return s.status.Status(ctx, req.TenantInfo, req.IntegrationType)
}

func (s *Service) RemoveApp(
	ctx context.Context,
	req *accountingconnectionports.RemoveAccountingAppRequest,
) (*accountingconnectionports.AccountingSyncStatus, error) {
	if _, err := s.kit.Provider(req.IntegrationType); err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(req.IntegrationType)

	var removed *accountingconnection.AccountingAppCredential
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, found, getErr := s.kit.TenantCredential(
			txCtx,
			req.TenantInfo,
			req.IntegrationType,
		)
		if getErr != nil || !found {
			return getErr
		}
		conn, connErr := s.activeConnection(txCtx, req.TenantInfo, req.IntegrationType)
		if connErr != nil {
			return connErr
		}
		if conn != nil && conn.IsActive() &&
			conn.AppSource == accountingconnection.AppSourceTenant {
			return errortypes.NewBusinessError(
				"Disconnect {0} company {1} before removing the app it was connected through.",
				name,
				conn.ExternalCompanyName,
			)
		}
		removed = existing
		return s.apps.Delete(txCtx, accountingconnectionports.GetAccountingAppCredentialRequest{
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
	return s.status.Status(ctx, req.TenantInfo, req.IntegrationType)
}

func (s *Service) logAppAudit(
	cred *accountingconnection.AccountingAppCredential,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &auditports.LogActionParams{
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
