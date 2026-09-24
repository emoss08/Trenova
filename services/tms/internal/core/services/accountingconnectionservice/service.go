package accountingconnectionservice

import (
	"context"
	"errors"
	"regexp"
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
	States       repositories.AccountingOAuthStateRepository
	Integrations repositories.IntegrationRepository
	Connectors   services.AccountingConnectorRegistry
	Encryption   *encryptionservice.Service
	AuditService services.AuditService
	Realtime     services.RealtimeService     `optional:"true"`
	Watchtower   services.WatchtowerProjector `optional:"true"`
	Publisher    services.AgentEventPublisher `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	db           ports.DBConnection
	connections  repositories.AccountingConnectionRepository
	states       repositories.AccountingOAuthStateRepository
	integrations repositories.IntegrationRepository
	connectors   services.AccountingConnectorRegistry
	encryption   *encryptionservice.Service
	audit        services.AuditService
	realtime     services.RealtimeService
	watchtower   services.WatchtowerProjector
	publisher    services.AgentEventPublisher
}

var _ services.AccountingConnectionService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.accounting-connection"),
		db:           p.DB,
		connections:  p.Connections,
		states:       p.States,
		integrations: p.Integrations,
		connectors:   p.Connectors,
		encryption:   p.Encryption,
		audit:        p.AuditService,
		realtime:     p.Realtime,
		watchtower:   p.Watchtower,
		publisher:    p.Publisher,
	}
}

func (s *Service) connector(typ integration.Type) (services.AccountingConnector, error) {
	connector, ok := s.connectors.For(typ)
	if !ok || !accountingsync.SupportsAccountingSync(typ) {
		return nil, errortypes.NewValidationError(
			"integrationType",
			errortypes.ErrInvalid,
			"{0} is not an accounting system Trenova can sync with",
			string(typ),
		)
	}
	return connector, nil
}

func (s *Service) Status(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	typ integration.Type,
) (*services.AccountingSyncStatus, error) {
	connector, err := s.connector(typ)
	if err != nil {
		return nil, err
	}

	status := &services.AccountingSyncStatus{
		IntegrationType: typ,
		ProviderName:    accountingsync.ProviderName(typ),
		Available:       connector.Available(),
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
	connector, err := s.connector(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if !connector.Available() {
		return nil, errortypes.NewBusinessError(
			"{0} is not set up on this Trenova instance. Ask your administrator to add the Intuit app credentials.",
			accountingsync.ProviderName(req.IntegrationType),
		)
	}

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
		multiErr.Add("realmId", errortypes.ErrInvalid, "The company id returned by the provider is not valid")
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
	connector, err := s.connector(req.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingsync.ProviderName(req.IntegrationType)

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
	var conn *accountingsync.AccountingConnection
	var previous map[string]any
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(txCtx, repositories.GetAccountingConnectionRequest{
			TenantInfo:      req.TenantInfo,
			IntegrationType: req.IntegrationType,
		})
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
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			s.revokeQuietly(ctx, connector, grant.RefreshToken)
			return nil, errRealmTaken(provider)
		}
		return nil, err
	}

	s.syncIntegrationFlag(ctx, conn)
	s.logAudit(conn, req.UserID, previous, "Connected "+provider+" company "+conn.ExternalCompanyName)
	s.afterHealthChange(ctx, accountingsync.ConnectionStatusDisconnected, conn, now)
	s.publishInvalidation(ctx, conn, req.UserID)

	return conn, nil
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
	holders, err := s.connections.ListHoldingRealm(ctx, repositories.ListAccountingConnectionsByRealmRequest{
		IntegrationType: req.IntegrationType,
		RealmIDs:        []string{req.RealmID},
	})
	if err != nil {
		return err
	}
	for _, holder := range holders {
		if holder.OrganizationID != req.TenantInfo.OrgID || holder.BusinessUnitID != req.TenantInfo.BuID {
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
	conn.ApplyCompanyFacts(*facts)

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
	connector, err := s.connector(req.IntegrationType)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	var conn *accountingsync.AccountingConnection
	var previous map[string]any
	var refreshCiphertext string
	changed := false
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		existing, lockErr := s.connections.LockByTypeWithTokens(txCtx, repositories.GetAccountingConnectionRequest{
			TenantInfo:      req.TenantInfo,
			IntegrationType: req.IntegrationType,
		})
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
		if refreshToken, decryptErr := s.open(conn, refreshTokenField, refreshCiphertext); decryptErr == nil {
			s.revokeQuietly(ctx, connector, refreshToken)
		} else {
			s.l.Warn("could not read refresh token to revoke it", zap.Error(decryptErr))
		}
	}

	s.syncIntegrationFlag(ctx, conn)
	s.logAudit(conn, req.UserID, previous,
		"Disconnected "+accountingsync.ProviderName(conn.IntegrationType)+" company "+conn.ExternalCompanyName)
	s.resolveWatchtower(ctx, conn)
	s.publishInvalidation(ctx, conn, req.UserID)

	return conn, nil
}

type tokenOutcome struct {
	conn        *accountingsync.AccountingConnection
	accessToken string
	failure     error
	category    accountingsync.ErrorCategory
	before      accountingsync.ConnectionStatus
}

func (s *Service) freshAccessToken(
	ctx context.Context,
	connector services.AccountingConnector,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	now int64,
) (*tokenOutcome, error) {
	outcome := &tokenOutcome{}
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		conn, lockErr := s.connections.LockWithTokens(txCtx, repositories.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		})
		if lockErr != nil {
			return lockErr
		}
		outcome.conn = conn
		outcome.before = conn.Status
		if !conn.IsActive() {
			return nil
		}

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
	connector, err := s.connector(current.IntegrationType)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	outcome, err := s.freshAccessToken(ctx, connector, tenantInfo, connectionID, now)
	if err != nil {
		return nil, err
	}
	if !outcome.conn.IsActive() || outcome.failure != nil {
		s.afterHealthChange(ctx, outcome.before, outcome.conn, now)
		return outcome.conn, nil
	}

	facts, factsErr := connector.CompanyFacts(ctx, outcome.conn.ExternalRealmID, outcome.accessToken)

	var conn *accountingsync.AccountingConnection
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		locked, lockErr := s.connections.LockWithTokens(txCtx, repositories.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		})
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
		locked.ApplyCompanyFacts(*facts)
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

func (s *Service) CheckDue(ctx context.Context, limit int) (*services.AccountingHealthSweep, error) {
	now := timeutils.NowUnix()
	due, err := s.connections.ListDueForHealthCheck(ctx, repositories.ListDueAccountingConnectionsRequest{
		CheckedBefore: now - HealthCheckInterval + 60,
		Limit:         limit,
	})
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
	connector, err := s.connector(req.IntegrationType)
	if err != nil {
		return err
	}
	if err = connector.VerifyWebhook(req.Signature, req.Body); err != nil {
		return errortypes.NewAuthenticationError("The webhook signature did not verify").WithInternal(err)
	}

	realms, err := connector.WebhookRealmIDs(req.Body)
	if err != nil {
		return errortypes.NewValidationError("body", errortypes.ErrInvalid, "The webhook body could not be read")
	}

	updated, err := s.connections.MarkWebhookReceived(ctx, repositories.MarkAccountingWebhookRequest{
		IntegrationType: req.IntegrationType,
		RealmIDs:        realms,
		ReceivedAt:      timeutils.NowUnix(),
	})
	if err != nil {
		return err
	}
	if updated == 0 {
		s.l.Debug("webhook for companies with no connection", zap.Int("realms", len(realms)))
	}

	return nil
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

func (s *Service) resolveWatchtower(ctx context.Context, conn *accountingsync.AccountingConnection) {
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

func (s *Service) syncIntegrationFlag(ctx context.Context, conn *accountingsync.AccountingConnection) {
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

func (s *Service) aad(conn *accountingsync.AccountingConnection, field string) encryptionservice.AAD {
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
	access, err := s.encryption.EncryptStringWithAAD(grant.AccessToken, s.aad(conn, accessTokenField))
	if err != nil {
		return accountingsync.TokenGrant{}, err
	}
	refresh, err := s.encryption.EncryptStringWithAAD(grant.RefreshToken, s.aad(conn, refreshTokenField))
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
		TenantInfo:             pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID},
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
