package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	agreementEntity    = "FactoringAgreement"
	activeOverlapIndex = "ex_factoring_agreements_active_overlap"
)

var (
	agreementErrors = dberror.Mapping{Entity: agreementEntity}
	feeTierErrors   = dberror.Mapping{Entity: "FactoringFeeTier"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type agreementRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAgreementRepository(p Params) factoringports.AgreementRepository {
	return &agreementRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-agreement"),
	}
}

func (r *agreementRepository) Create(
	ctx context.Context,
	entity *factoring.Agreement,
) (*factoring.Agreement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		if err := factoringdb.Agreements.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, translateAgreementError(err, entity.ID)
		}
		if err := r.replaceChildren(ctx, entity); err != nil {
			return nil, err
		}

		return r.load(ctx, entity.ID)
	})
}

func (r *agreementRepository) Update(
	ctx context.Context,
	entity *factoring.Agreement,
) (*factoring.Agreement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		a := factoringdb.Agreements
		if err := a.SaveExcept(ctx, r.db.Keel(ctx), entity, a.FactoringCompanyID); err != nil {
			return nil, translateAgreementError(err, entity.ID)
		}
		if err := r.replaceChildren(ctx, entity); err != nil {
			return nil, err
		}

		return r.load(ctx, entity.ID)
	})
}

func (r *agreementRepository) GetByID(
	ctx context.Context,
	req factoringports.GetAgreementRequest,
) (*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		return r.load(ctx, req.ID)
	})
}

func (r *agreementRepository) ListByFactoringCompany(
	ctx context.Context,
	req factoringports.ListAgreementsRequest,
) ([]*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.Agreement, error) {
		a := factoringdb.Agreements
		rows, err := withChildren().
			Where(a.FactoringCompanyID.Eq(req.FactoringCompanyID)).
			Order(a.EffectiveFrom.Desc(), a.ID.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if rows == nil {
			return []*factoring.Agreement{}, nil
		}

		return rows, nil
	})
}

func (r *agreementRepository) ListByFactoringCompanyIDs(
	ctx context.Context,
	req *factoringports.ListAgreementsByFactoringCompanyIDsRequest,
) (map[pulid.ID][]*factoring.Agreement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*factoring.Agreement, error) {
			result := make(map[pulid.ID][]*factoring.Agreement, len(req.FactoringCompanyIDs))
			if len(req.FactoringCompanyIDs) == 0 {
				return result, nil
			}

			a := factoringdb.Agreements
			rows, err := withChildren().
				Where(a.FactoringCompanyID.In(req.FactoringCompanyIDs...)).
				Order(a.EffectiveFrom.Desc(), a.ID.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				result[row.FactoringCompanyID] = append(result[row.FactoringCompanyID], row)
			}

			return result, nil
		},
	)
}

func (r *agreementRepository) GetInForce(
	ctx context.Context,
	req factoringports.GetAgreementInForceRequest,
) (*factoring.Agreement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Agreement, error) {
		a := factoringdb.Agreements
		entity, err := withChildren().
			Where(inForce(req.At), a.FactoringCompanyID.Eq(req.FactoringCompanyID)).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, agreementErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *agreementRepository) ListInForce(
	ctx context.Context,
	req *factoringports.ListAgreementsInForceRequest,
) (map[pulid.ID]*factoring.Agreement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoring.Agreement, error) {
			result := make(map[pulid.ID]*factoring.Agreement, len(req.FactoringCompanyIDs))
			if len(req.FactoringCompanyIDs) == 0 {
				return result, nil
			}

			a := factoringdb.Agreements
			rows, err := withChildren().
				Where(inForce(req.At), a.FactoringCompanyID.In(req.FactoringCompanyIDs...)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				result[row.FactoringCompanyID] = row
			}

			return result, nil
		},
	)
}

func (r *agreementRepository) load(ctx context.Context, id pulid.ID) (*factoring.Agreement, error) {
	entity, err := withChildren().
		Where(factoringdb.Agreements.ID.Eq(id)).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, agreementErrors.Map(err, id.String())
	}

	return entity, nil
}

func withChildren() *keel.Query[*factoring.Agreement] {
	a := factoringdb.Agreements
	tiers := a.FeeTiers()

	return a.Load(tiers.Order(tiers.DaysFrom.Asc()), a.DocumentTypes())
}

func (r *agreementRepository) replaceChildren(
	ctx context.Context,
	entity *factoring.Agreement,
) error {
	db := r.db.Keel(ctx)
	ft := factoringdb.FeeTiers
	if _, err := ft.Where(ft.AgreementID.Eq(entity.ID)).Delete(ctx, db); err != nil {
		return feeTierErrors.Map(err, entity.ID.String())
	}

	if len(entity.FeeTiers) > 0 {
		tiers := make([]*factoring.FeeTier, 0, len(entity.FeeTiers))
		for _, tier := range factoring.SortedFeeTiers(entity.FeeTiers) {
			tiers = append(tiers, &factoring.FeeTier{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				AgreementID:    entity.ID,
				DaysFrom:       tier.DaysFrom,
				DaysTo:         tier.DaysTo,
				RateBP:         tier.RateBP,
				FlatMinor:      tier.FlatMinor,
			})
		}
		if err := ft.CreateMany(ctx, db, tiers); err != nil {
			return feeTierErrors.Map(err, entity.ID.String())
		}
	}

	err := keel.SetLinks(
		ctx,
		db,
		factoringdb.Agreements.DocumentTypes(),
		entity.ID,
		entity.DocumentTypeIDs(),
	)
	if dberror.IsForeignKeyConstraintViolation(err) {
		return errortypes.NewValidationError(
			"documentTypeIds",
			errortypes.ErrInvalid,
			"One of the required document types does not exist",
		)
	}

	return err
}

func inForce(at int64) keel.Pred {
	a := factoringdb.Agreements
	return keel.And(
		a.Status.Eq(factoring.AgreementStatusActive),
		a.EffectiveFrom.Lte(at),
		keel.Or(a.EffectiveTo.IsNull(), a.EffectiveTo.Gt(at)),
	)
}

func translateAgreementError(err error, id pulid.ID) error {
	if dberror.IsExclusionConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == activeOverlapIndex {
		return errortypes.NewValidationError(
			"effectiveFrom",
			errortypes.ErrInvalid,
			"Another active agreement with this factor covers part of these dates",
		)
	}
	if dberror.IsCheckConstraintViolation(err) {
		return errortypes.NewValidationError(
			"agreement",
			errortypes.ErrInvalid,
			"The agreement terms are not valid",
		)
	}

	return agreementErrors.Map(err, id.String())
}
