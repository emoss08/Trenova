package instancebootstraprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	stateScopeReason    = "read whether this instance has been bootstrapped and whether any user exists in any organization"
	finalizeScopeReason = "record the instance bootstrap and create the instance-wide system user before any tenant session exists"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.InstanceBootstrapRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.instance-bootstrap-repository"),
	}
}

func (r *repository) GetState(ctx context.Context) (*repositories.InstanceBootstrapState, error) {
	ctx = dbscope.WithSystem(ctx, stateScopeReason)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*repositories.InstanceBootstrapState, error) {
			db := r.db.DBForContext(ctx)

			state := new(repositories.InstanceBootstrapState)
			record := new(instancebootstrap.InstanceBootstrap)
			err := db.NewSelect().
				Model(record).
				Limit(1).
				Scan(ctx)
			switch {
			case err == nil:
				state.Record = record
			case !dberror.IsNotFoundError(err):
				return nil, fmt.Errorf("read instance bootstrap: %w", err)
			}

			userCols := buncolgen.UserColumns
			count, err := db.NewSelect().
				Model((*tenant.User)(nil)).
				Where("lower("+userCols.Username.Qualified()+") <> ?", tenant.SystemUsername).
				Count(ctx)
			if err != nil {
				return nil, fmt.Errorf("count users: %w", err)
			}
			state.UserCount = count

			return state, nil
		},
	)
}

func (r *repository) Finalize(
	ctx context.Context,
	req *repositories.FinalizeInstanceBootstrapRequest,
) (*repositories.FinalizeInstanceBootstrapResult, error) {
	ctx = dbscope.WithSystem(ctx, finalizeScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*repositories.FinalizeInstanceBootstrapResult, error) {
			db := r.db.DBForContext(ctx)
			scope := tenantbootstrap.Scope{
				OrganizationID: req.OrganizationID,
				BusinessUnitID: req.BusinessUnitID,
				Now:            req.Now,
			}

			if _, err := tenantbootstrap.CreateTCAAllowlist(ctx, db, scope); err != nil {
				return nil, err
			}

			systemUserID, created, err := tenantbootstrap.EnsureSystemUser(
				ctx,
				db,
				tenantbootstrap.SystemUserParams{Scope: scope, Password: req.SystemUserPassword},
			)
			if err != nil {
				return nil, err
			}

			record := &instancebootstrap.InstanceBootstrap{
				BusinessUnitID: req.BusinessUnitID,
				OrganizationID: req.OrganizationID,
				AdminUserID:    req.AdminUserID,
				Inputs:         req.Inputs,
				CompletedAt:    req.Now,
				CreatedAt:      req.Now,
			}
			if _, err = db.NewInsert().Model(record).Exec(ctx); err != nil {
				r.l.Error("failed to record the instance bootstrap", zap.Error(err))
				return nil, fmt.Errorf("record instance bootstrap: %w", err)
			}

			return &repositories.FinalizeInstanceBootstrapResult{
				Record:            record,
				SystemUserID:      systemUserID,
				SystemUserCreated: created,
			}, nil
		},
	)
}
