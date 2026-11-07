package dispatchlearningrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning/dispatchlearningdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultTenantLimit = 100
	maxTenantLimit     = 500
)

var styleErrors = dberror.Mapping{Entity: "DispatchStyle"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) dispatchlearningports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dispatch-learning-repository"),
	}
}

func (r *repository) find(ctx context.Context) (*dispatchlearning.DispatchStyle, error) {
	return dispatchlearningdb.DispatchStyles.Cache(resultcache.Settings()...).
		One(ctx, r.db.Keel(ctx))
}

func (r *repository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*dispatchlearning.DispatchStyle, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*dispatchlearning.DispatchStyle, error) {
		rows, err := dispatchlearningdb.DispatchStyles.Cache(resultcache.Settings()...).
			Limit(1).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, styleErrors.Map(err, tenantInfo.OrgID.String())
		}
		if len(rows) == 0 {
			return nil, nil
		}

		return rows[0], nil
	})
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	style *dispatchlearning.DispatchStyle,
) (*dispatchlearning.DispatchStyle, error) {
	s := dispatchlearningdb.DispatchStyles
	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*dispatchlearning.DispatchStyle]{
		Find: r.find,
		Create: func(ctx context.Context) error {
			if _, err := s.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*dispatchlearning.DispatchStyle{style},
				s.OrganizationID,
				s.BusinessUnitID,
			); err != nil {
				return dberror.MapRetryableTransactionError(
					err,
					"Dispatch style is busy. Retry the request.",
				)
			}

			return nil
		},
		Default: func() *dispatchlearning.DispatchStyle { return style },
		MapError: func(err error) error {
			r.l.Error("failed to get or create dispatch style", zap.Error(err))
			return styleErrors.Map(err, style.OrganizationID.String())
		},
	})
}

func (r *repository) Save(
	ctx context.Context,
	style *dispatchlearning.DispatchStyle,
) (*dispatchlearning.DispatchStyle, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*dispatchlearning.DispatchStyle, error) {
			if err := dispatchlearningdb.DispatchStyles.Save(
				ctx,
				r.db.Keel(ctx),
				style,
			); err != nil {
				return nil, styleErrors.Map(err, style.ID.String())
			}

			return style, nil
		},
	)
}

func (r *repository) ListLearningTenants(
	ctx context.Context,
	req *dispatchlearningports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that learn their dispatch style")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		dc := dispatchcontroldb.DispatchControls
		limit := intutils.Clamp(
			intutils.WithDefault(max(req.Limit, 0), defaultTenantLimit),
			1,
			maxTenantLimit,
		)
		q := dc.AllTenants().
			Pick(dc.OrganizationID, dc.BusinessUnitID).
			Where(dc.LearnDispatchStyle.IsTrue()).
			Order(dc.OrganizationID.Asc(), dc.BusinessUnitID.Asc()).
			Limit(limit)
		if !req.After.OrgID.IsNil() {
			q = q.Where(
				keel.Tuple(dc.OrganizationID, dc.BusinessUnitID).
					Gt(keel.Tuple(req.After.OrgID.String(), req.After.BuID.String())),
			)
		}

		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), q)
		if err != nil {
			return nil, fmt.Errorf("list dispatch style tenants: %w", err)
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}
