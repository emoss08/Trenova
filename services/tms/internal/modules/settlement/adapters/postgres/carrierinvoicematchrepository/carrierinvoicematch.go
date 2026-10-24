package carrierinvoicematchrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var invoiceMatchErrors = dberror.Mapping{Entity: "CarrierInvoiceMatch"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) carriersettlementports.CarrierInvoiceMatchRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-invoice-match-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *carriersettlementports.ListCarrierInvoiceMatchesRequest,
) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error) {
			im := carriersettlementdb.InvoiceMatches
			q := im.Where(
				when(req.Filter.Query != "", im.InvoiceNumber.ContainsFold(req.Filter.Query)),
				when(!req.CarrierID.IsNil(), im.CarrierID.Eq(req.CarrierID)),
				when(req.Status != "", im.Status.Eq(req.Status)),
			).Load(im.Carrier(), im.CarrierAssignment()).Order(im.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list carrier invoice matches: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req carriersettlementports.GetCarrierInvoiceMatchByIDRequest,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
		im := carriersettlementdb.InvoiceMatches
		entity, err := im.Where(im.ID.Eq(req.ID)).
			Load(im.Carrier(), im.CarrierAssignment()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceMatchErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetOpenByEDIInvoiceID(
	ctx context.Context,
	_ pagination.TenantInfo,
	invoiceID pulid.ID,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
		im := carriersettlementdb.InvoiceMatches
		entity, err := im.Where(
			im.EDICarrierInvoiceID.Eq(invoiceID),
			im.Status.Ne(carriersettlement.InvoiceMatchStatusRejected),
		).Order(im.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))

		return optional(entity, err, "get open carrier invoice match")
	})
}

func (r *repository) GetLiveByCarrierInvoiceNumber(
	ctx context.Context,
	req *carriersettlementports.GetLiveCarrierInvoiceMatchByNumberRequest,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
		key := edi.InvoiceNumberKey(req.InvoiceNumber)
		if key == "" {
			return nil, nil //nolint:nilnil // a match without a number never collides
		}

		im := carriersettlementdb.InvoiceMatches
		entity, err := im.Where(
			im.CarrierID.Eq(req.CarrierID),
			im.InvoiceNumberKey.Eq(key),
			im.DuplicateOfMatchID.IsNull(),
			im.Status.Ne(carriersettlement.InvoiceMatchStatusRejected),
		).Order(im.CreatedAt.Asc()).First(ctx, r.db.Keel(ctx))

		return optional(entity, err, "get live carrier invoice match by number")
	})
}

func (r *repository) ListLiveByAssignment(
	ctx context.Context,
	req carriersettlementports.ListLiveCarrierInvoiceMatchesByAssignmentRequest,
) ([]*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carriersettlement.InvoiceMatch, error) {
			im := carriersettlementdb.InvoiceMatches
			entities, err := im.Where(
				im.CarrierAssignmentID.Eq(req.AssignmentID),
				im.Status.Ne(carriersettlement.InvoiceMatchStatusRejected),
			).Order(im.CreatedAt.Asc(), im.ID.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list live carrier invoice matches by assignment: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *repository) ListLiveByMoveIDs(
	ctx context.Context,
	req carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest,
) (map[pulid.ID][]*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*carriersettlement.InvoiceMatch, error) {
			if len(req.MoveIDs) == 0 {
				return map[pulid.ID][]*carriersettlement.InvoiceMatch{}, nil
			}

			im := carriersettlementdb.InvoiceMatches
			ca := im.CarrierAssignment()
			matches, err := im.Where(
				ca.ShipmentMoveID.In(req.MoveIDs...),
				im.Status.Ne(carriersettlement.InvoiceMatchStatusRejected),
			).
				Load(ca.Pick(ca.ShipmentMoveID)).
				Order(im.CreatedAt.Desc(), im.ID.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list live carrier invoice matches by move: %w", err)
			}

			grouped := make(map[pulid.ID][]*carriersettlement.InvoiceMatch, len(req.MoveIDs))
			for _, match := range matches {
				if match.CarrierAssignment == nil {
					continue
				}
				moveID := match.CarrierAssignment.ShipmentMoveID
				match.CarrierAssignment = nil
				grouped[moveID] = append(grouped[moveID], match)
			}

			return grouped, nil
		},
	)
}

func (r *repository) ListResolvedAssignmentIDs(
	ctx context.Context,
	req carriersettlementports.ListResolvedMatchAssignmentsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		if len(req.AssignmentIDs) == 0 {
			return []pulid.ID{}, nil
		}

		im := carriersettlementdb.InvoiceMatches
		ids, err := keel.Into[pulid.ID](
			ctx,
			r.db.Keel(ctx),
			im.Pick(im.CarrierAssignmentID).
				Where(
					im.CarrierAssignmentID.In(req.AssignmentIDs...),
					im.Status.Eq(carriersettlement.InvoiceMatchStatusResolved),
				).
				Group(im.CarrierAssignmentID),
		)
		if err != nil {
			return nil, fmt.Errorf("list resolved carrier invoice match assignments: %w", err)
		}

		return ids, nil
	})
}

func (r *repository) GetOpenByExtractionID(
	ctx context.Context,
	_ pagination.TenantInfo,
	extractionID pulid.ID,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
		im := carriersettlementdb.InvoiceMatches
		entity, err := im.Where(
			im.DocumentAIExtractionID.Eq(extractionID),
			im.Status.Ne(carriersettlement.InvoiceMatchStatusRejected),
		).Order(im.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))

		return optional(entity, err, "get open carrier invoice match by extraction")
	})
}

func (r *repository) UpdateSettlementForAdjustmentEvents(
	ctx context.Context,
	_ pagination.TenantInfo,
	costEventIDs []pulid.ID,
	settlementID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(costEventIDs) == 0 {
			return nil
		}

		im := carriersettlementdb.InvoiceMatches
		_, err := im.Where(
			im.AdjustmentCostEventID.In(costEventIDs...),
			im.CarrierSettlementID.IsNull(),
		).Update(ctx, r.db.Keel(ctx), im.CarrierSettlementID.To(settlementID))
		if err != nil {
			return fmt.Errorf("update carrier invoice match settlements: %w", err)
		}

		return nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *carriersettlement.InvoiceMatch,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
			err := carriersettlementdb.InvoiceMatches.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, fmt.Errorf(
					"create carrier invoice match: %w",
					invoiceMatchErrors.Map(err, entity.ID.String()),
				)
			}

			return r.GetByID(ctx, carriersettlementports.GetCarrierInvoiceMatchByIDRequest{
				ID: entity.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			})
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *carriersettlement.InvoiceMatch,
) (*carriersettlement.InvoiceMatch, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlement.InvoiceMatch, error) {
			im := carriersettlementdb.InvoiceMatches
			if err := im.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				im.Status,
				im.CarrierSettlementID,
				im.AdjustmentCostEventID,
				im.InvoiceTotalMinor,
				im.ExpectedTotalMinor,
				im.VarianceMinor,
				im.ResolutionNote,
				im.ResolvedByID,
				im.ResolvedAt,
			); err != nil {
				return nil, invoiceMatchErrors.Map(err, entity.ID.String())
			}

			return r.GetByID(ctx, carriersettlementports.GetCarrierInvoiceMatchByIDRequest{
				ID: entity.ID,
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			})
		},
	)
}

func when(include bool, p keel.Pred) keel.Pred {
	if !include {
		return nil
	}

	return p
}

func optional(
	entity *carriersettlement.InvoiceMatch,
	err error,
	operation string,
) (*carriersettlement.InvoiceMatch, error) {
	if dberror.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // nil match represents an optional absence
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}

	return entity, nil
}
