package accountingconnectionservice

import (
	"context"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const companyChoiceTTL = 10 * time.Minute

type stateOwner struct {
	tenantInfo      pagination.TenantInfo
	userID          pulid.ID
	integrationType integration.Type
}

type pendingGrant struct {
	AccessToken     string        `json:"accessToken"`
	RefreshToken    string        `json:"refreshToken"`
	AccessTokenTTL  time.Duration `json:"accessTokenTtl"`
	RefreshTokenTTL time.Duration `json:"refreshTokenTtl"`
}

type authorizedApp struct {
	connector services.AccountingConnector
	identity  accountingsync.AppIdentity
	provider  string
}

type companyConnect struct {
	owner   stateOwner
	app     authorizedApp
	grant   *services.AccountingTokenGrant
	company services.AccountingCompany
}

func validateCompletion(
	req *services.CompleteAccountingAuthorizationRequest,
	profile *accountingsync.ProviderProfile,
) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.State) == "" {
		multiErr.Add("state", errortypes.ErrRequired, "The connection request is missing its state")
	}
	if strings.TrimSpace(req.Code) == "" {
		multiErr.Add("code", errortypes.ErrRequired, "The authorization code is missing")
	}
	if (profile.CallbackCarriesCompany || req.RealmID != "") &&
		!realmIDPattern.MatchString(req.RealmID) {
		multiErr.Add(
			"realmId",
			errortypes.ErrInvalid,
			"The company id returned by the provider is not valid",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) CompleteAuthorization(
	ctx context.Context,
	req *services.CompleteAccountingAuthorizationRequest,
) (*services.AccountingAuthorizationCompletion, error) {
	accountingProvider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	profile := accountingsync.MustProfile(req.IntegrationType)
	if err = validateCompletion(req, &profile); err != nil {
		return nil, err
	}
	owner := stateOwner{
		tenantInfo:      req.TenantInfo,
		userID:          req.UserID,
		integrationType: req.IntegrationType,
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

	grant, err := app.connector.ExchangeCode(ctx, req.Code)
	if err != nil {
		return nil, errortypes.NewBusinessError("{0} did not accept the authorization. Try connecting again.", app.provider).
			WithInternal(err)
	}

	companies, err := s.grantedCompanies(ctx, app, grant, req.RealmID)
	if err != nil {
		return nil, err
	}
	if len(companies) == 1 {
		conn, connectErr := s.connectCompany(ctx, &companyConnect{
			owner:   owner,
			app:     app,
			grant:   grant,
			company: companies[0],
		})
		if connectErr != nil {
			return nil, connectErr
		}
		return &services.AccountingAuthorizationCompletion{Connection: conn}, nil
	}

	return s.offerCompanyChoice(ctx, &companyChoiceOffer{
		owner:     owner,
		state:     state,
		app:       app,
		grant:     grant,
		companies: companies,
	})
}

func (s *Service) grantedCompanies(
	ctx context.Context,
	app authorizedApp,
	grant *services.AccountingTokenGrant,
	callbackRealmID string,
) ([]services.AccountingCompany, error) {
	granted, err := app.connector.Companies(ctx, grant, callbackRealmID)
	if err != nil {
		s.revokeQuietly(ctx, app.connector, grant.RefreshToken)
		return nil, errortypes.NewBusinessError("Connected, but {0} would not say which company was authorized. Try connecting again.", app.provider).
			WithInternal(err)
	}
	companies := make([]services.AccountingCompany, 0, len(granted))
	for _, company := range granted {
		if realmIDPattern.MatchString(company.ID) {
			companies = append(companies, company)
		}
	}
	if len(companies) == 0 {
		s.revokeQuietly(ctx, app.connector, grant.RefreshToken)
		return nil, errortypes.NewBusinessError(
			"{0} did not authorize any company. Connect again and choose the company to sync with.",
			app.provider,
		)
	}
	return companies, nil
}

type companyChoiceOffer struct {
	owner     stateOwner
	state     *repositories.AccountingOAuthState
	app       authorizedApp
	grant     *services.AccountingTokenGrant
	companies []services.AccountingCompany
}

func (s *Service) offerCompanyChoice(
	ctx context.Context,
	offer *companyChoiceOffer,
) (*services.AccountingAuthorizationCompletion, error) {
	token, tokenHash, err := tokenutils.New()
	if err != nil {
		s.revokeQuietly(ctx, offer.app.connector, offer.grant.RefreshToken)
		return nil, err
	}
	sealed, err := s.sealPendingGrant(offer.owner.tenantInfo, tokenHash, offer.grant)
	if err != nil {
		s.revokeQuietly(ctx, offer.app.connector, offer.grant.RefreshToken)
		return nil, err
	}

	choices := make([]repositories.AccountingOAuthCompany, 0, len(offer.companies))
	for _, company := range offer.companies {
		choices = append(choices, repositories.AccountingOAuthCompany{
			ID:           company.ID,
			Name:         company.Name,
			ConnectionID: company.ConnectionID,
		})
	}

	now := timeutils.NowUnix()
	if err = s.states.Save(ctx, &repositories.AccountingOAuthState{
		State:           tokenHash,
		IntegrationType: offer.owner.integrationType,
		UserID:          offer.owner.userID,
		OrganizationID:  offer.owner.tenantInfo.OrgID,
		BusinessUnitID:  offer.owner.tenantInfo.BuID,
		CreatedAt:       now,
		AppSource:       offer.state.AppSource,
		AppFingerprint:  offer.state.AppFingerprint,
		SealedGrant:     sealed,
		Companies:       choices,
	}, companyChoiceTTL); err != nil {
		s.revokeQuietly(ctx, offer.app.connector, offer.grant.RefreshToken)
		return nil, errortypes.NewBusinessError("Could not hold the authorization while you choose a company. Try connecting again.").
			WithInternal(err)
	}

	return &services.AccountingAuthorizationCompletion{
		Companies:       offer.companies,
		ChoiceToken:     token,
		ChoiceExpiresAt: now + int64(companyChoiceTTL/time.Second),
	}, nil
}

func validateChoice(req *services.ChooseAccountingCompanyRequest) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.ChoiceToken) == "" {
		multiErr.Add(
			"choiceToken",
			errortypes.ErrRequired,
			"The company choice is missing its token",
		)
	}
	if !realmIDPattern.MatchString(req.CompanyID) {
		multiErr.Add("companyId", errortypes.ErrInvalid, "Choose one of the offered companies")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) ChooseCompany(
	ctx context.Context,
	req *services.ChooseAccountingCompanyRequest,
) (*accountingsync.AccountingConnection, error) {
	accountingProvider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if err = validateChoice(req); err != nil {
		return nil, err
	}
	owner := stateOwner{
		tenantInfo:      req.TenantInfo,
		userID:          req.UserID,
		integrationType: req.IntegrationType,
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
	grant, err := s.openPendingGrant(req.TenantInfo, tokenHash, state.SealedGrant)
	if err != nil {
		return nil, errortypes.NewBusinessError("The held authorization could not be read. Try connecting again.").
			WithInternal(err)
	}

	chosen, others := splitChoice(state.Companies, req.CompanyID)
	if chosen == nil {
		s.releaseQuietly(ctx, app.connector, grant, others)
		s.revokeQuietly(ctx, app.connector, grant.RefreshToken)
		return nil, errortypes.NewValidationError(
			"companyId",
			errortypes.ErrInvalid,
			"That company was not among the ones {0} authorized. Connect again.",
			app.provider,
		)
	}

	s.releaseQuietly(ctx, app.connector, grant, others)
	return s.connectCompany(ctx, &companyConnect{
		owner:   owner,
		app:     app,
		grant:   grant,
		company: *chosen,
	})
}

func splitChoice(
	choices []repositories.AccountingOAuthCompany,
	companyID string,
) (*services.AccountingCompany, []services.AccountingCompany) {
	var chosen *services.AccountingCompany
	others := make([]services.AccountingCompany, 0, len(choices))
	for _, choice := range choices {
		company := services.AccountingCompany{
			ID:           choice.ID,
			Name:         choice.Name,
			ConnectionID: choice.ConnectionID,
		}
		if chosen == nil && choice.ID == companyID {
			chosen = &company
			continue
		}
		others = append(others, company)
	}
	return chosen, others
}

func (s *Service) releaseQuietly(
	ctx context.Context,
	connector services.AccountingConnector,
	grant *services.AccountingTokenGrant,
	companies []services.AccountingCompany,
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
	provider services.AccountingProvider,
	owner stateOwner,
	state *repositories.AccountingOAuthState,
) (authorizedApp, error) {
	name := accountingsync.ProviderName(owner.integrationType)
	app, err := s.appForTenant(ctx, owner.tenantInfo, provider)
	if err != nil {
		return authorizedApp{}, err
	}
	identity := app.Identity()
	if state.AppSource != identity.Source || state.AppFingerprint != identity.Fingerprint {
		return authorizedApp{}, errortypes.NewValidationError(
			"state",
			errortypes.ErrInvalid,
			"The {0} app keys changed while you were signing in. Start the connection again.",
			name,
		)
	}
	connector, err := provider.Bind(app)
	if err != nil {
		return authorizedApp{}, err
	}
	return authorizedApp{connector: connector, identity: identity, provider: name}, nil
}

func (s *Service) connectCompany(
	ctx context.Context,
	in *companyConnect,
) (*accountingsync.AccountingConnection, error) {
	provider := in.app.provider
	facts, err := in.app.connector.CompanyFacts(ctx, in.company.ID, in.grant.AccessToken)
	if err != nil {
		s.revokeQuietly(ctx, in.app.connector, in.grant.RefreshToken)
		return nil, errortypes.NewBusinessError("Connected, but {0} would not describe the company. Try connecting again.", provider).
			WithInternal(err)
	}

	if err = s.ensureNoOtherSystem(ctx, in.owner.tenantInfo, in.owner.integrationType); err != nil {
		s.revokeQuietly(ctx, in.app.connector, in.grant.RefreshToken)
		return nil, err
	}
	if err = s.ensureRealmIsFree(ctx, in); err != nil {
		s.revokeQuietly(ctx, in.app.connector, in.grant.RefreshToken)
		return nil, err
	}

	now := timeutils.NowUnix()
	conn, previous, err := s.saveConnection(ctx, &connectionSave{
		owner:    in.owner,
		realmID:  in.company.ID,
		grant:    in.grant,
		facts:    facts,
		app:      in.app.identity,
		provider: provider,
		now:      now,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			s.revokeQuietly(ctx, in.app.connector, in.grant.RefreshToken)
			return nil, errRealmTaken(provider)
		}
		return nil, err
	}

	s.syncIntegrationFlag(ctx, conn)
	s.logAudit(
		conn,
		in.owner.userID,
		previous,
		"Connected "+provider+" company "+conn.ExternalCompanyName,
	)
	s.afterHealthChange(ctx, accountingsync.ConnectionStatusDisconnected, conn, now)
	s.publishInvalidation(ctx, conn, in.owner.userID)
	s.requestReferenceRefresh(ctx, conn)

	return conn, nil
}

func errStateExpired() error {
	return errortypes.NewValidationError(
		"state",
		errortypes.ErrInvalid,
		"This connection request expired or was already used. Start the connection again.",
	)
}

func (s *Service) takeState(
	ctx context.Context,
	token string,
	owner stateOwner,
) (*repositories.AccountingOAuthState, error) {
	state, err := s.states.Take(ctx, tokenutils.Hash(token))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errStateExpired()
		}
		return nil, err
	}
	if state.UserID != owner.userID ||
		state.OrganizationID != owner.tenantInfo.OrgID ||
		state.BusinessUnitID != owner.tenantInfo.BuID ||
		state.IntegrationType != owner.integrationType {
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
	grant *services.AccountingTokenGrant,
) (string, error) {
	raw, err := sonic.Marshal(pendingGrant{
		AccessToken:     grant.AccessToken,
		RefreshToken:    grant.RefreshToken,
		AccessTokenTTL:  grant.AccessTokenTTL,
		RefreshTokenTTL: grant.RefreshTokenTTL,
	})
	if err != nil {
		return "", err
	}
	return s.encryption.EncryptStringWithAAD(string(raw), s.pendingGrantAAD(tenantInfo, stateHash))
}

func (s *Service) openPendingGrant(
	tenantInfo pagination.TenantInfo,
	stateHash string,
	sealed string,
) (*services.AccountingTokenGrant, error) {
	raw, err := s.encryption.DecryptStringWithAAD(sealed, s.pendingGrantAAD(tenantInfo, stateHash))
	if err != nil {
		return nil, err
	}
	var held pendingGrant
	if err = sonic.UnmarshalString(raw, &held); err != nil {
		return nil, err
	}
	return &services.AccountingTokenGrant{
		AccessToken:     held.AccessToken,
		RefreshToken:    held.RefreshToken,
		AccessTokenTTL:  held.AccessTokenTTL,
		RefreshTokenTTL: held.RefreshTokenTTL,
	}, nil
}

type connectionSave struct {
	owner    stateOwner
	realmID  string
	grant    *services.AccountingTokenGrant
	facts    *accountingsync.CompanyFacts
	app      accountingsync.AppIdentity
	provider string
	now      int64
}

func (s *Service) saveConnection(
	ctx context.Context,
	in *connectionSave,
) (*accountingsync.AccountingConnection, map[string]any, error) {
	owner, grant, facts, provider, now := in.owner, in.grant, in.facts, in.provider, in.now
	var conn *accountingsync.AccountingConnection
	var previous map[string]any
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(
			txCtx,
			repositories.GetAccountingConnectionRequest{
				TenantInfo:      owner.tenantInfo,
				IntegrationType: owner.integrationType,
			},
		)
		if lockErr != nil && !errortypes.IsNotFoundError(lockErr) {
			return lockErr
		}

		if existing == nil {
			conn = &accountingsync.AccountingConnection{
				ID:              pulid.MustNew("acctc_"),
				OrganizationID:  owner.tenantInfo.OrgID,
				BusinessUnitID:  owner.tenantInfo.BuID,
				IntegrationType: owner.integrationType,
				ExternalRealmID: in.realmID,
			}
			conn.BindApp(in.app)
			if sealErr := s.connect(conn, owner.userID, grant, facts, now); sealErr != nil {
				return sealErr
			}
			_, createErr := s.connections.Create(txCtx, conn)
			return createErr
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
		if sealErr := s.connect(existing, owner.userID, grant, facts, now); sealErr != nil {
			return sealErr
		}
		if _, updateErr := s.connections.Update(txCtx, existing); updateErr != nil {
			return updateErr
		}
		if storeErr := s.connections.StoreTokens(txCtx, tokensOf(existing, now)); storeErr != nil {
			return storeErr
		}
		conn = existing
		return nil
	})
	return conn, previous, err
}

func errRealmTaken(provider string) error {
	return errortypes.NewBusinessError(
		"This {0} company is already connected to another Trenova organization. Disconnect it there first.",
		provider,
	)
}

func (s *Service) ensureRealmIsFree(ctx context.Context, in *companyConnect) error {
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		repositories.ListAccountingConnectionsByRealmRequest{
			IntegrationType: in.owner.integrationType,
			RealmIDs:        []string{in.company.ID},
		},
	)
	if err != nil {
		return err
	}
	for _, holder := range holders {
		if holder.OrganizationID != in.owner.tenantInfo.OrgID ||
			holder.BusinessUnitID != in.owner.tenantInfo.BuID {
			return errRealmTaken(in.app.provider)
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
			conn.Status == accountingsync.ConnectionStatusDisconnected {
			continue
		}
		return errortypes.NewBusinessError(
			"Disconnect {0} before connecting {1}. Trenova syncs with one accounting system at a time.",
			accountingsync.ProviderName(conn.IntegrationType),
			accountingsync.ProviderName(typ),
		)
	}
	return nil
}
