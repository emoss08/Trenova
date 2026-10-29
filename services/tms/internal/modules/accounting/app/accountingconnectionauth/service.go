package accountingconnectionauth

import (
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionsubscriptions"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func (s *Service) CompleteAuthorization(
	ctx context.Context,
	req *accountingconnectionports.CompleteAccountingAuthorizationRequest,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	accountingProvider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	profile := accountingprovider.MustProfile(req.IntegrationType)
	if err = validateCompletion(req, &profile); err != nil {
		return nil, err
	}
	owner := &StateOwner{
		TenantInfo:      req.TenantInfo,
		UserID:          req.UserID,
		IntegrationType: req.IntegrationType,
	}

	state, err := s.takeState(ctx, req.State, owner)
	if err != nil {
		return nil, err
	}
	if state.SealedGrant != "" {
		return nil, errStateExpired()
	}
	app, err := s.authorizedApp(ctx, accountingProvider, owner, state)
	if err != nil {
		return nil, err
	}
	verifier, err := s.openCodeVerifier(req.TenantInfo, tokenutils.Hash(req.State), state)
	if err != nil {
		return nil, errStateExpired()
	}

	grant, err := app.Connector.ExchangeCode(ctx, &accountingconnectorports.AccountingCodeExchange{
		Code:            req.Code,
		CallbackRealmID: req.RealmID,
		CodeVerifier:    verifier,
	})
	if err != nil {
		return nil, errortypes.NewBusinessError("{0} did not accept the authorization. Try connecting again.", app.Provider).
			WithInternal(err)
	}

	companies, err := s.GrantedCompanies(ctx, app, grant, req.RealmID)
	if err != nil {
		return nil, err
	}
	if len(companies) == 1 {
		conn, connectErr := s.ConnectCompany(ctx, &CompanyConnect{
			Owner:   owner,
			App:     app,
			Grant:   grant,
			Company: companies[0],
		})
		if connectErr != nil {
			return nil, connectErr
		}
		return &accountingconnectionports.AccountingAuthorizationCompletion{Connection: conn}, nil
	}

	return s.OfferCompanyChoice(ctx, &CompanyChoiceOffer{
		Owner:      owner,
		State:      state,
		App:        app,
		Grant:      grant,
		Companies:  companies,
		GrantRealm: grantRealmOf(req.RealmID, companies),
	})
}

func (s *Service) GrantedCompanies(
	ctx context.Context,
	app *AuthorizedApp,
	grant *accountingconnectorports.AccountingTokenGrant,
	callbackRealmID string,
) ([]accountingconnectorports.AccountingCompany, error) {
	granted, err := app.Connector.Companies(ctx, grant, callbackRealmID)
	if err != nil {
		s.kit.RevokeQuietly(ctx, app.Connector, grant.RefreshToken, callbackRealmID)
		return nil, errortypes.NewBusinessError("Connected, but {0} would not say which company was authorized. Try connecting again.", app.Provider).
			WithInternal(err)
	}
	companies := make([]accountingconnectorports.AccountingCompany, 0, len(granted))
	for _, company := range granted {
		if accountingconnectionkit.RealmIDPattern.MatchString(company.ID) {
			companies = append(companies, company)
		}
	}
	if len(companies) == 0 {
		s.kit.RevokeQuietly(
			ctx,
			app.Connector,
			grant.RefreshToken,
			grantRealmOf(callbackRealmID, granted),
		)
		return nil, errortypes.NewBusinessError(
			"{0} did not authorize any company. Connect again and choose the company to sync with.",
			app.Provider,
		)
	}
	return companies, nil
}

func (s *Service) OfferCompanyChoice(
	ctx context.Context,
	offer *CompanyChoiceOffer,
) (*accountingconnectionports.AccountingAuthorizationCompletion, error) {
	token, tokenHash, err := tokenutils.New()
	if err != nil {
		s.kit.RevokeQuietly(ctx, offer.App.Connector, offer.Grant.RefreshToken, offer.GrantRealm)
		return nil, err
	}
	sealed, err := s.sealPendingGrant(offer.Owner.TenantInfo, tokenHash, &pendingGrant{
		AccessToken:     offer.Grant.AccessToken,
		RefreshToken:    offer.Grant.RefreshToken,
		AccessTokenTTL:  offer.Grant.AccessTokenTTL,
		RefreshTokenTTL: offer.Grant.RefreshTokenTTL,
		CertificateID:   offer.CertificateID,
	})
	if err != nil {
		s.kit.RevokeQuietly(ctx, offer.App.Connector, offer.Grant.RefreshToken, offer.GrantRealm)
		return nil, err
	}

	choices := make([]accountingconnectionports.AccountingOAuthCompany, 0, len(offer.Companies))
	for _, company := range offer.Companies {
		choices = append(choices, accountingconnectionports.AccountingOAuthCompany(company))
	}

	now := timeutils.NowUnix()
	if err = s.states.Save(ctx, &accountingconnectionports.AccountingOAuthState{
		State:           tokenHash,
		IntegrationType: offer.Owner.IntegrationType,
		UserID:          offer.Owner.UserID,
		OrganizationID:  offer.Owner.TenantInfo.OrgID,
		BusinessUnitID:  offer.Owner.TenantInfo.BuID,
		CreatedAt:       now,
		AppSource:       offer.State.AppSource,
		AppFingerprint:  offer.State.AppFingerprint,
		SealedGrant:     sealed,
		Companies:       choices,
		GrantRealmID:    offer.GrantRealm,
	}, CompanyChoiceTTL); err != nil {
		s.kit.RevokeQuietly(ctx, offer.App.Connector, offer.Grant.RefreshToken, offer.GrantRealm)
		return nil, errortypes.NewBusinessError("Could not hold the authorization while you choose a company. Try connecting again.").
			WithInternal(err)
	}

	return &accountingconnectionports.AccountingAuthorizationCompletion{
		Companies:       offer.Companies,
		ChoiceToken:     token,
		ChoiceExpiresAt: now + int64(CompanyChoiceTTL/time.Second),
	}, nil
}

func (s *Service) ChooseCompany(
	ctx context.Context,
	req *accountingconnectionports.ChooseAccountingCompanyRequest,
) (*accountingconnection.AccountingConnection, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	accountingProvider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if err = validateChoice(req); err != nil {
		return nil, err
	}
	owner := &StateOwner{
		TenantInfo:      req.TenantInfo,
		UserID:          req.UserID,
		IntegrationType: req.IntegrationType,
	}

	tokenHash := tokenutils.Hash(req.ChoiceToken)
	state, err := s.takeState(ctx, req.ChoiceToken, owner)
	if err != nil {
		return nil, err
	}
	if state.SealedGrant == "" {
		return nil, errStateExpired()
	}
	app, err := s.authorizedApp(ctx, accountingProvider, owner, state)
	if err != nil {
		return nil, err
	}
	held, err := s.openPendingGrant(req.TenantInfo, tokenHash, state.SealedGrant)
	if err != nil {
		return nil, errortypes.NewBusinessError("The held authorization could not be read. Try connecting again.").
			WithInternal(err)
	}
	grant := held.tokenGrant()
	var certificate *accountingconnectionkit.HeldCertificate
	if held.CertificateID != "" {
		if s.certificates == nil {
			return nil, accountingconnectionkit.ErrCertificatesUnsupported(app.Provider)
		}
		if certificate, err = s.kit.HeldPendingCertificate(
			ctx,
			req.TenantInfo,
			req.IntegrationType,
			held.CertificateID,
		); err != nil {
			return nil, err
		}
	}

	chosen, others := splitChoice(state.Companies, req.CompanyID)
	if chosen == nil {
		s.releaseQuietly(ctx, app.Connector, grant, others)
		s.kit.RevokeQuietly(ctx, app.Connector, grant.RefreshToken, state.GrantRealmID)
		return nil, errortypes.NewValidationError(
			"companyId",
			errortypes.ErrInvalid,
			"That company was not among the ones {0} authorized. Connect again.",
			app.Provider,
		)
	}

	s.releaseQuietly(ctx, app.Connector, grant, others)
	return s.ConnectCompany(ctx, &CompanyConnect{
		Owner:       owner,
		App:         app,
		Grant:       grant,
		Company:     *chosen,
		Certificate: certificate,
	})
}

func (s *Service) releaseQuietly(
	ctx context.Context,
	connector accountingconnectorports.AccountingConnector,
	grant *accountingconnectorports.AccountingTokenGrant,
	companies []accountingconnectorports.AccountingCompany,
) {
	if len(companies) == 0 {
		return
	}
	if err := connector.ReleaseCompanies(ctx, grant, companies); err != nil {
		s.l.Warn("could not release the companies that were not chosen", zap.Error(err))
	}
}

func (s *Service) authorizedApp(
	ctx context.Context,
	provider accountingconnectorports.AccountingProvider,
	owner *StateOwner,
	state *accountingconnectionports.AccountingOAuthState,
) (*AuthorizedApp, error) {
	name := accountingprovider.ProviderName(owner.IntegrationType)
	app, err := s.kit.AppForTenant(ctx, owner.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	identity := app.Identity()
	if state.AppSource != identity.Source || state.AppFingerprint != identity.Fingerprint {
		return nil, errortypes.NewValidationError(
			"state",
			errortypes.ErrInvalid,
			"The {0} app keys changed while you were signing in. Start the connection again.",
			name,
		)
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	return &AuthorizedApp{Connector: connector, Identity: identity, Provider: name}, nil
}

func (s *Service) ConnectCompany(
	ctx context.Context,
	in *CompanyConnect,
) (*accountingconnection.AccountingConnection, error) {
	provider := in.App.Provider
	facts, err := in.App.Connector.CompanyFacts(ctx, in.Company.ID, in.Grant.AccessToken)
	if err != nil {
		s.kit.RevokeQuietly(ctx, in.App.Connector, in.Grant.RefreshToken, in.Company.ID)
		return nil, errortypes.NewBusinessError("Connected, but {0} would not describe the company. Try connecting again.", provider).
			WithInternal(err)
	}

	if err = s.ensureNoOtherSystem(ctx, in.Owner.TenantInfo, in.Owner.IntegrationType); err != nil {
		s.kit.RevokeQuietly(ctx, in.App.Connector, in.Grant.RefreshToken, in.Company.ID)
		return nil, err
	}
	if err = s.ensureRealmIsFree(ctx, in); err != nil {
		s.kit.RevokeQuietly(ctx, in.App.Connector, in.Grant.RefreshToken, in.Company.ID)
		return nil, err
	}

	now := timeutils.NowUnix()
	conn, previous, err := s.saveConnection(ctx, &connectionSave{
		owner:       in.Owner,
		realmID:     in.Company.ID,
		grant:       in.Grant,
		facts:       facts,
		app:         in.App.Identity,
		provider:    provider,
		now:         now,
		certificate: in.Certificate,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			s.kit.RevokeQuietly(ctx, in.App.Connector, in.Grant.RefreshToken, in.Company.ID)
			return nil, errRealmTaken(provider)
		}
		return nil, err
	}

	s.kit.SyncIntegrationFlag(ctx, conn)
	s.kit.LogAudit(
		conn,
		in.Owner.UserID,
		previous,
		"Connected "+provider+" company "+conn.ExternalCompanyName,
	)
	s.kit.AfterHealthChange(ctx, accountingconnection.ConnectionStatusDisconnected, conn, now)
	s.kit.PublishInvalidation(ctx, conn, in.Owner.UserID)
	s.kit.RequestReferenceRefresh(ctx, conn)

	return conn, nil
}

func (s *Service) takeState(
	ctx context.Context,
	token string,
	owner *StateOwner,
) (*accountingconnectionports.AccountingOAuthState, error) {
	state, err := s.states.Take(ctx, tokenutils.Hash(token))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errStateExpired()
		}
		return nil, err
	}
	if state.UserID != owner.UserID ||
		state.OrganizationID != owner.TenantInfo.OrgID ||
		state.BusinessUnitID != owner.TenantInfo.BuID ||
		state.IntegrationType != owner.IntegrationType {
		return nil, errortypes.NewAuthorizationError(
			"This connection was started by someone else. Start the connection again from your own account.",
		)
	}

	return state, nil
}

func (s *Service) pendingGrantAAD(
	tenantInfo pagination.TenantInfo,
	stateHash string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingPendingGrant,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     stateHash,
	}
}

func (s *Service) sealPendingGrant(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	grant *pendingGrant,
) (string, error) {
	raw, err := sonic.Marshal(grant)
	if err != nil {
		return "", err
	}
	return s.encryption.EncryptStringWithAAD(string(raw), s.pendingGrantAAD(tenantInfo, stateHash))
}

func (s *Service) openPendingGrant(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	sealed string,
) (*pendingGrant, error) {
	raw, err := s.encryption.DecryptStringWithAAD(sealed, s.pendingGrantAAD(tenantInfo, stateHash))
	if err != nil {
		return nil, err
	}
	held := new(pendingGrant)
	if err = sonic.UnmarshalString(raw, held); err != nil {
		return nil, err
	}
	return held, nil
}

func (s *Service) saveConnection(
	ctx context.Context,
	in *connectionSave,
) (*accountingconnection.AccountingConnection, map[string]any, error) {
	owner, grant, facts, provider, now := in.owner, in.grant, in.facts, in.provider, in.now
	var conn *accountingconnection.AccountingConnection
	var previous map[string]any
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionRequest{
				TenantInfo:      owner.TenantInfo,
				IntegrationType: owner.IntegrationType,
			},
		)
		if lockErr != nil && !errortypes.IsNotFoundError(lockErr) {
			return lockErr
		}

		if existing == nil {
			conn = &accountingconnection.AccountingConnection{
				ID:              pulid.MustNew("acctc_"),
				OrganizationID:  owner.TenantInfo.OrgID,
				BusinessUnitID:  owner.TenantInfo.BuID,
				IntegrationType: owner.IntegrationType,
				ExternalRealmID: in.realmID,
			}
			conn.BindApp(in.app)
			if methodErr := s.kit.ApplyConnectMethod(conn, in.certificate); methodErr != nil {
				return methodErr
			}
			if sealErr := s.kit.Connect(conn, owner.UserID, grant, facts, now); sealErr != nil {
				return sealErr
			}
			if _, createErr := s.connections.Create(txCtx, conn); createErr != nil {
				return createErr
			}
			return s.kit.ConsumePendingCertificate(txCtx, conn, in.certificate)
		}

		if existing.IsActive() && existing.ExternalRealmID != in.realmID {
			return errortypes.NewBusinessError(
				"Disconnect {0} before connecting a different {1} company.",
				existing.ExternalCompanyName,
				provider,
			)
		}

		previous = jsonutils.MustToJSON(existing)
		existing.ExternalRealmID = in.realmID
		existing.BindApp(in.app)
		if methodErr := s.kit.ApplyConnectMethod(existing, in.certificate); methodErr != nil {
			return methodErr
		}
		if sealErr := s.kit.Connect(existing, owner.UserID, grant, facts, now); sealErr != nil {
			return sealErr
		}
		if _, updateErr := s.connections.Update(txCtx, existing); updateErr != nil {
			return updateErr
		}
		if storeErr := s.connections.StoreTokens(txCtx, accountingconnectionkit.TokensOf(existing, now)); storeErr != nil {
			return storeErr
		}
		conn = existing
		return s.kit.ConsumePendingCertificate(txCtx, existing, in.certificate)
	})
	return conn, previous, err
}

func (s *Service) ensureRealmIsFree(ctx context.Context, in *CompanyConnect) error {
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		accountingconnectionports.ListAccountingConnectionsByRealmRequest{
			IntegrationType: in.Owner.IntegrationType,
			RealmIDs:        []string{in.Company.ID},
		},
	)
	if err != nil {
		return err
	}
	for _, holder := range holders {
		if holder.OrganizationID != in.Owner.TenantInfo.OrgID ||
			holder.BusinessUnitID != in.Owner.TenantInfo.BuID {
			return errRealmTaken(in.App.Provider)
		}
	}
	return nil
}

func (s *Service) ensureNoOtherSystem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) error {
	conns, err := s.connections.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return err
	}
	for _, conn := range conns {
		if conn.IntegrationType == typ ||
			conn.Status == accountingconnection.ConnectionStatusDisconnected {
			continue
		}
		return errortypes.NewBusinessError(
			"Disconnect {0} before connecting {1}. Trenova syncs with one accounting system at a time.",
			accountingprovider.ProviderName(conn.IntegrationType),
			accountingprovider.ProviderName(typ),
		)
	}
	return nil
}

func (s *Service) verifierAAD(
	tenantInfo pagination.TenantInfo,
	stateHash string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingOAuthVerifier,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     stateHash,
	}
}

func (s *Service) newCodeVerifier(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	typ integration.Type,
) (challenge, sealed string, err error) {
	if !accountingprovider.MustProfile(typ).PKCE {
		return "", "", nil
	}
	verifier, err := tokenutils.PKCEVerifier()
	if err != nil {
		return "", "", err
	}
	sealed, err = s.encryption.EncryptStringWithAAD(verifier, s.verifierAAD(tenantInfo, stateHash))
	if err != nil {
		return "", "", errortypes.NewBusinessError("Could not start the connection. Try again.").
			WithInternal(err)
	}
	return tokenutils.PKCEChallenge(verifier), sealed, nil
}

func (s *Service) openCodeVerifier(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	state *accountingconnectionports.AccountingOAuthState,
) (string, error) {
	if !accountingprovider.MustProfile(state.IntegrationType).PKCE {
		return "", nil
	}
	if state.SealedVerifier == "" {
		return "", errVerifierMissing
	}
	return s.encryption.DecryptStringWithAAD(
		state.SealedVerifier,
		s.verifierAAD(tenantInfo, stateHash),
	)
}

const (
	authorizationStateTTL = 10 * time.Minute
)

func (s *Service) StartAuthorization(
	ctx context.Context,
	req *accountingconnectionports.StartAccountingAuthorizationRequest,
) (*accountingconnectionports.AccountingAuthorizationStart, error) {
	if err := s.kit.RequireIntegrations(ctx, req.TenantInfo); err != nil {
		return nil, err
	}
	provider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingprovider.ProviderName(req.IntegrationType)
	if provider.RedirectURL() == "" {
		return nil, accountingconnectionkit.ErrNoRedirect(name)
	}
	if err = s.ensureNoOtherSystem(ctx, req.TenantInfo, req.IntegrationType); err != nil {
		return nil, err
	}
	app, err := s.kit.AppForTenant(ctx, req.TenantInfo, provider)
	if err != nil {
		return nil, err
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return nil, err
	}
	identity := app.Identity()

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	challenge, sealedVerifier, err := s.newCodeVerifier(
		req.TenantInfo,
		tokenHash,
		req.IntegrationType,
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	if err = s.states.Save(ctx, &accountingconnectionports.AccountingOAuthState{
		State:           tokenHash,
		IntegrationType: req.IntegrationType,
		UserID:          req.UserID,
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		CreatedAt:       now,
		AppSource:       identity.Source,
		AppFingerprint:  identity.Fingerprint,
		SealedVerifier:  sealedVerifier,
	}, authorizationStateTTL); err != nil {
		return nil, errortypes.NewBusinessError("Could not start the connection. Try again.").
			WithInternal(err)
	}

	authorizeURL, err := connector.AuthorizeURL(accountingconnectorports.AccountingAuthorizeRequest{
		State:         token,
		CodeChallenge: challenge,
	})
	if err != nil {
		return nil, err
	}

	return &accountingconnectionports.AccountingAuthorizationStart{
		AuthorizeURL: authorizeURL,
		ExpiresAt:    now + int64(authorizationStateTTL/time.Second),
	}, nil
}

func (s *Service) Disconnect(
	ctx context.Context,
	req *accountingconnectionports.DisconnectAccountingRequest,
) (*accountingconnection.AccountingConnection, error) {
	provider, err := s.kit.Provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}

	if current, getErr := s.connections.GetByType(
		ctx,
		accountingconnectionports.GetAccountingConnectionRequest{
			TenantInfo:      req.TenantInfo,
			IntegrationType: req.IntegrationType,
		},
	); getErr == nil &&
		current.IsActive() {
		s.subs.Release(ctx, current)
	}

	now := timeutils.NowUnix()
	var conn *accountingconnection.AccountingConnection
	var previous map[string]any
	var refreshCiphertext string
	changed := false
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionRequest{
				TenantInfo:      req.TenantInfo,
				IntegrationType: req.IntegrationType,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		conn = existing
		if existing.Status == accountingconnection.ConnectionStatusDisconnected {
			return nil
		}

		previous = jsonutils.MustToJSON(existing)
		refreshCiphertext = existing.RefreshTokenCiphertext
		existing.Disconnect(req.UserID, now)
		if _, updateErr := s.connections.Update(txCtx, existing); updateErr != nil {
			return updateErr
		}
		changed = true
		return s.connections.StoreTokens(txCtx, accountingconnectionkit.TokensOf(existing, now))
	})
	if err != nil {
		return nil, err
	}
	if !changed {
		return conn, nil
	}

	if refreshCiphertext != "" {
		s.kit.RevokeStored(ctx, provider, conn, refreshCiphertext)
	}

	s.kit.SyncIntegrationFlag(ctx, conn)
	s.kit.LogAudit(
		conn,
		req.UserID,
		previous,
		"Disconnected "+accountingprovider.ProviderName(
			conn.IntegrationType,
		)+" company "+conn.ExternalCompanyName,
	)
	s.kit.ResolveWatchtower(ctx, conn)
	s.kit.PublishInvalidation(ctx, conn, req.UserID)

	return conn, nil
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Connections  accountingconnectionports.AccountingConnectionRepository
	States       accountingconnectionports.AccountingOAuthStateRepository
	Certificates accountingconnectionports.AccountingPendingCertificateRepository `optional:"true"`
	Encryption   *encryptionservice.Service
	Kit          *accountingconnectionkit.Kit
	Subs         *accountingconnectionsubscriptions.Service
}

type Service struct {
	l            *zap.Logger
	db           kernelports.Transactor
	connections  accountingconnectionports.AccountingConnectionRepository
	states       accountingconnectionports.AccountingOAuthStateRepository
	certificates accountingconnectionports.AccountingPendingCertificateRepository
	encryption   *encryptionservice.Service
	kit          *accountingconnectionkit.Kit
	subs         *accountingconnectionsubscriptions.Service
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.accounting-connection"),
		db:           p.DB,
		connections:  p.Connections,
		states:       p.States,
		certificates: p.Certificates,
		encryption:   p.Encryption,
		kit:          p.Kit,
		subs:         p.Subs,
	}
}

var _ accountingconnectionports.AccountingAuthorizationService = (*Service)(nil)
