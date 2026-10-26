package instancebootstraprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap/instancebootstrapdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
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

func New(p Params) instancebootstrapports.InstanceBootstrapRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.instance-bootstrap-repository"),
	}
}

func (r *repository) GetState(
	ctx context.Context,
) (*instancebootstrapports.InstanceBootstrapState, error) {
	ctx = dbscope.WithSystem(ctx, stateScopeReason)

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*instancebootstrapports.InstanceBootstrapState, error) {
			db := r.db.Keel(ctx)

			state := new(instancebootstrapports.InstanceBootstrapState)
			record, err := instancebootstrapdb.InstanceBootstraps.AllTenants().First(ctx, db)
			switch {
			case err == nil:
				state.Record = record
			case !dberror.IsNotFoundError(err):
				return nil, fmt.Errorf("read instance bootstrap: %w", err)
			}

			u := tenantdb.Users
			count, err := u.AllTenants().
				Where(u.Username.Lower().Ne(tenant.SystemUsername)).
				Count(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("count users: %w", err)
			}
			state.UserCount = int(count)

			return state, nil
		},
	)
}

func (r *repository) Finalize(
	ctx context.Context,
	req *instancebootstrapports.FinalizeInstanceBootstrapRequest,
) (*instancebootstrapports.FinalizeInstanceBootstrapResult, error) {
	ctx = dbscope.WithSystem(ctx, finalizeScopeReason)

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*instancebootstrapports.FinalizeInstanceBootstrapResult, error) {
			scope := tenantbootstrap.Scope{
				OrganizationID: req.OrganizationID,
				BusinessUnitID: req.BusinessUnitID,
				Now:            req.Now,
			}

			systemUserID, created, err := tenantbootstrap.EnsureSystemUser(
				ctx,
				r.db.Keel(ctx),
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
			if err = instancebootstrapdb.InstanceBootstraps.Create(
				ctx,
				r.db.Keel(ctx),
				record,
			); err != nil {
				r.l.Error("failed to record the instance bootstrap", zap.Error(err))
				return nil, fmt.Errorf("record instance bootstrap: %w", err)
			}

			return &instancebootstrapports.FinalizeInstanceBootstrapResult{
				Record:            record,
				SystemUserID:      systemUserID,
				SystemUserCreated: created,
			}, nil
		},
	)
}
