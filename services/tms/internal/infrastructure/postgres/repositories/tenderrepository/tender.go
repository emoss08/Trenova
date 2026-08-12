package tenderrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

const (
	defaultSweepLimit      = 100
	defaultTokenPurgeLimit = 500
)

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TenderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tender-repository"),
	}
}

func (r *repository) offersRelation(q *bun.SelectQuery) *bun.SelectQuery {
	q = q.Relation("Offers", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Order(buncolgen.TenderOfferColumns.Rank.OrderAsc())
	})
	return q.Relation("Offers.Carrier")
}

func (r *repository) Create(ctx context.Context, entity *tender.Tender) (*tender.Tender, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("shipmentMoveId", entity.ShipmentMoveID.String()),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create tender", zap.Error(err))
		return nil, err
	}

	for _, offer := range entity.Offers {
		offer.TenderID = entity.ID
		offer.OrganizationID = entity.OrganizationID
		offer.BusinessUnitID = entity.BusinessUnitID
	}

	if len(entity.Offers) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Offers).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to create tender offers", zap.Error(err))
			return nil, err
		}
	}

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetTenderByIDRequest,
) (*tender.Tender, error) {
	cols := buncolgen.TenderColumns
	entity := new(tender.Tender)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.IncludeOffers {
				sq = r.offersRelation(sq)
			}
			return sq
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.TenderID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get tender", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Tender")
	}

	return entity, nil
}

func (r *repository) GetLiveByMoveID(
	ctx context.Context,
	req repositories.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	cols := buncolgen.TenderColumns
	entity := new(tender.Tender)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(r.offersRelation).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentMoveID.Eq(), req.MoveID).
				Where(cols.Status.In(), bun.List([]tender.Status{
					tender.StatusActive,
					tender.StatusNeedsReview,
				}))
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil tender represents an optional absence
		}
		r.l.Error("failed to get live tender by move", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListLiveByMoveIDs(
	ctx context.Context,
	req *repositories.ListLiveTendersByMovesRequest,
) ([]*tender.Tender, error) {
	if len(req.MoveIDs) == 0 {
		return []*tender.Tender{}, nil
	}

	cols := buncolgen.TenderColumns
	entities := make([]*tender.Tender, 0, len(req.MoveIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(r.offersRelation).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentMoveID.In(), bun.List(req.MoveIDs)).
				Where(cols.Status.In(), bun.List([]tender.Status{
					tender.StatusActive,
					tender.StatusNeedsReview,
				}))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list live tenders by moves", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetOfferByID(
	ctx context.Context,
	req repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	cols := buncolgen.TenderOfferColumns
	entity := new(tender.TenderOffer)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Carrier").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			if req.IncludeTender {
				sq = sq.Relation("Tender")
			}
			return sq
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderOfferScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.OfferID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get tender offer", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Tender offer")
	}

	return entity, nil
}

// GetAcceptedOfferForPartnerReference resolves a late-arriving carrier 214/210
// to the accepted offer it belongs to. It anchors on OFFER status rather than
// tender status because the tender row may have moved on by the time the
// carrier's status or invoice document arrives.
func (r *repository) GetAcceptedOfferForPartnerReference(
	ctx context.Context,
	req repositories.GetAcceptedOfferForPartnerReferenceRequest,
) (*tender.TenderOffer, error) {
	if req.OfferID.IsNil() && len(req.References) == 0 {
		return nil, errortypes.NewNotFoundError("Tender offer")
	}

	offerCols := buncolgen.TenderOfferColumns
	tenderCols := buncolgen.TenderColumns
	shipmentCols := buncolgen.ShipmentColumns
	entity := new(tender.TenderOffer)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.TenderOfferRelations.Tender).
		Relation(buncolgen.TenderOfferRelations.Carrier).
		Join("JOIN "+buncolgen.TenderTable.Name+" AS "+buncolgen.TenderTable.Alias+
			" ON "+tenderCols.ID.EqColumn(offerCols.TenderID)).
		Join("JOIN "+buncolgen.ShipmentTable.Name+" AS "+buncolgen.ShipmentTable.Alias+
			" ON "+shipmentCols.ID.EqColumn(tenderCols.ShipmentID)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.TenderOfferScopeTenant(sq, req.TenantInfo)
			sq = buncolgen.TenderScopeTenant(sq, req.TenantInfo)
			sq = buncolgen.ShipmentScopeTenant(sq, req.TenantInfo)
			return sq.
				Where(offerCols.EDIPartnerID.Eq(), req.PartnerID).
				Where(offerCols.Status.Eq(), tender.OfferStatusAccepted).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					if !req.OfferID.IsNil() {
						inner = inner.WhereOr(offerCols.ID.Eq(), req.OfferID)
					}
					if len(req.References) > 0 {
						references := bun.List(req.References)
						inner = inner.
							WhereOr(shipmentCols.ID.In(), references).
							WhereOr(shipmentCols.ProNumber.In(), references).
							WhereOr(shipmentCols.BOL.In(), references)
					}
					return inner
				})
		}).
		OrderExpr(offerCols.RespondedAt.Qualified() + " DESC NULLS LAST").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if !dberror.IsNotFoundError(err) {
			r.l.Error("failed to get accepted tender offer for partner reference", zap.Error(err))
		}
		return nil, dberror.HandleNotFoundError(err, "Tender offer")
	}

	return entity, nil
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req repositories.ListTendersByShipmentRequest,
) ([]*tender.Tender, error) {
	cols := buncolgen.TenderColumns
	entities := make([]*tender.Tender, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(r.offersRelation).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentID.Eq(), req.ShipmentID)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list tenders by shipment", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// ListForSweep finds live tenders whose current Sent offer has been expired
// for longer than the grace window, so the safety-net sweep can nudge or
// recover their workflows.
func (r *repository) ListForSweep(
	ctx context.Context,
	req repositories.ListTendersForSweepRequest,
) ([]*tender.Tender, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultSweepLimit
	}

	entities := make([]*tender.Tender, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(r.offersRelation).
		Where(buncolgen.TenderColumns.Status.Eq(), tender.StatusActive).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			// Arm 1: the current offer is overdue. Arm 2: no offer is in
			// flight at all past the grace window — a workflow that died
			// before its first dispatch, or after its last offer resolved
			// without reaching a terminal tender status. Nudging a healthy
			// workflow that is merely between offers is harmless.
			return sq.Where(
				"EXISTS (SELECT 1 FROM tender_offers tof WHERE tof.tender_id = tnd.id "+
					"AND tof.organization_id = tnd.organization_id "+
					"AND tof.business_unit_id = tnd.business_unit_id "+
					"AND tof.status = ? AND tof.expires_at <= ?)",
				tender.OfferStatusSent,
				req.ExpiredBy,
			).WhereOr(
				"NOT EXISTS (SELECT 1 FROM tender_offers tof WHERE tof.tender_id = tnd.id "+
					"AND tof.organization_id = tnd.organization_id "+
					"AND tof.business_unit_id = tnd.business_unit_id "+
					"AND tof.status = ?) AND tnd.created_at <= ?",
				tender.OfferStatusSent,
				req.ExpiredBy,
			)
		}).
		Limit(limit)

	if req.TenantInfo != nil {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.TenderScopeTenant(sq, *req.TenantInfo)
		})
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list tenders for sweep", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) SetWorkflowID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tenderID pulid.ID,
	workflowID string,
) error {
	cols := buncolgen.TenderColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.Tender)(nil)).
		Set("workflow_id = ?", workflowID).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.Eq(), tenderID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to set tender workflow id", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Tender", tenderID.String())
}

// UpdateTenderStatus performs a compare-and-set transition. It returns false
// (with no error) when the tender was not in one of the expected statuses,
// which callers treat as "another actor already moved it".
func (r *repository) UpdateTenderStatus(
	ctx context.Context,
	req *repositories.UpdateTenderStatusRequest,
) (bool, error) {
	cols := buncolgen.TenderColumns
	now := timeutils.NowUnix()

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.Tender)(nil)).
		Set("status = ?", req.ToStatus).
		Set("version = version + 1").
		Set("updated_at = ?", now)

	if req.CurrentRank != nil {
		q = q.Set("current_rank = ?", *req.CurrentRank)
	}
	if req.CanceledByID != nil {
		q = q.Set("canceled_by_id = ?", *req.CanceledByID)
	}
	if req.CancellationReason != "" {
		q = q.Set("cancellation_reason = ?", req.CancellationReason)
	}
	if req.AcceptedOfferID != nil {
		q = q.Set("accepted_offer_id = ?", *req.AcceptedOfferID)
	}
	if req.AcceptedAt != nil {
		q = q.Set("accepted_at = ?", *req.AcceptedAt)
	}
	if req.ExhaustedAt != nil {
		q = q.Set("exhausted_at = ?", *req.ExhaustedAt)
	}
	if req.CanceledAt != nil {
		q = q.Set("canceled_at = ?", *req.CanceledAt)
	}

	results, err := q.
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.TenderID).
				Where(cols.Status.In(), bun.List(req.FromStatus))
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update tender status", zap.Error(err))
		return false, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// UpdateOfferStatus performs a compare-and-set transition on one offer,
// returning false when the offer had already left the expected status.
func (r *repository) UpdateOfferStatus(
	ctx context.Context,
	req *repositories.UpdateOfferStatusRequest,
) (bool, error) {
	cols := buncolgen.TenderOfferColumns
	now := timeutils.NowUnix()

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.TenderOffer)(nil)).
		Set("status = ?", req.ToStatus).
		Set("version = version + 1").
		Set("updated_at = ?", now)

	if req.SentAt != nil {
		q = q.Set("sent_at = ?", *req.SentAt)
	}
	if req.ExpiresAt != nil {
		q = q.Set("expires_at = ?", *req.ExpiresAt)
	}
	if req.RespondedAt != nil {
		q = q.Set("responded_at = ?", *req.RespondedAt)
	}
	if req.ResponseSource != "" {
		q = q.Set("response_source = ?", req.ResponseSource)
	}
	if req.DeclineReason != "" {
		q = q.Set("decline_reason = ?", req.DeclineReason)
	}
	if req.DeliveryError != "" {
		q = q.Set("delivery_error = ?", req.DeliveryError)
	}
	if req.EDIMessageID != nil {
		q = q.Set("edi_message_id = ?", *req.EDIMessageID)
	}

	results, err := q.
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderOfferScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.OfferID).
				Where(cols.Status.In(), bun.List(req.FromStatus))
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update tender offer status", zap.Error(err))
		return false, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *repository) BulkUpdateOfferStatus(
	ctx context.Context,
	req *repositories.BulkOfferStatusRequest,
) (int64, error) {
	cols := buncolgen.TenderOfferColumns

	q := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.TenderOffer)(nil)).
		Set("status = ?", req.ToStatus).
		Set("version = version + 1").
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.TenderOfferScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.TenderID.Eq(), req.TenderID).
				Where(cols.Status.In(), bun.List(req.FromStatus))
			if !req.ExceptID.IsNil() {
				uq = uq.Where(cols.ID.Ne(), req.ExceptID)
			}
			return uq
		})

	results, err := q.Exec(ctx)
	if err != nil {
		r.l.Error("failed to bulk update tender offer status", zap.Error(err))
		return 0, err
	}

	return results.RowsAffected()
}

// RecordLateOfferResponse appends a late response to an already-terminal
// offer without touching its status.
func (r *repository) RecordLateOfferResponse(
	ctx context.Context,
	req *repositories.RecordLateOfferResponseRequest,
) error {
	cols := buncolgen.TenderOfferColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.TenderOffer)(nil)).
		Set("late_response_action = ?", req.Action).
		Set("late_response_at = ?", req.OccurredAt).
		Set("response_source = ?", req.Source).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderOfferScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.OfferID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to record late tender offer response", zap.Error(err))
	}
	return err
}

func (r *repository) CreateToken(
	ctx context.Context,
	entity *tender.TenderOfferToken,
) (*tender.TenderOfferToken, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create tender offer token", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// GetTokenByHash resolves a public token. It is deliberately not
// tenant-scoped: the caller only holds the opaque token, and the row itself
// carries the tenant that every follow-up query is scoped to.
func (r *repository) GetTokenByHash(
	ctx context.Context,
	tokenHash string,
) (*tender.TenderOfferToken, error) {
	cols := buncolgen.TenderOfferTokenColumns
	entity := new(tender.TenderOfferToken)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.TokenHash.Eq(), tokenHash).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil token represents an optional absence
		}
		r.l.Error("failed to get tender offer token", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// MarkTokenUsed is the single-use gate for public response links: the
// conditional update only wins for the first caller.
func (r *repository) MarkTokenUsed(
	ctx context.Context,
	req *repositories.MarkTokenUsedRequest,
) (bool, error) {
	cols := buncolgen.TenderOfferTokenColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.TenderOfferToken)(nil)).
		Set("used_at = ?", req.UsedAt).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderOfferTokenScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.TokenID).
				Where(cols.UsedAt.IsNull()).
				Where(cols.RevokedAt.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark tender offer token used", zap.Error(err))
		return false, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *repository) RevokeTokensForOffer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
	revokedAt int64,
) error {
	cols := buncolgen.TenderOfferTokenColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tender.TenderOfferToken)(nil)).
		Set("revoked_at = ?", revokedAt).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.TenderOfferTokenScopeTenantUpdate(uq, tenantInfo).
				Where(cols.TenderOfferID.Eq(), offerID).
				Where(cols.RevokedAt.IsNull()).
				Where(cols.UsedAt.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to revoke tender offer tokens", zap.Error(err))
	}
	return err
}

// ListAcceptedMissingRateConfirmation is the recovery sweep's cross-tenant
// read: accepted tenders whose carrier is already covering the move but whose
// rate confirmation never landed. The NOT EXISTS arm deliberately treats any
// non-voided revision as covered, so a dispatcher-managed agreement is never
// re-entered, and the assignment arm excludes the needs-review family because
// those moves have no matching-carrier assignment at all.
func (r *repository) ListAcceptedMissingRateConfirmation(
	ctx context.Context,
	req repositories.ListAcceptedMissingRateConfirmationRequest,
) ([]*tender.Tender, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultSweepLimit
	}

	tenderCols := buncolgen.TenderColumns
	offerCols := buncolgen.TenderOfferColumns
	assignCols := buncolgen.CarrierAssignmentColumns
	rateConCols := buncolgen.RateConfirmationColumns
	dba := r.db.DBForContext(ctx)

	acceptedOfferCarrier := dba.NewSelect().
		Model((*tender.TenderOffer)(nil)).
		ColumnExpr(offerCols.CarrierID.Qualified()).
		Where(offerCols.ID.EqColumn(tenderCols.AcceptedOfferID)).
		Where(offerCols.OrganizationID.EqColumn(tenderCols.OrganizationID)).
		Where(offerCols.BusinessUnitID.EqColumn(tenderCols.BusinessUnitID))

	standingRateConfirmation := dba.NewSelect().
		Model((*rateconfirmation.RateConfirmation)(nil)).
		ColumnExpr("1").
		Where(rateConCols.CarrierAssignmentID.EqColumn(assignCols.ID)).
		Where(rateConCols.OrganizationID.EqColumn(assignCols.OrganizationID)).
		Where(rateConCols.BusinessUnitID.EqColumn(assignCols.BusinessUnitID)).
		Where(rateConCols.Status.NotEq(), rateconfirmation.StatusVoided)

	coveringAssignment := dba.NewSelect().
		Model((*shipment.CarrierAssignment)(nil)).
		ColumnExpr("1").
		Where(assignCols.ShipmentMoveID.EqColumn(tenderCols.ShipmentMoveID)).
		Where(assignCols.OrganizationID.EqColumn(tenderCols.OrganizationID)).
		Where(assignCols.BusinessUnitID.EqColumn(tenderCols.BusinessUnitID)).
		Where(assignCols.Status.NotEq(), shipment.CarrierAssignmentStatusCanceled).
		Where(assignCols.CarrierID.Expr("{} = (?)"), acceptedOfferCarrier).
		Where("NOT EXISTS (?)", standingRateConfirmation)

	entities := make([]*tender.Tender, 0, limit)
	err := dba.NewSelect().
		Model(&entities).
		Where(tenderCols.Status.Eq(), tender.StatusAccepted).
		Where(tenderCols.AcceptedOfferID.IsNotNull()).
		Where(tenderCols.AcceptedAt.Gt(), req.AcceptedAfter).
		Where(tenderCols.AcceptedAt.Lte(), req.AcceptedBefore).
		Where("EXISTS (?)", coveringAssignment).
		Order(tenderCols.AcceptedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list accepted tenders missing a rate confirmation", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// PurgeDeadOfferTokens deletes offer links that can no longer authorize
// anything — used, revoked, or expired — once the retention window past their
// terminal moment has elapsed. A live link always coalesces to a future
// expires_at, so it can never match.
func (r *repository) PurgeDeadOfferTokens(
	ctx context.Context,
	req repositories.PurgeDeadTokensRequest,
) (int64, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultTokenPurgeLimit
	}

	cols := buncolgen.TenderOfferTokenColumns
	deadAt := buncolgen.Expr(
		"COALESCE({0}, {1}, {2})", cols.UsedAt, cols.RevokedAt, cols.ExpiresAt,
	)
	dba := r.db.DBForContext(ctx)

	dead := dba.NewSelect().
		Model((*tender.TenderOfferToken)(nil)).
		Column(cols.ID.Bare()).
		Where(deadAt+" < ?", req.DeadBefore).
		Limit(limit)

	result, err := dba.NewDelete().
		Model((*tender.TenderOfferToken)(nil)).
		Where(cols.ID.Expr("{} IN (?)"), dead).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge dead tender offer tokens", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}
