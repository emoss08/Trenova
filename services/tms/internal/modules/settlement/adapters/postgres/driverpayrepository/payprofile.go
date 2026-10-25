package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var payProfileErrors = dberror.Mapping{Entity: "PayProfile"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type payProfileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayProfile(p Params) driverpayports.PayProfileRepository {
	return &payProfileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-profile-repository"),
	}
}

func orderedComponents(pp *driverpaydb.PayProfilesTable) keel.Loader {
	c := pp.Components()
	return c.Order(c.Sequence.Asc())
}

func (r *payProfileRepository) List(
	ctx context.Context,
	req *driverpayports.ListPayProfilesRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driverpay.PayProfile], error) {
			pp := driverpaydb.PayProfiles
			q := pp.Where(
				searchPred(req.Filter.Query, pp.Name, pp.Description),
				when(req.Classification != "", pp.Classification.Eq(req.Classification)),
			).Load(orderedComponents(pp)).Order(pp.Name.Asc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list pay profiles: %w", err)
			}

			return result, nil
		},
	)
}

func (r *payProfileRepository) ListConnection(
	ctx context.Context,
	req *driverpayports.ListPayProfileConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driverpay.PayProfile], error) {
			pp := driverpaydb.PayProfiles
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				pp.Query(),
				&listquery.Request[driverpay.PayProfile]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*driverpay.PayProfile],
					) *keel.Query[*driverpay.PayProfile] {
						return q.Load(orderedComponents(pp))
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list pay profiles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *payProfileRepository) SelectOptions(
	ctx context.Context,
	req *driverpayports.PayProfileSelectOptionsRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driverpay.PayProfile], error) {
			pp := driverpaydb.PayProfiles
			q := pp.Pick(pp.ID, pp.Name, pp.Description, pp.Classification, pp.CurrencyCode, pp.CreatedAt).
				Where(
					pp.Status.Eq(domaintypes.StatusActive),
					when(req.Classification != "", pp.Classification.Eq(req.Classification)),
				).
				Order(pp.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(pp.Name.ContainsFold(s), pp.Description.ContainsFold(s))
				},
			)
		},
	)
}

func (r *payProfileRepository) GetByID(
	ctx context.Context,
	req driverpayports.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpay.PayProfile, error) {
		pp := driverpaydb.PayProfiles
		q := pp.Where(pp.ID.Eq(req.ID))
		if req.IncludeComponents {
			q = q.Load(orderedComponents(pp))
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, payProfileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *payProfileRepository) Create(
	ctx context.Context,
	entity *driverpay.PayProfile,
) (*driverpay.PayProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.PayProfile, error) {
		db := r.db.Keel(ctx)
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("dpp_")
		}
		assignComponentFields(entity)

		if err := driverpaydb.PayProfiles.Create(ctx, db, entity); err != nil {
			return nil, payProfileErrors.Map(err, entity.ID.String())
		}
		if err := driverpaydb.PayProfileComponents.CreateMany(
			ctx,
			db,
			entity.Components,
		); err != nil {
			return nil, fmt.Errorf("create pay profile components: %w", err)
		}

		return r.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
			ID:                entity.ID,
			TenantInfo:        tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			IncludeComponents: true,
		})
	})
}

func (r *payProfileRepository) Update(
	ctx context.Context,
	entity *driverpay.PayProfile,
) (*driverpay.PayProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpay.PayProfile, error) {
		db := r.db.Keel(ctx)
		pp := driverpaydb.PayProfiles
		assignComponentFields(entity)

		if err := pp.Save(
			ctx,
			db,
			entity,
			pp.Status,
			pp.Name,
			pp.Description,
			pp.Classification,
			pp.CurrencyCode,
			pp.GuaranteedPeriodMinimumMinor,
			pp.PerDiemRatePerMile,
			pp.PerDiemDailyCapMinor,
		); err != nil {
			return nil, payProfileErrors.Map(err, entity.ID.String())
		}

		if entity.Components != nil {
			c := driverpaydb.PayProfileComponents
			if _, err := c.Where(c.PayProfileID.Eq(entity.ID)).Delete(ctx, db); err != nil {
				return nil, fmt.Errorf("replace pay profile components: %w", err)
			}
			if err := c.CreateMany(ctx, db, entity.Components); err != nil {
				return nil, fmt.Errorf("insert pay profile components: %w", err)
			}
		}

		return r.GetByID(ctx, driverpayports.GetPayProfileByIDRequest{
			ID:                entity.ID,
			TenantInfo:        tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			IncludeComponents: true,
		})
	})
}

func (r *payProfileRepository) CountActiveAssignments(
	ctx context.Context,
	_ pagination.TenantInfo,
	profileID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		wpa := driverpaydb.WorkerPayAssignments
		count, err := wpa.Where(
			wpa.PayProfileID.Eq(profileID),
			openAfterNow(wpa.EffectiveTo),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count active pay assignments: %w", err)
		}

		return int(count), nil
	})
}

type profileCount struct {
	ProfileID pulid.ID
	Total     int64
}

func (r *payProfileRepository) CountActiveAssignmentsByIDs(
	ctx context.Context,
	req driverpayports.CountActivePayAssignmentsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.ProfileIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		wpa := driverpaydb.WorkerPayAssignments
		rows, err := keel.Into[profileCount](
			ctx,
			r.db.Keel(ctx),
			wpa.Where(
				wpa.PayProfileID.In(req.ProfileIDs...),
				openAfterNow(wpa.EffectiveTo),
			).Group(wpa.PayProfileID).Pick(wpa.PayProfileID, keel.Count()),
		)
		if err != nil {
			return nil, fmt.Errorf("count active pay assignments by profile: %w", err)
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.ProfileID] = int(row.Total)
		}

		return counts, nil
	})
}

func assignComponentFields(entity *driverpay.PayProfile) {
	for idx, comp := range entity.Components {
		if comp == nil {
			continue
		}
		comp.OrganizationID = entity.OrganizationID
		comp.BusinessUnitID = entity.BusinessUnitID
		comp.PayProfileID = entity.ID
		comp.Sequence = idx
	}
}
