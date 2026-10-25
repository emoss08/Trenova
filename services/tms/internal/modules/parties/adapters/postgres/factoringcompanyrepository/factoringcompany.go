package factoringcompanyrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxFactorsByEmail = 20

var factoringCompanyErrors = dberror.Mapping{
	Entity: "FactoringCompany",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_factoring_companies_code",
			Field:   "code",
			Message: "A factoring company with this code already exists in your organization",
		},
	},
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) factoringcompanyports.FactoringCompanyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.factoring-company-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *factoringcompanyports.ListFactoringCompanyRequest,
) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				factoringcompanydb.FactoringCompanies.Query(),
				&listquery.Request[factoringcompany.FactoringCompany]{Filter: req.Filter},
			)
			if err != nil {
				r.l.Error("failed to list factoring companies", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *factoringcompanyports.ListFactoringCompanyConnectionRequest,
) (*pagination.CursorListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoringcompany.FactoringCompany], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				factoringcompanydb.FactoringCompanies.Query(),
				&listquery.Request[factoringcompany.FactoringCompany]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.FactoringCompanyColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list factoring company connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req factoringcompanyports.GetFactoringCompaniesByIDsRequest,
) ([]*factoringcompany.FactoringCompany, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringcompany.FactoringCompany, error) {
			if len(req.IDs) == 0 {
				return []*factoringcompany.FactoringCompany{}, nil
			}

			f := factoringcompanydb.FactoringCompanies
			entities, err := f.Where(f.ID.In(req.IDs...)).
				Limit(len(req.IDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("get factoring companies by ids: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			f := factoringcompanydb.FactoringCompanies
			q := f.Where(f.ID.Eq(req.ID))
			if req.IncludeState {
				q = q.Load(f.State())
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get factoring company",
					zap.String("id", req.ID.String()),
					zap.Error(err),
				)
				return nil, factoringCompanyErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			err := factoringcompanydb.FactoringCompanies.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create factoring company", zap.Error(err))
				return nil, factoringCompanyErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *factoringcompany.FactoringCompany,
) (*factoringcompany.FactoringCompany, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompany.FactoringCompany, error) {
			err := factoringcompanydb.FactoringCompanies.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update factoring company",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, factoringCompanyErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *factoringcompanyports.FactoringCompanySelectOptionsRequest,
) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			f := factoringcompanydb.FactoringCompanies
			q := f.Pick(f.ID, f.Code, f.Name, f.City, f.Status).
				Where(f.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(f.Code.ContainsFold(s), f.Name.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) CountAssignedCustomers(
	ctx context.Context,
	req factoringcompanyports.CountAssignedCustomersRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cbp := customerdb.CustomerBillingProfiles
		count, err := cbp.Where(cbp.FactoringCompanyID.Eq(req.FactoringCompanyID)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count customers assigned to a factoring company: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) FindByEmail(
	ctx context.Context,
	req factoringcompanyports.FindFactoringCompaniesByEmailRequest,
) (*factoringcompanyports.FactoringCompaniesByEmail, error) {
	found := &factoringcompanyports.FactoringCompaniesByEmail{
		Exact:    []*factoringcompany.FactoringCompany{},
		ByDomain: []*factoringcompany.FactoringCompany{},
	}
	address := strings.ToLower(strings.TrimSpace(req.Address))
	domain := emailutils.Domain(address)
	if address == "" || domain == "" {
		return found, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringcompanyports.FactoringCompaniesByEmail, error) {
			f := factoringcompanydb.FactoringCompanies
			entities, err := f.Where(
				f.Status.Eq(domaintypes.StatusActive),
				keel.DynFunc("split_part", keel.DynFunc("lower", f.Email), "@", 2).Eq(domain),
			).
				Order(f.ID.Asc()).
				Limit(maxFactorsByEmail).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("find factoring companies by email: %w", err)
			}

			for _, entity := range entities {
				if strings.EqualFold(strings.TrimSpace(entity.Email), address) {
					found.Exact = append(found.Exact, entity)
					continue
				}
				found.ByDomain = append(found.ByDomain, entity)
			}

			return found, nil
		},
	)
}
