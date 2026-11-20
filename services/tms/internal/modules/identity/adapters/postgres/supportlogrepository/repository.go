package supportlogrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog/supportlogdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	instanceLoadReason = "count the instance's log requests in flight and today: every organization shares their bounds"
	operatorReadReason = "the operator reads and reports on the whole instance's log requests through the managed listener"
	overdueSweepReason = "fail the log requests the operator never picked up or never finished, across organizations"
)

var requestErrors = dberror.Mapping{Entity: "Log request"}

type Params struct {
	fx.In

	Entitlements entitlementports.Filter
	DB           *postgres.Connection
	Logger       *zap.Logger
}

type repository struct {
	entitlements entitlementports.Filter
	db           *postgres.Connection
	l            *zap.Logger
}

func New(p Params) supportlogports.Repository {
	return &repository{
		db:           p.DB,
		l:            p.Logger.Named("postgres.support-log-request-repository"),
		entitlements: p.Entitlements,
	}
}

func inFlight() keel.Pred {
	l := supportlogdb.LogRequests
	return l.Status.In(supportlog.StatusRequested, supportlog.StatusCollecting)
}

func (r *repository) Create(
	ctx context.Context,
	req *supportlog.LogRequest,
) (*supportlog.LogRequest, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*supportlog.LogRequest, error) {
		if err := supportlogdb.LogRequests.Create(ctx, r.db.Keel(ctx), req); err != nil {
			return nil, requestErrors.Map(err, "")
		}
		return req, nil
	})
}

func (r *repository) Recent(ctx context.Context, limit int) ([]*supportlog.LogRequest, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportlog.LogRequest, error) {
		l := supportlogdb.LogRequests
		rows, err := l.Query().
			Order(l.CreatedAt.Desc(), l.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list recent log requests: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) Requester(
	ctx context.Context,
	userID pulid.ID,
) (*supportlogports.Requester, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportlogports.Requester, error) {
		u := tenantdb.Users
		user, err := u.Where(u.ID.Eq(userID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.Mapping{Entity: "User"}.Map(err, userID.String())
		}
		return &supportlogports.Requester{
			UserID: user.ID,
			Name:   user.Name,
			Email:  user.EmailAddress,
		}, nil
	})
}

func (r *repository) InstanceLoad(ctx context.Context, since int64) (*supportlog.Load, error) {
	ctx = dbscope.WithSystem(ctx, instanceLoadReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportlog.Load, error) {
		l := supportlogdb.LogRequests
		var flying, recent int64
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.CountInto(l.Where(inFlight()).AllTenants(), &flying),
			keel.CountInto(l.Where(l.CreatedAt.Gt(since)).AllTenants(), &recent),
		)
		if err != nil {
			return nil, fmt.Errorf("count the instance's log requests: %w", err)
		}
		return &supportlog.Load{InFlight: flying > 0, Recent: int(recent)}, nil
	})
}

func (r *repository) Waiting(ctx context.Context, limit int) ([]*supportlog.LogRequest, error) {
	ctx = dbscope.WithSystem(ctx, operatorReadReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportlog.LogRequest, error) {
		l := supportlogdb.LogRequests
		rows, err := l.Where(inFlight()).
			AllTenants().
			Order(l.CreatedAt.Asc(), l.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list the log requests in flight: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) ForOperator(
	ctx context.Context,
	id pulid.ID,
) (*supportlog.LogRequest, error) {
	ctx = dbscope.WithSystem(ctx, operatorReadReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportlog.LogRequest, error) {
		l := supportlogdb.LogRequests
		row, err := l.Where(l.ID.Eq(id)).AllTenants().One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, supportlog.ErrRequestNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("read a log request for the operator: %w", err)
		}
		return row, nil
	})
}

func (r *repository) SaveReport(
	ctx context.Context,
	req *supportlog.LogRequest,
) (*supportlog.LogRequest, error) {
	ctx = dbscope.WithSystem(ctx, operatorReadReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*supportlog.LogRequest, error) {
		l := supportlogdb.LogRequests
		err := l.Save(
			ctx,
			r.db.Keel(ctx),
			req,
			l.Status,
			l.BundleID,
			l.FailureMessage,
			l.CollectingAt,
			l.FinishedAt,
		)
		if errors.Is(err, keel.ErrConflict) || errors.Is(err, keel.ErrNotFound) {
			return nil, supportlog.ErrChangedMeanwhile
		}
		if err != nil {
			return nil, requestErrors.Map(err, req.ID.String())
		}
		return req, nil
	})
}

func (r *repository) Overdue(
	ctx context.Context,
	req *supportlogports.OverdueRequest,
) ([]*supportlog.LogRequest, error) {
	ctx = dbscope.WithSystem(ctx, overdueSweepReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportlog.LogRequest, error) {
		l := supportlogdb.LogRequests
		entitled, entitledErr := r.entitlements.ForWork(ctx, l.OrganizationID.Col)
		if entitledErr != nil {
			return nil, entitledErr
		}
		rows, err := l.Where(entitled, keel.Or(
			keel.And(l.Status.Eq(supportlog.StatusRequested), l.CreatedAt.Lte(req.RequestedBefore)),
			keel.And(
				l.Status.Eq(supportlog.StatusCollecting),
				l.CollectingAt.Lte(req.CollectingBefore),
			),
		)).
			AllTenants().
			Order(l.CreatedAt.Asc(), l.ID.Asc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list overdue log requests: %w", err)
		}
		return rows, nil
	})
}
