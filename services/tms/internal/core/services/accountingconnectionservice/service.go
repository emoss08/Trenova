package accountingconnectionservice

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	authorizationStateTTL       = 10 * time.Minute
	accessTokenRefreshWindow    = int64(5 * 60)
	HealthCheckInterval         = int64(15 * 60)
	defaultRefreshTokenLifetime = int64(100 * 24 * 60 * 60)
	accessTokenField            = "access_token"
	refreshTokenField           = "refresh_token"
)

var realmIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,100}$`)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           ports.DBConnection
	Connections  repositories.AccountingConnectionRepository
	Apps         repositories.AccountingAppCredentialRepository
	States       repositories.AccountingOAuthStateRepository
	Integrations repositories.IntegrationRepository
	Connectors   services.AccountingConnectorRegistry
	Encryption   *encryptionservice.Service
	AuditService services.AuditService
	Realtime     services.RealtimeService              `optional:"true"`
	Watchtower   services.WatchtowerProjector          `optional:"true"`
	Publisher    services.AgentEventPublisher          `optional:"true"`
	Refresher    services.AccountingReferenceRefresher `optional:"true"`
	Poller       services.AccountingChangePoller       `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	db           ports.DBConnection
	connections  repositories.AccountingConnectionRepository
	apps         repositories.AccountingAppCredentialRepository
	states       repositories.AccountingOAuthStateRepository
	integrations repositories.IntegrationRepository
	connectors   services.AccountingConnectorRegistry
	encryption   *encryptionservice.Service
	audit        services.AuditService
	realtime     services.RealtimeService
	watchtower   services.WatchtowerProjector
	publisher    services.AgentEventPublisher
	refresher    services.AccountingReferenceRefresher
	poller       services.AccountingChangePoller
}

var _ services.AccountingConnectionService = (*Service)(nil)

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.accounting-connection"),
		db:           p.DB,
		connections:  p.Connections,
		apps:         p.Apps,
		states:       p.States,
		integrations: p.Integrations,
		connectors:   p.Connectors,
		encryption:   p.Encryption,
		audit:        p.AuditService,
		realtime:     p.Realtime,
		watchtower:   p.Watchtower,
		publisher:    p.Publisher,
		refresher:    p.Refresher,
		poller:       p.Poller,
	}
}

func (s *Service) Status(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*services.AccountingSyncStatus, error) {
	provider, err := s.provider(typ)
	if err != nil {
		return nil, err
	}
	settings, err := s.appSettings(ctx, tenantInfo, provider)
	if err != nil {
		return nil, err
	}

	status := &services.AccountingSyncStatus{
		IntegrationType: typ,
		ProviderName:    accountingsync.ProviderName(typ),
		Available:       settings.ActiveSource != "" && settings.RedirectURL != "",
		App:             settings,
	}

	conn, err := s.connections.GetByType(ctx, repositories.GetAccountingConnectionRequest{
		TenantInfo:      tenantInfo,
		IntegrationType: typ,
	})
	switch {
	case err == nil:
		status.Connection = conn
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	return status, nil
}

func (s *Service) StartAuthorization(
	ctx context.Context,
	req *services.StartAccountingAuthorizationRequest,
) (*services.AccountingAuthorizationStart, error) {
	provider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	name := accountingsync.ProviderName(req.IntegrationType)
	if provider.RedirectURL() == "" {
		return nil, errNoRedirect(name)
	}
	app, err := s.appForTenant(ctx, req.TenantInfo, provider)
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

	now := timeutils.NowUnix()
	if err = s.states.Save(ctx, &repositories.AccountingOAuthState{
		State:           tokenHash,
		IntegrationType: req.IntegrationType,
		UserID:          req.UserID,
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		CreatedAt:       now,
		AppSource:       identity.Source,
		AppFingerprint:  identity.Fingerprint,
	}, authorizationStateTTL); err != nil {
		return nil, errortypes.NewBusinessError("Could not start the connection. Try again.").
			WithInternal(err)
	}

	authorizeURL, err := connector.AuthorizeURL(token)
	if err != nil {
		return nil, err
	}

	return &services.AccountingAuthorizationStart{
		AuthorizeURL: authorizeURL,
		ExpiresAt:    now + int64(authorizationStateTTL/time.Second),
	}, nil
}

func validateCompletion(req *services.CompleteAccountingAuthorizationRequest) error {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(req.State) == "" {
		multiErr.Add("state", errortypes.ErrRequired, "The connection request is missing its state")
	}
	if strings.TrimSpace(req.Code) == "" {
		multiErr.Add("code", errortypes.ErrRequired, "The authorization code is missing")
	}
	if !realmIDPattern.MatchString(req.RealmID) {
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
) (*accountingsync.AccountingConnection, error) {
	if err := validateCompletion(req); err != nil {
		return nil, err
	}
	accountingProvider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingsync.ProviderName(req.IntegrationType)

	state, err := s.takeState(ctx, req)
	if err != nil {
		return nil, err
	}
	app, err := s.appForTenant(ctx, req.TenantInfo, accountingProvider)
	if err != nil {
		return nil, err
	}
	identity := app.Identity()
	if state.AppSource != identity.Source || state.AppFingerprint != identity.Fingerprint {
		return nil, errortypes.NewValidationError(
			"state",
			errortypes.ErrInvalid,
			"The {0} app keys changed while you were signing in. Start the connection again.",
			provider,
		)
	}
	connector, err := accountingProvider.Bind(app)
	if err != nil {
		return nil, err
	}

	grant, err := connector.ExchangeCode(ctx, req.Code)
	if err != nil {
		return nil, errortypes.NewBusinessError("{0} did not accept the authorization. Try connecting again.", provider).
			WithInternal(err)
	}

	facts, err := connector.CompanyFacts(ctx, req.RealmID, grant.AccessToken)
	if err != nil {
		s.revokeQuietly(ctx, connector, grant.RefreshToken)
		return nil, errortypes.NewBusinessError("Connected, but {0} would not describe the company. Try connecting again.", provider).
			WithInternal(err)
	}

	if err = s.ensureRealmIsFree(ctx, req, connector, grant); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	conn, previous, err := s.saveConnection(ctx, &connectionSave{
		req:      req,
		grant:    grant,
		facts:    facts,
		app:      identity,
		provider: provider,
		now:      now,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			s.revokeQuietly(ctx, connector, grant.RefreshToken)
			return nil, errRealmTaken(provider)
		}
		return nil, err
	}

	s.syncIntegrationFlag(ctx, conn)
	s.logAudit(
		conn,
		req.UserID,
		previous,
		"Connected "+provider+" company "+conn.ExternalCompanyName,
	)
	s.afterHealthChange(ctx, accountingsync.ConnectionStatusDisconnected, conn, now)
	s.publishInvalidation(ctx, conn, req.UserID)
	s.requestReferenceRefresh(ctx, conn)

	return conn, nil
}

func (s *Service) requestReferenceRefresh(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) {
	if s.refresher == nil {
		return
	}
	if err := s.refresher.RequestReferenceRefresh(ctx, pagination.TenantInfo{
		OrgID: conn.OrganizationID,
		BuID:  conn.BusinessUnitID,
	}, conn.ID); err != nil {
		s.l.Warn("could not start the reference data refresh after connecting",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
	}
}

func (s *Service) takeState(
	ctx context.Context,
	req *services.CompleteAccountingAuthorizationRequest,
) (*repositories.AccountingOAuthState, error) {
	state, err := s.states.Take(ctx, tokenutils.Hash(req.State))
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"state",
				errortypes.ErrInvalid,
				"This connection request expired or was already used. Start the connection again.",
			)
		}
		return nil, err
	}
	if state.UserID != req.UserID ||
		state.OrganizationID != req.TenantInfo.OrgID ||
		state.BusinessUnitID != req.TenantInfo.BuID ||
		state.IntegrationType != req.IntegrationType {
		return nil, errortypes.NewAuthorizationError(
			"This connection was started by someone else. Start the connection again from your own account.",
		)
	}

	return state, nil
}

type connectionSave struct {
	req      *services.CompleteAccountingAuthorizationRequest
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
	req, grant, facts, provider, now := in.req, in.grant, in.facts, in.provider, in.now
	var conn *accountingsync.AccountingConnection
	var previous map[string]any
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(
			txCtx,
			repositories.GetAccountingConnectionRequest{
				TenantInfo:      req.TenantInfo,
				IntegrationType: req.IntegrationType,
			},
		)
		if lockErr != nil && !errortypes.IsNotFoundError(lockErr) {
			return lockErr
		}

		if existing == nil {
			conn = &accountingsync.AccountingConnection{
				ID:              pulid.MustNew("acctc_"),
				OrganizationID:  req.TenantInfo.OrgID,
				BusinessUnitID:  req.TenantInfo.BuID,
				IntegrationType: req.IntegrationType,
				ExternalRealmID: req.RealmID,
			}
			conn.BindApp(in.app)
			if sealErr := s.connect(conn, req.UserID, grant, facts, now); sealErr != nil {
				return sealErr
			}
			_, createErr := s.connections.Create(txCtx, conn)
			return createErr
		}

		if existing.IsActive() && existing.ExternalRealmID != req.RealmID {
			return errortypes.NewBusinessError(
				"Disconnect {0} before connecting a different {1} company.",
				existing.ExternalCompanyName,
				provider,
			)
		}

		previous = jsonutils.MustToJSON(existing)
		existing.ExternalRealmID = req.RealmID
		existing.BindApp(in.app)
		if sealErr := s.connect(existing, req.UserID, grant, facts, now); sealErr != nil {
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

func (s *Service) ensureRealmIsFree(
	ctx context.Context,
	req *services.CompleteAccountingAuthorizationRequest,
	connector services.AccountingConnector,
	grant *services.AccountingTokenGrant,
) error {
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		repositories.ListAccountingConnectionsByRealmRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        []string{req.RealmID},
		},
	)
	if err != nil {
		return err
	}
	for _, holder := range holders {
		if holder.OrganizationID != req.TenantInfo.OrgID ||
			holder.BusinessUnitID != req.TenantInfo.BuID {
			s.revokeQuietly(ctx, connector, grant.RefreshToken)
			return errRealmTaken(accountingsync.ProviderName(req.IntegrationType))
		}
	}
	return nil
}

func (s *Service) connect(
	conn *accountingsync.AccountingConnection,
	userID pulid.ID,
	grant *services.AccountingTokenGrant,
	facts *accountingsync.CompanyFacts,
	now int64,
) error {
	sealed, err := s.seal(conn, grant, now)
	if err != nil {
		return err
	}
	conn.Connect(userID, sealed, now)
	conn.ApplyCompanyFacts(facts)

	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) Disconnect(
	ctx context.Context,
	req *services.DisconnectAccountingRequest,
) (*accountingsync.AccountingConnection, error) {
	provider, err := s.provider(req.IntegrationType)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	var conn *accountingsync.AccountingConnection
	var previous map[string]any
	var refreshCiphertext string
	changed := false
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(
			txCtx,
			repositories.GetAccountingConnectionRequest{
				TenantInfo:      req.TenantInfo,
				IntegrationType: req.IntegrationType,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		conn = existing
		if existing.Status == accountingsync.ConnectionStatusDisconnected {
			return nil
		}

		previous = jsonutils.MustToJSON(existing)
		refreshCiphertext = existing.RefreshTokenCiphertext
		existing.Disconnect(req.UserID, now)
		if _, updateErr := s.connections.Update(txCtx, existing); updateErr != nil {
			return updateErr
		}
		changed = true
		return s.connections.StoreTokens(txCtx, tokensOf(existing, now))
	})
	if err != nil {
		return nil, err
	}
	if !changed {
		return conn, nil
	}

	if refreshCiphertext != "" {
		s.revokeStored(ctx, provider, conn, refreshCiphertext)
	}

	s.syncIntegrationFlag(ctx, conn)
	s.logAudit(
		conn,
		req.UserID,
		previous,
		"Disconnected "+accountingsync.ProviderName(
			conn.IntegrationType,
		)+" company "+conn.ExternalCompanyName,
	)
	s.resolveWatchtower(ctx, conn)
	s.publishInvalidation(ctx, conn, req.UserID)

	return conn, nil
}

func (s *Service) revokeStored(
	ctx context.Context,
	provider services.AccountingProvider,
	conn *accountingsync.AccountingConnection,
	refreshCiphertext string,
) {
	connector, err := s.connectorFor(ctx, provider, conn)
	if err != nil {
		s.l.Warn("could not resolve the app to revoke the authorization with", zap.Error(err))
		return
	}
	refreshToken, err := s.open(conn, refreshTokenField, refreshCiphertext)
	if err != nil {
		s.l.Warn("could not read refresh token to revoke it", zap.Error(err))
		return
	}
	s.revokeQuietly(ctx, connector, refreshToken)
}

type tokenOutcome struct {
	conn        *accountingsync.AccountingConnection
	connector   services.AccountingConnector
	accessToken string
	failure     error
	category    accountingsync.ErrorCategory
	before      accountingsync.ConnectionStatus
}

func (s *Service) freshAccessToken(
	ctx context.Context,
	provider services.AccountingProvider,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	now int64,
) (*tokenOutcome, error) {
	outcome := &tokenOutcome{}
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		conn, lockErr := s.connections.LockWithTokens(
			txCtx,
			repositories.GetAccountingConnectionByIDRequest{
				TenantInfo: tenantInfo,
				ID:         connectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		outcome.conn = conn
		outcome.before = conn.Status
		if !conn.IsActive() {
			return nil
		}

		connector, bindErr := s.connectorFor(txCtx, provider, conn)
		if bindErr != nil {
			outcome.failure = bindErr
			outcome.category = accountingsync.ErrorCategoryConfiguration
			return s.recordFailure(txCtx, conn, outcome.category, bindErr, now)
		}
		outcome.connector = connector

		if conn.HasTokens() && conn.AccessTokenExpiresWithin(now, accessTokenRefreshWindow) {
			if refreshErr := s.refresh(ctx, txCtx, connector, conn, now); refreshErr != nil {
				outcome.failure = refreshErr
				outcome.category = connector.ClassifyError(refreshErr)
				return s.recordFailure(txCtx, conn, outcome.category, refreshErr, now)
			}
		}

		if !conn.HasTokens() {
			outcome.failure = errors.New("no stored authorization")
			outcome.category = accountingsync.ErrorCategoryRevoked
			return s.recordFailure(txCtx, conn, outcome.category, outcome.failure, now)
		}

		access, openErr := s.open(conn, accessTokenField, conn.AccessTokenCiphertext)
		if openErr != nil {
			outcome.failure = openErr
			outcome.category = accountingsync.ErrorCategoryConfiguration
			return s.recordFailure(txCtx, conn, outcome.category, errors.New(
				"the stored authorization could not be read; reconnect to replace it",
			), now)
		}
		outcome.accessToken = access
		return nil
	})
	return outcome, err
}

func (s *Service) refresh(
	ctx, txCtx context.Context,
	connector services.AccountingConnector,
	conn *accountingsync.AccountingConnection,
	now int64,
) error {
	refreshToken, err := s.open(conn, refreshTokenField, conn.RefreshTokenCiphertext)
	if err != nil {
		return err
	}
	grant, err := connector.Refresh(ctx, refreshToken)
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

func (s *Service) recordFailure(
	txCtx context.Context,
	conn *accountingsync.AccountingConnection,
	category accountingsync.ErrorCategory,
	cause error,
	now int64,
) error {
	conn.RecordFailure(category, failureMessage(category, cause), now)
	if _, err := s.connections.Update(txCtx, conn); err != nil {
		return err
	}
	if category == accountingsync.ErrorCategoryRevoked {
		return s.connections.StoreTokens(txCtx, tokensOf(conn, now))
	}
	return nil
}

func (s *Service) CheckHealth(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingsync.AccountingConnection, error) {
	current, err := s.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
	})
	if err != nil {
		return nil, err
	}
	provider, err := s.provider(current.IntegrationType)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	outcome, err := s.freshAccessToken(ctx, provider, tenantInfo, connectionID, now)
	if err != nil {
		return nil, err
	}
	if !outcome.conn.IsActive() || outcome.failure != nil {
		s.afterHealthChange(ctx, outcome.before, outcome.conn, now)
		return outcome.conn, nil //nolint:nilerr // a failed check is recorded as the connection's health, not returned
	}

	connector := outcome.connector
	facts, factsErr := connector.CompanyFacts(
		ctx,
		outcome.conn.ExternalRealmID,
		outcome.accessToken,
	)

	var conn *accountingsync.AccountingConnection
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			repositories.GetAccountingConnectionByIDRequest{
				TenantInfo: tenantInfo,
				ID:         connectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		conn = locked
		if !locked.IsActive() {
			return nil
		}
		if factsErr != nil {
			return s.recordFailure(txCtx, locked, connector.ClassifyError(factsErr), factsErr, now)
		}
		locked.ApplyCompanyFacts(facts)
		locked.RecordSuccess(now)
		_, updateErr := s.connections.Update(txCtx, locked)
		return updateErr
	})
	if err != nil {
		return nil, err
	}

	s.afterHealthChange(ctx, outcome.before, conn, now)
	return conn, nil
}

func (s *Service) Session(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*services.AccountingSession, error) {
	current, err := s.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
	})
	if err != nil {
		return nil, err
	}
	accountingProvider, err := s.provider(current.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingsync.ProviderName(current.IntegrationType)
	if !current.IsActive() {
		return nil, errNotConnected(provider)
	}

	now := timeutils.NowUnix()
	outcome, err := s.freshAccessToken(ctx, accountingProvider, tenantInfo, connectionID, now)
	if err != nil {
		return nil, err
	}
	if outcome.failure != nil || !outcome.conn.IsActive() {
		s.afterHealthChange(ctx, outcome.before, outcome.conn, now)
		return nil, errortypes.NewBusinessError(
			"{0} did not accept Trenova's authorization: {1}",
			provider,
			outcome.conn.AgentErrorSummary(),
		).WithInternal(outcome.failure)
	}

	return &services.AccountingSession{
		Connection:  outcome.conn,
		AccessToken: outcome.accessToken,
		Connector:   outcome.connector,
	}, nil
}

func (s *Service) ReportCallFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	cause error,
) accountingsync.ErrorCategory {
	current, err := s.connections.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenantInfo,
		ID:         connectionID,
	})
	if err != nil {
		s.l.Warn("could not load the connection to record a failed call", zap.Error(err))
		return accountingsync.ErrorCategoryUnknown
	}
	provider, err := s.provider(current.IntegrationType)
	if err != nil {
		return accountingsync.ErrorCategoryConfiguration
	}

	category := provider.ClassifyError(cause)
	if category != accountingsync.ErrorCategoryUnauthorized &&
		category != accountingsync.ErrorCategoryRevoked {
		return category
	}

	now := timeutils.NowUnix()
	var conn *accountingsync.AccountingConnection
	txErr := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			repositories.GetAccountingConnectionByIDRequest{
				TenantInfo: tenantInfo,
				ID:         connectionID,
			},
		)
		if lockErr != nil {
			return lockErr
		}
		conn = locked
		if !locked.IsActive() {
			return nil
		}
		return s.recordFailure(txCtx, locked, category, cause, now)
	})
	if txErr != nil {
		s.l.Warn("could not record a failed call on the connection", zap.Error(txErr))
		return category
	}

	s.afterHealthChange(ctx, current.Status, conn, now)
	return category
}

func errNotConnected(provider string) error {
	return errortypes.NewBusinessError(
		"{0} is not connected. A person must connect it from the integrations page.",
		provider,
	)
}

func (s *Service) CheckDue(
	ctx context.Context,
	limit int,
) (*services.AccountingHealthSweep, error) {
	now := timeutils.NowUnix()
	due, err := s.connections.ListDueForHealthCheck(
		ctx,
		repositories.ListDueAccountingConnectionsRequest{
			CheckedBefore: now - HealthCheckInterval + 60,
			Limit:         limit,
		},
	)
	if err != nil {
		return nil, err
	}

	sweep := &services.AccountingHealthSweep{Listed: len(due)}
	for _, conn := range due {
		if ctx.Err() != nil {
			return sweep, ctx.Err()
		}
		tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
		if _, checkErr := s.CheckHealth(ctx, tenant, conn.ID); checkErr != nil {
			sweep.Failed++
			s.l.Warn("accounting connection health check failed",
				zap.String("connectionId", conn.ID.String()), zap.Error(checkErr))
			continue
		}
		sweep.Checked++
	}

	return sweep, nil
}

func (s *Service) ReceiveWebhook(
	ctx context.Context,
	req *services.ReceiveAccountingWebhookRequest,
) error {
	provider, err := s.provider(req.IntegrationType)
	if err != nil {
		return err
	}

	realms, err := provider.WebhookRealmIDs(req.Body)
	if err != nil {
		return errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The webhook body could not be read",
		)
	}
	holders, err := s.connections.ListHoldingRealm(
		ctx,
		repositories.ListAccountingConnectionsByRealmRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        realms,
		},
	)
	if err != nil {
		return err
	}
	if len(holders) == 0 {
		s.l.Debug("webhook for companies with no connection", zap.Int("realms", len(realms)))
		return nil
	}

	verified := s.verifiedRealms(ctx, provider, holders, req)
	if len(verified) == 0 {
		return errortypes.NewAuthenticationError("The webhook signature did not verify")
	}

	if _, err = s.connections.MarkWebhookReceived(
		ctx,
		repositories.MarkAccountingWebhookRequest{
			IntegrationType: req.IntegrationType,
			RealmIDs:        verified,
			ReceivedAt:      timeutils.NowUnix(),
		},
	); err != nil {
		return err
	}

	s.pollChanged(ctx, holders, verified)
	return nil
}

func (s *Service) pollChanged(
	ctx context.Context,
	holders []*accountingsync.AccountingConnection,
	verified []string,
) {
	if s.poller == nil {
		return
	}
	for _, holder := range holders {
		if !slices.Contains(verified, holder.ExternalRealmID) || !holder.ReadsChanges() {
			continue
		}
		tenant := pagination.TenantInfo{OrgID: holder.OrganizationID, BuID: holder.BusinessUnitID}
		if err := s.poller.PollNow(ctx, tenant, holder.ID); err != nil {
			s.l.Warn("failed to wake the change reader after a webhook",
				zap.String("connectionId", holder.ID.String()), zap.Error(err))
		}
	}
}

func (s *Service) verifiedRealms(
	ctx context.Context,
	provider services.AccountingProvider,
	holders []*accountingsync.AccountingConnection,
	req *services.ReceiveAccountingWebhookRequest,
) []string {
	byApp := make(map[string]bool, len(holders))
	verified := make([]string, 0, len(holders))
	for _, holder := range holders {
		app, err := s.appForConnection(ctx, provider, holder)
		if err != nil {
			continue
		}
		key := string(app.Source) + ":" + app.Identity().Fingerprint
		ok, seen := byApp[key]
		if !seen {
			connector, bindErr := provider.Bind(app)
			ok = bindErr == nil && connector.VerifyWebhook(req.Signature, req.Body) == nil
			byApp[key] = ok
		}
		if ok {
			verified = append(verified, holder.ExternalRealmID)
		}
	}
	return verified
}

func (s *Service) afterHealthChange(
	ctx context.Context,
	before accountingsync.ConnectionStatus,
	conn *accountingsync.AccountingConnection,
	now int64,
) {
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}

	if item, open := watchtowersources.DescribeAccountingConnectionHealth(conn); open {
		if s.watchtower != nil {
			s.watchtower.Upsert(ctx, item)
		}
		if worsened(before, conn.Status) {
			services.PublishAgentEvent(ctx, s.publisher, services.AgentEvent{
				Kind:       agent.EventAccountingConnectionDegraded,
				SubjectID:  conn.ID,
				TenantInfo: tenant,
			})
		}
	} else if s.watchtower != nil {
		s.watchtower.Resolve(
			ctx,
			tenant,
			watchtower.SourceAccountingSync,
			watchtowersources.AccountingConnectionHealthSourceID(conn),
		)
	}

	if s.watchtower != nil {
		if item, open := watchtowersources.DescribeAccountingReconnect(conn, now); open {
			s.watchtower.Upsert(ctx, item)
		} else {
			s.watchtower.Resolve(
				ctx,
				tenant,
				watchtower.SourceAccountingSync,
				watchtowersources.AccountingReconnectSourceID(conn),
			)
		}
	}

	if before != conn.Status {
		s.syncIntegrationFlag(ctx, conn)
		s.publishInvalidation(ctx, conn, pulid.Nil)
	}
}

func (s *Service) resolveWatchtower(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) {
	if s.watchtower == nil {
		return
	}
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	s.watchtower.Resolve(
		ctx,
		tenant,
		watchtower.SourceAccountingSync,
		watchtowersources.AccountingConnectionHealthSourceID(conn),
	)
	s.watchtower.Resolve(
		ctx,
		tenant,
		watchtower.SourceAccountingSync,
		watchtowersources.AccountingReconnectSourceID(conn),
	)
}

func worsened(before, after accountingsync.ConnectionStatus) bool {
	return severityRank(after) > severityRank(before)
}

func severityRank(status accountingsync.ConnectionStatus) int {
	switch status { //nolint:exhaustive // connected and disconnected rank lowest
	case accountingsync.ConnectionStatusDegraded:
		return 1
	case accountingsync.ConnectionStatusFailing:
		return 2
	case accountingsync.ConnectionStatusRevoked:
		return 3
	default:
		return 0
	}
}

func (s *Service) syncIntegrationFlag(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) {
	tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
	enabled := conn.IsActive()

	record, err := s.integrations.GetByType(ctx, tenant, conn.IntegrationType)
	switch {
	case err == nil && record.Enabled == enabled:
		return
	case err == nil:
		record.Enabled = enabled
		if enabled {
			record.EnabledByID = conn.ConnectedByID
		}
	case errortypes.IsNotFoundError(err):
		if !enabled {
			return
		}
		record = &integration.Integration{
			OrganizationID: conn.OrganizationID,
			BusinessUnitID: conn.BusinessUnitID,
			Type:           conn.IntegrationType,
			Name:           accountingsync.ProviderName(conn.IntegrationType),
			Category:       integration.CategoryAccounting,
			Enabled:        true,
			EnabledByID:    conn.ConnectedByID,
		}
	default:
		s.l.Warn("could not read integration record", zap.Error(err))
		return
	}

	if record.Enabled && record.EnabledByID.IsNil() {
		record.EnabledByID = conn.ConnectedByID
	}
	if _, err = s.integrations.Upsert(ctx, record); err != nil {
		s.l.Warn("could not update integration record", zap.Error(err))
	}
}

func (s *Service) logAudit(
	conn *accountingsync.AccountingConnection,
	userID pulid.ID,
	previous map[string]any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceAccountingIntegration,
		ResourceID:     conn.ID.String(),
		Operation:      permission.OpManage,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(conn),
		PreviousState:  previous,
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		Critical:       true,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log accounting connection audit", zap.Error(err))
	}
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		ActorUserID:    userID,
		Resource:       permission.ResourceAccountingIntegration.String(),
		Action:         "updated",
		RecordID:       conn.ID,
	}); err != nil {
		s.l.Warn("failed to publish accounting connection invalidation", zap.Error(err))
	}
}

func (s *Service) revokeQuietly(
	ctx context.Context,
	connector services.AccountingConnector,
	refreshToken string,
) {
	if refreshToken == "" {
		return
	}
	if err := connector.Revoke(ctx, refreshToken); err != nil {
		s.l.Warn("could not revoke accounting authorization", zap.Error(err))
	}
}

func (s *Service) aad(
	conn *accountingsync.AccountingConnection,
	field string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeAccountingConnectionToken,
		OrganizationID: conn.OrganizationID,
		BusinessUnitID: conn.BusinessUnitID,
		ResourceID:     conn.ID.String() + ":" + field,
	}
}

func (s *Service) seal(
	conn *accountingsync.AccountingConnection,
	grant *services.AccountingTokenGrant,
	now int64,
) (accountingsync.TokenGrant, error) {
	access, err := s.encryption.EncryptStringWithAAD(
		grant.AccessToken,
		s.aad(conn, accessTokenField),
	)
	if err != nil {
		return accountingsync.TokenGrant{}, err
	}
	refresh, err := s.encryption.EncryptStringWithAAD(
		grant.RefreshToken,
		s.aad(conn, refreshTokenField),
	)
	if err != nil {
		return accountingsync.TokenGrant{}, err
	}

	refreshLifetime := int64(grant.RefreshTokenTTL / time.Second)
	if refreshLifetime <= 0 {
		refreshLifetime = defaultRefreshTokenLifetime
	}

	return accountingsync.TokenGrant{
		AccessTokenCiphertext:  access,
		AccessTokenExpiresAt:   now + int64(grant.AccessTokenTTL/time.Second),
		RefreshTokenCiphertext: refresh,
		RefreshTokenExpiresAt:  now + refreshLifetime,
	}, nil
}

func (s *Service) open(
	conn *accountingsync.AccountingConnection,
	field, ciphertext string,
) (string, error) {
	return s.encryption.DecryptStringWithAAD(ciphertext, s.aad(conn, field))
}

func tokensOf(
	conn *accountingsync.AccountingConnection,
	now int64,
) repositories.StoreAccountingTokensRequest {
	return repositories.StoreAccountingTokensRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: conn.OrganizationID,
			BuID:  conn.BusinessUnitID,
		},
		ID:                     conn.ID,
		AccessTokenCiphertext:  conn.AccessTokenCiphertext,
		AccessTokenExpiresAt:   conn.AccessTokenExpiresAt,
		RefreshTokenCiphertext: conn.RefreshTokenCiphertext,
		RefreshTokenExpiresAt:  conn.RefreshTokenExpiresAt,
		RefreshedAt:            conn.LastRefreshedAt,
		At:                     now,
	}
}

func failureMessage(category accountingsync.ErrorCategory, cause error) string {
	switch category { //nolint:exhaustive // other categories describe themselves through the cause
	case accountingsync.ErrorCategoryRevoked:
		return "The authorization was revoked or has expired. Reconnect to resume."
	case accountingsync.ErrorCategoryRateLimited:
		return "The provider asked Trenova to slow down; calls resume automatically."
	case accountingsync.ErrorCategoryUnauthorized:
		return "The provider refused the stored authorization. Reconnect if this continues."
	case accountingsync.ErrorCategoryTransient:
		return "The provider did not answer in time; Trenova retries automatically."
	default:
		if cause == nil {
			return ""
		}
		return cause.Error()
	}
}
