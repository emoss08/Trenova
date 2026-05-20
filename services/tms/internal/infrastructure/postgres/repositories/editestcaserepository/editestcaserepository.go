//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editestcaserepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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
