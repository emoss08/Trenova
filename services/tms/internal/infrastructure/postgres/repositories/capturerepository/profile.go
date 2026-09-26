package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type profileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProfileRepository(p Params) repositories.CaptureProfileRepository {
	return &profileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-profile-repository"),
	}
}

func (r *profileRepository) List(
	ctx context.Context,
	req *repositories.ListCaptureProfilesRequest,
) (*pagination.ListResult[*capture.CaptureProfile], error) {
	entities := make([]*capture.CaptureProfile, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.CaptureProfileColumns

	query := r.db.DBForContext(ctx).NewSelect().Model(&entities)
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := querybuilder.ApplyFilters(
		query,
		buncolgen.CaptureProfileTable.Alias,
		req.Filter,
		(*capture.CaptureProfile)(nil),
	).
		Order(cols.IsDefault.OrderDesc(), cols.Name.OrderAsc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*capture.CaptureProfile]{Items: entities, Total: total}, nil
}

func (r *profileRepository) GetByID(
	ctx context.Context,
	req repositories.GetCaptureProfileByIDRequest,
) (*capture.CaptureProfile, error) {
	entity := new(capture.CaptureProfile)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(buncolgen.CaptureProfileColumns.ID.Eq(), req.ID).
		Apply(buncolgen.CaptureProfileApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture profile")
	}

	return entity, nil
}

func (r *profileRepository) GetDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*capture.CaptureProfile, error) {
	entity := new(capture.CaptureProfile)
	cols := buncolgen.CaptureProfileColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.CaptureProfileApplyTenant(tenantInfo)).
		Where(cols.IsDefault.IsTrue()).
		Where(cols.Status.Eq(), capture.ProfileActive).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Capture profile")
	}

	return entity, nil
}

func (r *profileRepository) Create(
	ctx context.Context,
	entity *capture.CaptureProfile,
) (*capture.CaptureProfile, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if err := r.clearDefault(txCtx, tx, entity); err != nil {
				return err
			}
		}

		_, err := tx.NewInsert().Model(entity).Returning("*").Exec(txCtx)

		return err
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *profileRepository) Update(
	ctx context.Context,
	entity *capture.CaptureProfile,
) (*capture.CaptureProfile, error) {
	ov := entity.Version

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if entity.IsDefault {
			if err := r.clearDefault(txCtx, tx, entity); err != nil {
				return err
			}
		}

		entity.Version = ov + 1
		results, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.CaptureProfileColumns.Version.Eq(), ov).
			Returning("*").
			Exec(txCtx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(results, "Capture profile", entity.ID.String())
	})
	if err != nil {
		entity.Version = ov

		return nil, err
	}

	return entity, nil
}

// clearDefault unsets the flag on every other profile in the tenant. The
// partial unique index would refuse the write otherwise, and failing the
// save because another profile is the default would make "set as default"
// a two-step chore.
func (r *profileRepository) clearDefault(
	ctx context.Context,
	tx bun.Tx,
	entity *capture.CaptureProfile,
) error {
	cols := buncolgen.CaptureProfileColumns
	tenantInfo := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	_, err := tx.NewUpdate().
		Model((*capture.CaptureProfile)(nil)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			q := buncolgen.CaptureProfileScopeTenantUpdate(uq, tenantInfo).
				Where(cols.IsDefault.IsTrue())
			if entity.ID.IsNotNil() {
				q = q.Where(cols.ID.NotEq(), entity.ID)
			}

			return q
		}).
		Set(cols.IsDefault.Set(), false).
		Set(cols.Version.Inc(1)).
		Exec(ctx)

	return err
}

func (r *profileRepository) Delete(
	ctx context.Context,
	req repositories.DeleteCaptureProfileRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*capture.CaptureProfile)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CaptureProfileScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.CaptureProfileColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, "Capture profile", req.ID.String())
}
