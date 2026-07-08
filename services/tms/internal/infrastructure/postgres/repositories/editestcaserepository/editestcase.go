//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editestcaserepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.EDITestCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-test-case-repository"),
	}
}

func (r *repository) ListTestCases(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	entities := make([]*edi.EDITestCase, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDITestCaseColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("DocumentProfile").
		Relation("DocumentProfile.Partner").
		Apply(buncolgen.EDITestCaseApplyTenant(req.Filter.TenantInfo))
	if !req.PartnerDocumentProfileID.IsNil() {
		query = query.Where(cols.PartnerDocumentProfileID.Eq(), req.PartnerDocumentProfileID)
	}
	total, err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDITestCase]{Items: entities, Total: total}, nil
}

func (r *repository) GetTestCaseByID(
	ctx context.Context,
	req repositories.GetEDITestCaseByIDRequest,
) (*edi.EDITestCase, error) {
	entity := new(edi.EDITestCase)
	cols := buncolgen.EDITestCaseColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDITestCaseApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITestCase")
	}
	return entity, nil
}

func (r *repository) CreateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDITestCase", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) RecordTestCaseRun(
	ctx context.Context,
	req *repositories.RecordEDITestCaseRunRequest,
) error {
	cols := buncolgen.EDITestCaseColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.EDITestCase)(nil)).
		Set("last_run_at = ?", req.RanAt).
		Set("last_run_passed = ?", req.Passed).
		Set("last_run_warnings = ?", req.Warnings).
		Set("last_run_errors = ?", req.Errors).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Exec(ctx)
	return err
}

func (r *repository) DeleteTestCase(
	ctx context.Context,
	req repositories.DeleteEDITestCaseRequest,
) error {
	cols := buncolgen.EDITestCaseColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*edi.EDITestCase)(nil)).
		Where(cols.ID.Eq(), req.ID).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(results, "EDITestCase", req.ID.String())
}

func (r *repository) ListTestCasesCursor(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.CursorListResult[*edi.EDITestCase], error) {
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.EDITestCaseColumns
	extraFilters := func(sq *bun.SelectQuery) *bun.SelectQuery {
		if !req.PartnerDocumentProfileID.IsNil() {
			sq = sq.Where(cols.PartnerDocumentProfileID.Eq(), req.PartnerDocumentProfileID)
		}
		return sq
	}

	total, err := dba.
		NewSelect().
		Model((*edi.EDITestCase)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFiltersWithoutSort(
				sq,
				"etc",
				req.Filter,
				(*edi.EDITestCase)(nil),
			)
			return extraFilters(sq)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDITestCase]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*edi.EDITestCase) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				Relation("DocumentProfile").
				Relation("DocumentProfile.Partner")
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				"etc",
				req.Filter,
				req.Cursor,
				(*edi.EDITestCase)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return extraFilters(sq), nil
		},
	})
}
