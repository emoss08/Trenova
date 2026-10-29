package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	transportEntity      = "FactoringTransport"
	transportFactorIndex = "uq_factoring_transports_factor"
)

var transportErrors = dberror.Mapping{Entity: transportEntity}

type transportRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTransportRepository(p Params) factoringsubmissionports.TransportRepository {
	return &transportRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-transport"),
	}
}

func (r *transportRepository) Get(
	ctx context.Context,
	req factoringsubmissionports.GetTransportRequest,
) (*factoringsubmission.Transport, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringsubmission.Transport, error) {
		t := factoringsubmissiondb.Transports
		entity, err := t.Where(t.FactoringCompanyID.Eq(req.FactoringCompanyID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transportErrors.Map(err, req.FactoringCompanyID.String())
		}

		return entity, nil
	})
}

func (r *transportRepository) Create(
	ctx context.Context,
	entity *factoringsubmission.Transport,
) (*factoringsubmission.Transport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoringsubmission.Transport, error) {
		if err := factoringsubmissiondb.Transports.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, translateTransportError(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *transportRepository) Update(
	ctx context.Context,
	entity *factoringsubmission.Transport,
) (*factoringsubmission.Transport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoringsubmission.Transport, error) {
		t := factoringsubmissiondb.Transports
		if err := t.SaveExcept(ctx, r.db.Keel(ctx), entity, t.FactoringCompanyID); err != nil {
			return nil, translateTransportError(err, entity.ID)
		}

		return entity, nil
	})
}

func (r *transportRepository) RecordTest(
	ctx context.Context,
	entity *factoringsubmission.Transport,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := factoringsubmissiondb.Transports
		_, err := t.Where(t.ID.Eq(entity.ID)).Update(ctx, r.db.Keel(ctx),
			t.LastTestedAt.ToExpr(keel.Val(entity.LastTestedAt)),
			t.LastTestError.ToExpr(keel.Val(stringutils.NilIfEmpty(entity.LastTestError))),
			t.Version.ToExpr(t.Version),
		)

		return transportErrors.Map(err, entity.ID.String())
	})
}

func translateTransportError(err error, id pulid.ID) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == transportFactorIndex {
		return errortypes.NewConflictError(
			"This factor's delivery settings were just saved by someone else; reload and try again",
		)
	}

	return transportErrors.Map(err, id.String())
}
