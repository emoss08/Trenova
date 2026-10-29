package accountingconnectionhealth

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	HealthCheckInterval = int64(15 * 60)
)

func (s *Service) CheckHealth(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingconnection.AccountingConnection, error) {
	current, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	provider, err := s.kit.Provider(current.IntegrationType)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	outcome, err := s.kit.FreshAccessToken(ctx, provider, tenantInfo, connectionID, now)
	if err != nil {
		return nil, err
	}
	if !outcome.Conn.IsActive() || outcome.Failure != nil {
		s.kit.AfterHealthChange(ctx, outcome.Before, outcome.Conn, now)
		return outcome.Conn, nil //nolint:nilerr // a failed check is recorded as the connection's health, not returned
	}

	connector := outcome.Connector
	facts, factsErr := connector.CompanyFacts(
		ctx,
		outcome.Conn.ExternalRealmID,
		outcome.AccessToken,
	)

	var conn *accountingconnection.AccountingConnection
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionByIDRequest{
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
			return s.kit.RecordFailure(
				txCtx,
				locked,
				connector.ClassifyError(factsErr),
				factsErr,
				now,
			)
		}
		locked.ApplyCompanyFacts(facts)
		locked.RecordSuccess(now)
		_, updateErr := s.connections.Update(txCtx, locked)
		return updateErr
	})
	if err != nil {
		return nil, err
	}

	s.kit.AfterHealthChange(ctx, outcome.Before, conn, now)
	return conn, nil
}

func (s *Service) Session(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) (*accountingconnectionports.AccountingSession, error) {
	current, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		},
	)
	if err != nil {
		return nil, err
	}
	accountingProvider, err := s.kit.Provider(current.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingprovider.ProviderName(current.IntegrationType)
	if !current.IsActive() {
		return nil, errNotConnected(provider)
	}

	now := timeutils.NowUnix()
	outcome, err := s.kit.FreshAccessToken(ctx, accountingProvider, tenantInfo, connectionID, now)
	if err != nil {
		return nil, err
	}
	if outcome.Failure != nil || !outcome.Conn.IsActive() {
		s.kit.AfterHealthChange(ctx, outcome.Before, outcome.Conn, now)
		return nil, errortypes.NewBusinessError(
			"{0} did not accept Trenova's authorization: {1}",
			provider,
			outcome.Conn.AgentErrorSummary(),
		).WithInternal(outcome.Failure)
	}

	return &accountingconnectionports.AccountingSession{
		Connection:  outcome.Conn,
		AccessToken: outcome.AccessToken,
		Connector:   outcome.Connector,
	}, nil
}

func (s *Service) ReportCallFailure(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
	cause error,
) accountingsync.ErrorCategory {
	current, err := s.connections.GetByID(
		ctx,
		accountingconnectionports.GetAccountingConnectionByIDRequest{
			TenantInfo: tenantInfo,
			ID:         connectionID,
		},
	)
	if err != nil {
		s.l.Warn("could not load the connection to record a failed call", zap.Error(err))
		return accountingsync.ErrorCategoryUnknown
	}
	provider, err := s.kit.Provider(current.IntegrationType)
	if err != nil {
		return accountingsync.ErrorCategoryConfiguration
	}

	category := provider.ClassifyError(cause)
	if category != accountingsync.ErrorCategoryUnauthorized &&
		category != accountingsync.ErrorCategoryRevoked {
		return category
	}

	now := timeutils.NowUnix()
	var conn *accountingconnection.AccountingConnection
	txErr := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		locked, lockErr := s.connections.LockWithTokens(
			txCtx,
			accountingconnectionports.GetAccountingConnectionByIDRequest{
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
		return s.kit.RecordFailure(txCtx, locked, category, cause, now)
	})
	if txErr != nil {
		s.l.Warn("could not record a failed call on the connection", zap.Error(txErr))
		return category
	}

	s.kit.AfterHealthChange(ctx, current.Status, conn, now)
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
) (*accountingconnectionports.AccountingHealthSweep, error) {
	now := timeutils.NowUnix()
	due, err := s.connections.ListDueForHealthCheck(
		ctx,
		accountingconnectionports.ListDueAccountingConnectionsRequest{
			CheckedBefore: now - HealthCheckInterval + 60,
			Limit:         limit,
		},
	)
	if err != nil {
		return nil, err
	}

	sweep := &accountingconnectionports.AccountingHealthSweep{Listed: len(due)}
	for _, conn := range due {
		if ctx.Err() != nil {
			return sweep, ctx.Err()
		}
		tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
		if _, checkErr := s.CheckHealth(
			dbscope.WithTenant(ctx, tenant.DBTenant()),
			tenant,
			conn.ID,
		); checkErr != nil {
			sweep.Failed++
			s.l.Warn("accounting connection health check failed",
				zap.String("connectionId", conn.ID.String()), zap.Error(checkErr))
			continue
		}
		sweep.Checked++
	}

	return sweep, nil
}

type Deps struct {
	Logger      *zap.Logger
	DB          kernelports.Transactor
	Connections accountingconnectionports.AccountingConnectionRepository
	Kit         *accountingconnectionkit.Kit
}

type Service struct {
	l           *zap.Logger
	db          kernelports.Transactor
	connections accountingconnectionports.AccountingConnectionRepository
	kit         *accountingconnectionkit.Kit
}

func New(d Deps) *Service {
	return &Service{l: d.Logger, db: d.DB, connections: d.Connections, kit: d.Kit}
}
