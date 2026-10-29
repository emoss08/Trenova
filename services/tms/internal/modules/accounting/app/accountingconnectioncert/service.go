package accountingconnectioncert

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionauth"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Connections   accountingconnectionports.AccountingConnectionRepository
	Certificates  accountingconnectionports.AccountingPendingCertificateRepository `optional:"true"`
	Encryption    *encryptionservice.Service
	Organizations tenantports.OrganizationRepository `optional:"true"`
	Kit           *accountingconnectionkit.Kit
	Auth          *accountingconnectionauth.Service
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	connections   accountingconnectionports.AccountingConnectionRepository
	certificates  accountingconnectionports.AccountingPendingCertificateRepository
	encryption    *encryptionservice.Service
	organizations tenantports.OrganizationRepository
	kit           *accountingconnectionkit.Kit
	auth          *accountingconnectionauth.Service
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.accounting-connection"),
		db:            p.DB,
		connections:   p.Connections,
		certificates:  p.Certificates,
		encryption:    p.Encryption,
		organizations: p.Organizations,
		kit:           p.Kit,
		auth:          p.Auth,
	}
}

var _ accountingconnectionports.AccountingCertificateService = (*Service)(nil)

func (s *Service) requireCertificates(
	typ integration.Type,
) (accountingconnectorports.AccountingProvider, error) {
	provider, err := s.kit.Provider(typ)
	if err != nil {
		return nil, err
	}
	if !accountingprovider.MustProfile(typ).ClientCredentials || s.certificates == nil {
		return nil, accountingconnectionkit.ErrCertificatesUnsupported(
			accountingprovider.ProviderName(typ),
		)
	}
	return provider, nil
}

func (s *Service) certificateSubject(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	if s.organizations == nil {
		return strings.TrimSpace(certificateSubjectPrefix)
	}
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil || strings.TrimSpace(org.Name) == "" {
		if err != nil {
			s.l.Warn("could not read the organization name for a certificate", zap.Error(err))
		}
		return strings.TrimSpace(certificateSubjectPrefix)
	}
	return stringutils.TruncateRunes(
		certificateSubjectPrefix+strings.TrimSpace(org.Name),
		maxCertificateSubjectRune,
	)
}

func (s *Service) PrepareCertificate(
	ctx context.Context,
	req *accountingconnectionports.PrepareAccountingCertificateRequest,
) (*accountingconnectionports.AccountingPreparedCertificate, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.requireCertificates(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(req.IntegrationType)
	app, err := s.kit.AppForTenant(ctx, req.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	cc, err := accountingconnectionkit.ClientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}

	cert, err := cc.PrepareCertificate(s.certificateSubject(ctx, req.TenantInfo))
	if err != nil {
		return nil, errortypes.NewBusinessError("Could not generate a certificate. Try again.").
			WithInternal(err)
	}

	now := timeutils.NowUnix()
	pending := &accountingconnection.AccountingPendingCertificate{
		ID:              pulid.MustNew("acctpc_"),
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		IntegrationType: req.IntegrationType,
		CertificatePEM:  string(cert.CertificatePEM),
		ExpiresAt:       cert.ExpiresAt.Unix(),
		PreparedByID:    req.UserID,
		PreparedAt:      now,
	}
	pending.PrivateKeyCiphertext, err = s.encryption.EncryptStringWithAAD(
		string(cert.PrivateKeyPEM),
		accountingconnectionkit.PendingKeyAAD(pending),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError("Could not generate a certificate. Try again.").
			WithInternal(err)
	}
	if _, err = s.certificates.Replace(ctx, pending); err != nil {
		return nil, err
	}

	return accountingconnectionkit.PreparedCertificateOf(pending), nil
}

func (s *Service) ConnectWithCertificate(
	ctx context.Context,
	req *accountingconnectionports.ConnectAccountingWithCertificateRequest,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.requireCertificates(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(req.IntegrationType)
	app, err := s.kit.AppForTenant(ctx, req.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	cc, err := accountingconnectionkit.ClientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}

	multiErr := validateCertificateID("certificateId", req.CertificateID)
	accountID, accountErr := cc.NormalizeAccountID(req.AccountID)
	if accountErr != nil || !accountingconnectionkit.RealmIDPattern.MatchString(accountID) {
		multiErr.Add(
			"accountId",
			errortypes.ErrInvalid,
			"Account ID is not valid. Copy it from Setup > Company > Company Information in {0}.",
			name,
		)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	held, err := s.kit.HeldPendingCertificate(
		ctx,
		req.TenantInfo,
		req.IntegrationType,
		req.CertificateID,
	)
	if err != nil {
		return nil, err
	}

	grant, err := cc.TokenFromCertificate(ctx, &accountingconnectorports.AccountingCertificateGrant{
		RealmID:       accountID,
		CertificateID: held.CertificateID,
		PrivateKeyPEM: []byte(held.PrivateKeyPEM),
	})
	if err != nil {
		return nil, errCertificateRefused(name, err)
	}

	owner := &accountingconnectionauth.StateOwner{
		TenantInfo:      req.TenantInfo,
		UserID:          req.UserID,
		IntegrationType: req.IntegrationType,
	}
	authorized := &accountingconnectionauth.AuthorizedApp{
		Connector: connector,
		Identity:  app.Identity(),
		Provider:  name,
	}
	companies, err := s.auth.GrantedCompanies(ctx, authorized, grant, accountID)
	if err != nil {
		return nil, err
	}
	if len(companies) == 1 {
		conn, connectErr := s.auth.ConnectCompany(ctx, &accountingconnectionauth.CompanyConnect{
			Owner:       owner,
			App:         authorized,
			Grant:       grant,
			Company:     companies[0],
			Certificate: held,
		})
		if connectErr != nil {
			return nil, connectErr
		}
		return &accountingconnectionports.AccountingAuthorizationCompletion{Connection: conn}, nil
	}

	return s.auth.OfferCompanyChoice(ctx, &accountingconnectionauth.CompanyChoiceOffer{
		Owner: owner,
		State: &accountingconnectionports.AccountingOAuthState{
			AppSource:      authorized.Identity.Source,
			AppFingerprint: authorized.Identity.Fingerprint,
		},
		App:           authorized,
		Grant:         grant,
		Companies:     companies,
		GrantRealm:    accountID,
		CertificateID: held.CertificateID,
	})
}

func (s *Service) RenewCertificate(
	ctx context.Context,
	req *accountingconnectionports.RenewAccountingCertificateRequest,
) (*accountingconnection.AccountingConnection, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	if multiErr := validateCertificateID("certificateId", req.CertificateID); multiErr.HasErrors() {
		return nil, multiErr
	}
	current, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: req.TenantInfo,
			ID:         req.ConnectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	provider, err := s.requireCertificates(current.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(current.IntegrationType)
	if !current.UsesCertificate() ||
		current.Status == accountingconnection.ConnectionStatusDisconnected {
		return nil, errortypes.NewBusinessError(
			"Only a {0} connection made with a certificate can have its certificate renewed.",
			name,
		)
	}

	connector, err := s.kit.ConnectorFor(ctx, provider, current)
	if err != nil {
		return nil, err
	}
	cc, err := accountingconnectionkit.ClientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}
	held, err := s.kit.HeldPendingCertificate(
		ctx,
		req.TenantInfo,
		current.IntegrationType,
		req.CertificateID,
	)
	if err != nil {
		return nil, err
	}
	grant, err := cc.TokenFromCertificate(ctx, &accountingconnectorports.AccountingCertificateGrant{
		RealmID:       current.ExternalRealmID,
		CertificateID: held.CertificateID,
		PrivateKeyPEM: []byte(held.PrivateKeyPEM),
	})
	if err != nil {
		return nil, errCertificateRefused(name, err)
	}

	now := timeutils.NowUnix()
	renewed, err := s.storeRenewedCertificate(ctx, &certificateRenewal{
		req:   req,
		name:  name,
		held:  held,
		grant: grant,
		now:   now,
	})
	if err != nil {
		return nil, err
	}
	conn := renewed.conn

	s.kit.LogAudit(conn, req.UserID, renewed.previous, "Renewed the "+name+" certificate")
	s.kit.AfterHealthChange(ctx, renewed.before, conn, now)
	s.kit.PublishInvalidation(ctx, conn, req.UserID)
	return conn, nil
}

func (s *Service) storeRenewedCertificate(
	ctx context.Context,
	in *certificateRenewal,
) (*renewedCertificate, error) {
	out := &renewedCertificate{}
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionByIDRequest{
				TenantInfo: in.req.TenantInfo,
				ID:         in.req.ConnectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		if !locked.UsesCertificate() ||
			locked.Status == accountingconnection.ConnectionStatusDisconnected {
			return errortypes.NewBusinessError(
				"The {0} connection changed while the certificate was being checked. Try again.",
				in.name,
			)
		}
		out.before = locked.Status
		out.previous = jsonutils.MustToJSON(locked)
		if useErr := s.kit.UseCertificate(locked, in.held); useErr != nil {
			return useErr
		}
		sealed, sealErr := s.kit.Seal(locked, in.grant, in.now)
		if sealErr != nil {
			return sealErr
		}
		locked.ApplyGrant(sealed, in.now)
		locked.RecordSuccess(in.now)
		if _, updateErr := s.connections.Update(txCtx, locked); updateErr != nil {
			return updateErr
		}
		if storeErr := s.connections.StoreTokens(
			txCtx,
			accountingconnectionkit.TokensOf(locked, in.now),
		); storeErr != nil {
			return storeErr
		}
		out.conn = locked
		return s.certificates.Delete(
			txCtx,
			accountingconnectionports.GetAccountingConnectionRequest{
				TenantInfo:      in.req.TenantInfo,
				IntegrationType: locked.IntegrationType,
			},
		)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
