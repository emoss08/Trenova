//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editestcaserepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var testCaseErrors = dberror.Mapping{Entity: "EDITestCase"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDITestCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-test-case-repository"),
	}
}

func (r *repository) ListTestCases(
	ctx context.Context,
	req *ediports.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITestCase], error) {
			tc := edidb.EDITestCases
			q := withDocumentProfile(listedTestCases(req)).Order(tc.CreatedAt.Desc())

			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) ListTestCasesCursor(
	ctx context.Context,
	req *ediports.ListEDITestCasesRequest,
) (*pagination.CursorListResult[*edi.EDITestCase], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDITestCase], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedTestCases(req),
				&listquery.Request[edi.EDITestCase]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withDocumentProfile,
				},
			)
		},
	)
}

func (r *repository) GetTestCaseByID(
	ctx context.Context,
	req ediports.GetEDITestCaseByIDRequest,
) (*edi.EDITestCase, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITestCase, error) {
		entity, err := edidb.EDITestCases.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, testCaseErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITestCase, error) {
		if err := edidb.EDITestCases.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, testCaseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITestCase, error) {
		if err := edidb.EDITestCases.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, testCaseErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) RecordTestCaseRun(
	ctx context.Context,
	req *ediports.RecordEDITestCaseRunRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tc := edidb.EDITestCases
		_, err := tc.Where(tc.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			tc.LastRunAt.To(req.RanAt),
			tc.LastRunPassed.To(req.Passed),
			tc.LastRunWarnings.To(int64(req.Warnings)),
			tc.LastRunErrors.To(int64(req.Errors)),
			tc.Version.ToExpr(tc.Version),
			tc.UpdatedAt.ToExpr(tc.UpdatedAt),
		)

		return testCaseErrors.Map(err, req.ID.String())
	})
}

func (r *repository) DeleteTestCase(
	ctx context.Context,
	req ediports.DeleteEDITestCaseRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tc := edidb.EDITestCases
		n, err := tc.Where(tc.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return testCaseErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "EDITestCase", req.ID.String())
	})
}

func listedTestCases(req *ediports.ListEDITestCasesRequest) *keel.Query[edi.EDITestCase] {
	tc := edidb.EDITestCases
	return tc.Where(keel.When(
		req.PartnerDocumentProfileID.IsNotNil(),
		tc.PartnerDocumentProfileID.Eq(req.PartnerDocumentProfileID),
	))
}

func withDocumentProfile(q *keel.Query[edi.EDITestCase]) *keel.Query[edi.EDITestCase] {
	return q.Load(edidb.EDITestCases.DocumentProfile().Partner())
}
