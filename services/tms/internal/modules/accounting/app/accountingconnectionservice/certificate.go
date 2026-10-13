package accountingconnectionservice

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
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
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	certificateSubjectPrefix  = "Trenova "
	maxCertificateSubjectRune = 64
)

var certificateIDPattern = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]{1,100}$`)

type heldCertificate struct {
	certificateID string
	privateKeyPEM string
	expiresAt     int64
}

func errCertificatesUnsupported(provider string) error {
	return errortypes.NewBusinessError(
		"{0} connections cannot be made with a certificate. Sign in instead.",
		provider,
	)
}

func errNoPendingCertificate(provider string) error {
	return errortypes.NewBusinessError(
		"There is no {0} certificate waiting to be connected. Generate a certificate, upload it to {0}, then try again.",
		provider,
	)
}

func (s *Service) pendingKeyAAD(
	pending *accountingsync.AccountingPendingCertificate,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingClientKey,
		OrganizationID: pending.OrganizationID,
		BusinessUnitID: pending.BusinessUnitID,
		ResourceID:     pending.ID.String(),
	}
}

func (s *Service) requireCertificates(
	typ integration.Type,
) (accountingsyncports.AccountingProvider, error) {
	provider, err := s.provider(typ)
	if err != nil {
		return nil, err
	}
	if !accountingsync.MustProfile(typ).ClientCredentials || s.certificates == nil {
		return nil, errCertificatesUnsupported(accountingsync.ProviderName(typ))
	}
	return provider, nil
}

func clientCredentialsOf(
	connector accountingsyncports.AccountingConnector,
	provider string,
) (accountingsyncports.AccountingClientCredentialsConnector, error) {
	cc, ok := connector.(accountingsyncports.AccountingClientCredentialsConnector)
	if !ok {
		return nil, errCertificatesUnsupported(provider)
	}
	return cc, nil
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
	req *accountingsyncports.PrepareAccountingCertificateRequest,
) (*accountingsyncports.AccountingPreparedCertificate, error) {
	if err := s.requireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.requireCertificates(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingsync.ProviderName(req.IntegrationType)
	app, err := s.appForTenant(ctx, req.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	cc, err := clientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}

	cert, err := cc.PrepareCertificate(s.certificateSubject(ctx, req.TenantInfo))
	if err != nil {
		return nil, errortypes.NewBusinessError("Could not generate a certificate. Try again.").
			WithInternal(err)
	}

	now := timeutils.NowUnix()
	pending := &accountingsync.AccountingPendingCertificate{
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
		s.pendingKeyAAD(pending),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError("Could not generate a certificate. Try again.").
			WithInternal(err)
	}
	if _, err = s.certificates.Replace(ctx, pending); err != nil {
		return nil, err
	}

	return preparedCertificateOf(pending), nil
}

func validateCertificateID(field, certificateID string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(certificateID) == "" {
		multiErr.Add(field, errortypes.ErrRequired, "Certificate ID is required")
	} else if !certificateIDPattern.MatchString(certificateID) {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Certificate ID is not valid. Copy it from the certificate's row in the mapping.",
		)
	}
	return multiErr
}

func (s *Service) heldPendingCertificate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
	certificateID string,
) (*heldCertificate, error) {
	name := accountingsync.ProviderName(typ)
	pending, err := s.certificates.GetByType(
		ctx,
		accountingsyncports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errNoPendingCertificate(name)
		}
		return nil, err
	}
	if pending.ExpiresAt <= timeutils.NowUnix() {
		return nil, errortypes.NewBusinessError(
			"The {0} certificate waiting to be connected has expired. Generate a new one.",
			name,
		)
	}
	key, err := s.encryption.DecryptStringWithAAD(
		pending.PrivateKeyCiphertext,
		s.pendingKeyAAD(pending),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"The {0} certificate's key could not be read. Generate a new certificate.",
			name,
		).WithInternal(err)
	}
	return &heldCertificate{
		certificateID: certificateID,
		privateKeyPEM: key,
		expiresAt:     pending.ExpiresAt,
	}, nil
}

func errCertificateRefused(provider string, err error) error {
	return errortypes.NewBusinessError(
		"{0} did not accept the certificate. Check that it is uploaded to the mapping for Trenova's application, that the mapping's role can use REST web services, and that the certificate ID and account ID are right.",
		provider,
	).WithInternal(err)
}

func (s *Service) ConnectWithCertificate(
	ctx context.Context,
	req *accountingsyncports.ConnectAccountingWithCertificateRequest,
) (*accountingsyncports.AccountingAuthorizationCompletion, error) {
	if err := s.requireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.requireCertificates(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingsync.ProviderName(req.IntegrationType)
	app, err := s.appForTenant(ctx, req.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	cc, err := clientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}

	multiErr := validateCertificateID("certificateId", req.CertificateID)
	accountID, accountErr := cc.NormalizeAccountID(req.AccountID)
	if accountErr != nil || !realmIDPattern.MatchString(accountID) {
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

	held, err := s.heldPendingCertificate(
		ctx,
		req.TenantInfo,
		req.IntegrationType,
		req.CertificateID,
	)
	if err != nil {
		return nil, err
	}

	grant, err := cc.TokenFromCertificate(ctx, &accountingsyncports.AccountingCertificateGrant{
		RealmID:       accountID,
		CertificateID: held.certificateID,
		PrivateKeyPEM: []byte(held.privateKeyPEM),
	})
	if err != nil {
		return nil, errCertificateRefused(name, err)
	}

	owner := &stateOwner{
		tenantInfo:      req.TenantInfo,
		userID:          req.UserID,
		integrationType: req.IntegrationType,
	}
	authorized := &authorizedApp{connector: connector, identity: app.Identity(), provider: name}
	companies, err := s.grantedCompanies(ctx, authorized, grant, accountID)
	if err != nil {
		return nil, err
	}
	if len(companies) == 1 {
		conn, connectErr := s.connectCompany(ctx, &companyConnect{
			owner:       owner,
			app:         authorized,
			grant:       grant,
			company:     companies[0],
			certificate: held,
		})
		if connectErr != nil {
			return nil, connectErr
		}
		return &accountingsyncports.AccountingAuthorizationCompletion{Connection: conn}, nil
	}

	return s.offerCompanyChoice(ctx, &companyChoiceOffer{
		owner: owner,
		state: &accountingsyncports.AccountingOAuthState{
			AppSource:      authorized.identity.Source,
			AppFingerprint: authorized.identity.Fingerprint,
		},
		app:           authorized,
		grant:         grant,
		companies:     companies,
		grantRealm:    accountID,
		certificateID: held.certificateID,
	})
}

func (s *Service) RenewCertificate(
	ctx context.Context,
	req *accountingsyncports.RenewAccountingCertificateRequest,
) (*accountingsync.AccountingConnection, error) {
	if err := s.requireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	if multiErr := validateCertificateID("certificateId", req.CertificateID); multiErr.HasErrors() {
		return nil, multiErr
	}
	current, err := s.connections.GetByID(
		ctx,
		accountingsyncports.GetAccountingConnectionByIDRequest{
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
	name := accountingsync.ProviderName(current.IntegrationType)
	if !current.UsesCertificate() || current.Status == accountingsync.ConnectionStatusDisconnected {
		return nil, errortypes.NewBusinessError(
			"Only a {0} connection made with a certificate can have its certificate renewed.",
			name,
		)
	}

	connector, err := s.connectorFor(ctx, provider, current)
	if err != nil {
		return nil, err
	}
	cc, err := clientCredentialsOf(connector, name)
	if err != nil {
		return nil, err
	}
	held, err := s.heldPendingCertificate(
		ctx,
		req.TenantInfo,
		current.IntegrationType,
		req.CertificateID,
	)
	if err != nil {
		return nil, err
	}
	grant, err := cc.TokenFromCertificate(ctx, &accountingsyncports.AccountingCertificateGrant{
		RealmID:       current.ExternalRealmID,
		CertificateID: held.certificateID,
		PrivateKeyPEM: []byte(held.privateKeyPEM),
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

	s.logAudit(conn, req.UserID, renewed.previous, "Renewed the "+name+" certificate")
	s.afterHealthChange(ctx, renewed.before, conn, now)
	s.publishInvalidation(ctx, conn, req.UserID)
	return conn, nil
}

func (s *Service) useCertificate(
	conn *accountingsync.AccountingConnection,
	held *heldCertificate,
) error {
	sealed, err := s.encryption.EncryptStringWithAAD(
		held.privateKeyPEM,
		s.aad(conn, clientKeyField),
	)
	if err != nil {
		return err
	}
	conn.UseCertificate(accountingsync.ClientCertificate{
		KeyCiphertext: sealed,
		CertificateID: held.certificateID,
		ExpiresAt:     held.expiresAt,
	})
	return nil
}

func (s *Service) tokenFromStoredCertificate(
	ctx, txCtx context.Context,
	connector accountingsyncports.AccountingConnector,
	conn *accountingsync.AccountingConnection,
	now int64,
) error {
	cc, err := clientCredentialsOf(connector, accountingsync.ProviderName(conn.IntegrationType))
	if err != nil {
		return err
	}
	key, err := s.open(conn, clientKeyField, conn.ClientKeyCiphertext)
	if err != nil {
		return err
	}
	grant, err := cc.TokenFromCertificate(ctx, &accountingsyncports.AccountingCertificateGrant{
		RealmID:       conn.ExternalRealmID,
		CertificateID: conn.ClientCertificateID,
		PrivateKeyPEM: []byte(key),
	})
	if err != nil {
		return err
	}
	sealed, err := s.seal(conn, grant, now)
	if err != nil {
		return err
	}
	conn.ApplyGrant(sealed, now)
	return s.connections.StoreTokens(txCtx, tokensOf(conn, now))
}

func (s *Service) applyConnectMethod(
	conn *accountingsync.AccountingConnection,
	certificate *heldCertificate,
) error {
	if certificate == nil {
		conn.UseAuthorization()
		return nil
	}
	return s.useCertificate(conn, certificate)
}

func (s *Service) consumePendingCertificate(
	txCtx context.Context,
	conn *accountingsync.AccountingConnection,
	certificate *heldCertificate,
) error {
	if certificate == nil || s.certificates == nil {
		return nil
	}
	return s.certificates.Delete(txCtx, accountingsyncports.GetAccountingConnectionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: conn.OrganizationID,
			BuID:  conn.BusinessUnitID,
		},
		IntegrationType: conn.IntegrationType,
	})
}

func (s *Service) pendingCertificate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*accountingsyncports.AccountingPreparedCertificate, error) {
	if s.certificates == nil || !accountingsync.MustProfile(typ).ClientCredentials {
		return nil, nil //nolint:nilnil // no certificate is waiting
	}
	pending, err := s.certificates.GetByType(
		ctx,
		accountingsyncports.GetAccountingConnectionRequest{
			TenantInfo:      tenantInfo,
			IntegrationType: typ,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // no certificate is waiting
		}
		return nil, err
	}
	return preparedCertificateOf(pending), nil
}

func preparedCertificateOf(
	pending *accountingsync.AccountingPendingCertificate,
) *accountingsyncports.AccountingPreparedCertificate {
	return &accountingsyncports.AccountingPreparedCertificate{
		IntegrationType: pending.IntegrationType,
		CertificatePEM:  pending.CertificatePEM,
		ExpiresAt:       pending.ExpiresAt,
		PreparedAt:      pending.PreparedAt,
	}
}

type certificateRenewal struct {
	req   *accountingsyncports.RenewAccountingCertificateRequest
	name  string
	held  *heldCertificate
	grant *accountingsyncports.AccountingTokenGrant
	now   int64
}

type renewedCertificate struct {
	conn     *accountingsync.AccountingConnection
	previous map[string]any
	before   accountingsync.ConnectionStatus
}

func (s *Service) storeRenewedCertificate(
	ctx context.Context,
	in *certificateRenewal,
) (*renewedCertificate, error) {
	out := &renewedCertificate{}
	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			accountingsyncports.GetAccountingConnectionByIDRequest{
				TenantInfo: in.req.TenantInfo,
				ID:         in.req.ConnectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		if !locked.UsesCertificate() ||
			locked.Status == accountingsync.ConnectionStatusDisconnected {
			return errortypes.NewBusinessError(
				"The {0} connection changed while the certificate was being checked. Try again.",
				in.name,
			)
		}
		out.before = locked.Status
		out.previous = jsonutils.MustToJSON(locked)
		if useErr := s.useCertificate(locked, in.held); useErr != nil {
			return useErr
		}
		sealed, sealErr := s.seal(locked, in.grant, in.now)
		if sealErr != nil {
			return sealErr
		}
		locked.ApplyGrant(sealed, in.now)
		locked.RecordSuccess(in.now)
		if _, updateErr := s.connections.Update(txCtx, locked); updateErr != nil {
			return updateErr
		}
		if storeErr := s.connections.StoreTokens(txCtx, tokensOf(locked, in.now)); storeErr != nil {
			return storeErr
		}
		out.conn = locked
		return s.certificates.Delete(txCtx, accountingsyncports.GetAccountingConnectionRequest{
			TenantInfo:      in.req.TenantInfo,
			IntegrationType: locked.IntegrationType,
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
