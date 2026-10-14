package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	transportEntity      = "FactoringTransport"
	transportFactorIndex = "uq_factoring_transports_factor"
)

type transportRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewTransportRepository(p Params) factoringports.TransportRepository {
	return &transportRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-transport"),
	}
}

func (r *transportRepository) Get(
	ctx context.Context,
	req factoringports.GetTransportRequest,
) (*factoring.Transport, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Transport, error) {
		entity := new(factoring.Transport)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.TransportScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.TransportColumns.FactoringCompanyID.Eq(), req.FactoringCompanyID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, transportEntity)
		}

		return entity, nil
	})
}

func (r *transportRepository) Create(
	ctx context.Context,
	entity *factoring.Transport,
) (*factoring.Transport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Transport, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, translateTransportError(err, "create factoring transport")
		}

		return entity, nil
	})
}

func (r *transportRepository) Update(
	ctx context.Context,
	entity *factoring.Transport,
) (*factoring.Transport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Transport, error) {
		cols := factoringcol.TransportColumns
		previousVersion := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			ExcludeColumn(cols.CreatedAt.Bare(), cols.FactoringCompanyID.Bare()).
			WherePK().
			Where(cols.Version.Eq(), previousVersion).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previousVersion
			return nil, translateTransportError(err, "update factoring transport")
		}
		if err = dberror.CheckRowsAffected(res, transportEntity, entity.ID.String()); err != nil {
			entity.Version = previousVersion
			return nil, err
		}

		return entity, nil
	})
}

func (r *transportRepository) RecordTest(ctx context.Context, entity *factoring.Transport) error {
	_, err := dbtx.Write(ctx, r.db, func(ctx context.Context) (struct{}, error) {
		cols := factoringcol.TransportColumns
		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(cols.LastTestedAt.Bare(), cols.LastTestError.Bare()).
			WherePK().
			Exec(ctx); err != nil {
			return struct{}{}, fmt.Errorf("record factoring transport test: %w", err)
		}
		return struct{}{}, nil
	})
	return err
}

func translateTransportError(err error, action string) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == transportFactorIndex {
		return errortypes.NewConflictError(
			"This factor's delivery settings were just saved by someone else; reload and try again",
		)
	}
	return fmt.Errorf("%s: %w", action, err)
}
