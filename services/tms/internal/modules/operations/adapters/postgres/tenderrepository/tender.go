package tenderrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

var tenderErrors = dberror.Mapping{
	Entity: "Tender",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_tenders_live_move",
			Field:   "shipmentMoveId",
			Message: "Move already has a live tender",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_tenders_shipment",
			Field:   "shipmentId",
			Message: "Shipment does not exist in your organization",
		},
		{
			Name:    "fk_tenders_shipment_move",
			Field:   "shipmentMoveId",
			Message: "Move does not exist in your organization",
		},
	},
}

var offerErrors = dberror.Mapping{
	Entity: "Tender offer",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_tender_offers_carrier",
			Field:   "carrierId",
			Message: "Carrier does not exist in your organization",
		},
		{
			Name:    "fk_tender_offers_edi_partner",
			Field:   "ediPartnerId",
			Message: "EDI partner does not exist in your organization",
		},
	},
}

var tokenErrors = dberror.Mapping{
	Entity: "Tender offer token",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_tender_offer_tokens_hash",
			Field:   "tokenHash",
			Message: "Tender offer token already exists",
		},
	},
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenderports.TenderRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tender-repository"),
	}
}

var awaitingAnswerStatuses = []tender.OfferStatus{
	tender.OfferStatusSent,
	tender.OfferStatusCountered,
}

var liveStatuses = []tender.Status{tender.StatusActive, tender.StatusNeedsReview}

func rankedOffers() keel.Loader {
	o := tenderdb.Tenders.Offers()
	return o.Order(o.Rank.Asc()).Load(o.Carrier())
}

func stampOffers(offers []*tender.TenderOffer, tenderID pulid.ID, tenant pagination.TenantInfo) {
	for _, offer := range offers {
		offer.TenderID = tenderID
		offer.OrganizationID = tenant.OrgID
		offer.BusinessUnitID = tenant.BuID
	}
}

func (r *repository) Create(ctx context.Context, entity *tender.Tender) (*tender.Tender, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tender.Tender, error) {
		log := r.l.With(
			zap.String("operation", "Create"),
			zap.String("shipmentMoveId", entity.ShipmentMoveID.String()),
		)

		db := r.db.Keel(ctx)
		if err := tenderdb.Tenders.Create(ctx, db, entity); err != nil {
			log.Error("failed to create tender", zap.Error(err))
			return nil, tenderErrors.Map(err, entity.ID.String())
		}

		stampOffers(entity.Offers, entity.ID, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
		if err := tenderdb.TenderOffers.CreateMany(ctx, db, entity.Offers); err != nil {
			log.Error("failed to create tender offers", zap.Error(err))
			return nil, offerErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req tenderports.GetTenderByIDRequest,
) (*tender.Tender, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.Tender, error) {
		t := tenderdb.Tenders
		q := t.Where(t.ID.Eq(req.TenderID))
		if req.IncludeOffers {
			q = q.Load(rankedOffers())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get tender", zap.Error(err))
			return nil, tenderErrors.Map(err, req.TenderID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetLiveByMoveID(
	ctx context.Context,
	req tenderports.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.Tender, error) {
		t := tenderdb.Tenders
		entity, err := t.Where(t.ShipmentMoveID.Eq(req.MoveID), t.Status.In(liveStatuses...)).
			Load(rankedOffers()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil tender represents an optional absence
			}
			r.l.Error("failed to get live tender by move", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListLiveByMoveIDs(
	ctx context.Context,
	req *tenderports.ListLiveTendersByMovesRequest,
) ([]*tender.Tender, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tender.Tender, error) {
		if len(req.MoveIDs) == 0 {
			return []*tender.Tender{}, nil
		}

		t := tenderdb.Tenders
		entities, err := t.Where(t.ShipmentMoveID.In(req.MoveIDs...), t.Status.In(liveStatuses...)).
			Load(rankedOffers()).
			Limit(len(req.MoveIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list live tenders by moves", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetOfferByID(
	ctx context.Context,
	req tenderports.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.TenderOffer, error) {
		o := tenderdb.TenderOffers
		q := o.Where(o.ID.Eq(req.OfferID)).Load(o.Carrier())
		if req.IncludeTender {
			q = q.Load(o.Tender())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get tender offer", zap.Error(err))
			return nil, offerErrors.Map(err, req.OfferID.String())
		}

		return entity, nil
	})
}

// GetAcceptedOfferForPartnerReference resolves a late-arriving carrier 214/210
// to the accepted offer it belongs to. It anchors on OFFER status rather than
// tender status because the tender row may have moved on by the time the
// carrier's status or invoice document arrives.
func (r *repository) GetAcceptedOfferForPartnerReference(
	ctx context.Context,
	req tenderports.GetAcceptedOfferForPartnerReferenceRequest,
) (*tender.TenderOffer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.TenderOffer, error) {
		if req.OfferID.IsNil() && len(req.References) == 0 {
			return nil, errortypes.NewNotFoundError("Tender offer")
		}

		o := tenderdb.TenderOffers
		sp := o.Tender().Shipment()
		references := make([]pulid.ID, len(req.References))
		for i, reference := range req.References {
			references[i] = pulid.ID(reference)
		}

		var match []keel.Pred
		if !req.OfferID.IsNil() {
			match = append(match, o.ID.Eq(req.OfferID))
		}
		if len(req.References) > 0 {
			match = append(match,
				sp.ID.In(references...),
				sp.ProNumber.In(req.References...),
				sp.BOL.In(req.References...),
			)
		}

		entity, err := o.Where(
			o.EDIPartnerID.Eq(req.PartnerID),
			o.Status.Eq(tender.OfferStatusAccepted),
			sp.ID.IsNotNull(),
			keel.Or(match...),
		).
			Load(o.Tender(), o.Carrier()).
			Order(o.RespondedAt.Desc().NullsLast()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if !dberror.IsNotFoundError(err) {
				r.l.Error(
					"failed to get accepted tender offer for partner reference",
					zap.Error(err),
				)
			}
			return nil, offerErrors.Map(err, req.OfferID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req tenderports.ListTendersByShipmentRequest,
) ([]*tender.Tender, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tender.Tender, error) {
		t := tenderdb.Tenders
		entities, err := t.Where(
			t.ShipmentID.Eq(req.ShipmentID),
			keel.When(req.ShipmentMoveID.IsNotNil(), t.ShipmentMoveID.Eq(req.ShipmentMoveID)),
		).
			Load(rankedOffers()).
			Order(t.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list tenders by shipment", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// ListForSweep finds live tenders whose current Sent offer has been expired
// for longer than the grace window, so the safety-net sweep can nudge or
// recover their workflows.
func (r *repository) ListForSweep(
	ctx context.Context,
	req tenderports.ListTendersForSweepRequest,
) ([]*tender.Tender, error) {
	system := req.TenantInfo == nil
	if system {
		ctx = dbscope.WithSystem(
			ctx,
			"list stalled tenders across every organization for the tender sweep",
		)
	} else {
		ctx = dbscope.EnsureTenant(ctx, dbscope.Tenant{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		})
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tender.Tender, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultSweepLimit
		}

		t := tenderdb.Tenders
		inFlight := tenderdb.TenderOffers.As("tof")
		awaiting := func(extra ...keel.Pred) *keel.Query[tender.TenderOffer] {
			q := inFlight.Pick(keel.Raw("1")).Where(
				inFlight.TenderID.EqCol(t.ID),
				inFlight.OrganizationID.EqCol(t.OrganizationID),
				inFlight.BusinessUnitID.EqCol(t.BusinessUnitID),
				inFlight.Status.In(awaitingAnswerStatuses...),
			).Where(extra...)
			if system {
				q = q.AllTenants()
			}
			return q
		}

		// Arm 1: the current offer is overdue. Arm 2: no offer is in flight at
		// all past the grace window — a workflow that died before its first
		// dispatch, or after its last offer resolved without reaching a terminal
		// tender status. Nudging a healthy workflow that is merely between
		// offers is harmless.
		q := t.Where(
			t.Status.Eq(tender.StatusActive),
			keel.Or(
				keel.Exists(awaiting(inFlight.ExpiresAt.Lte(req.ExpiredBy))),
				keel.And(
					keel.NotExists(awaiting()),
					keel.Raw(
						"greatest(?, coalesce(?, 0)) <= ?",
						t.CreatedAt,
						t.LastBouncedAt,
						req.ExpiredBy,
					),
				),
			),
		).
			Load(rankedOffers()).
			Limit(limit)
		if system {
			q = q.AllTenants()
		}

		entities, err := q.List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list tenders for sweep", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SetWorkflowID(
	ctx context.Context,
	_ pagination.TenantInfo,
	tenderID pulid.ID,
	workflowID string,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := tenderdb.Tenders
		n, err := t.Where(t.ID.Eq(tenderID)).Update(
			ctx,
			r.db.Keel(ctx),
			t.WorkflowID.To(workflowID),
			t.Version.ToExpr(t.Version),
		)
		if err != nil {
			r.l.Error("failed to set tender workflow id", zap.Error(err))
			return err
		}

		return dberror.CheckAffected(n, "Tender", tenderID.String())
	})
}

// UpdateTenderStatus performs a compare-and-set transition. It returns false
// (with no error) when the tender was not in one of the expected statuses,
// which callers treat as "another actor already moved it".
func (r *repository) UpdateTenderStatus(
	ctx context.Context,
	req *tenderports.UpdateTenderStatusRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := tenderdb.Tenders
		sets := []keel.Assignment{t.Status.To(req.ToStatus)}
		if req.CurrentRank != nil {
			sets = append(sets, t.CurrentRank.To(*req.CurrentRank))
		}
		if req.CanceledByID != nil {
			sets = append(sets, t.CanceledByID.To(*req.CanceledByID))
		}
		if req.CancellationReason != "" {
			sets = append(sets, t.CancellationReason.To(req.CancellationReason))
		}
		if req.AcceptedOfferID != nil {
			sets = append(sets, t.AcceptedOfferID.To(*req.AcceptedOfferID))
		}
		if req.AcceptedAt != nil {
			sets = append(sets, t.AcceptedAt.To(*req.AcceptedAt))
		}
		if req.ExhaustedAt != nil {
			sets = append(sets, t.ExhaustedAt.To(*req.ExhaustedAt))
		}
		if req.CanceledAt != nil {
			sets = append(sets, t.CanceledAt.To(*req.CanceledAt))
		}

		n, err := t.Where(t.ID.Eq(req.TenderID), t.Status.In(req.FromStatus...)).
			Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error("failed to update tender status", zap.Error(err))
			return false, err
		}

		return n > 0, nil
	})
}

// UpdateOfferStatus performs a compare-and-set transition on one offer,
// returning false when the offer had already left the expected status.
func (r *repository) UpdateOfferStatus(
	ctx context.Context,
	req *tenderports.UpdateOfferStatusRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		o := tenderdb.TenderOffers
		sets := []keel.Assignment{o.Status.To(req.ToStatus)}
		if req.SentAt != nil {
			sets = append(sets, o.SentAt.To(*req.SentAt))
		}
		if req.ExpiresAt != nil {
			sets = append(sets, o.ExpiresAt.To(*req.ExpiresAt))
		}
		if req.RespondedAt != nil {
			sets = append(sets, o.RespondedAt.To(*req.RespondedAt))
		}
		if req.ResponseSource != "" {
			sets = append(sets, o.ResponseSource.To(req.ResponseSource))
		}
		if req.DeclineReason != "" {
			sets = append(sets, o.DeclineReason.To(req.DeclineReason))
		}
		if req.DeliveryError != "" {
			sets = append(sets, o.DeliveryError.To(req.DeliveryError))
		}
		if req.EDIMessageID != nil {
			sets = append(sets, o.EDIMessageID.To(*req.EDIMessageID))
		}
		sets = append(sets, offerCounterSets(req)...)

		n, err := o.Where(o.ID.Eq(req.OfferID), o.Status.In(req.FromStatus...)).
			Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			r.l.Error("failed to update tender offer status", zap.Error(err))
			return false, err
		}

		return n > 0, nil
	})
}

func offerCounterSets(req *tenderports.UpdateOfferStatusRequest) []keel.Assignment {
	o := tenderdb.TenderOffers
	var sets []keel.Assignment
	if counter := req.Counter; counter != nil {
		sets = append(sets,
			o.CounterRate.To(decimal.NullDecimal{Decimal: counter.Rate, Valid: true}),
			o.CounterNote.To(counter.Note),
			o.CounteredAt.To(counter.At),
			o.CounterMargin.To(counter.MarginPercent),
			o.CounterNeedsApproval.To(counter.NeedsApproval),
		)
	}
	if decided := req.CounterDecided; decided != nil {
		sets = append(sets, o.CounterDecidedAt.To(decided.At))
		if decided.UserID != nil {
			sets = append(sets, o.CounterDecidedByID.To(*decided.UserID))
		}
	}
	return sets
}

// errReopenLost unwinds a reopen whose tender moved but whose accepted offer
// had already left Accepted, so the tender is never live without a revoked
// acceptance behind it.
var errReopenLost = errors.New("tender reopen rolled back: accepted offer moved")

// Reopen revokes the accepted offer, puts the tender back to live on the next
// round and inserts the re-offers, all in one savepoint. It reports false when
// the tender no longer holds that acceptance.
func (r *repository) Reopen(
	ctx context.Context,
	req *tenderports.ReopenTenderRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		reopened := false
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			db := r.db.Keel(ctx)
			moved, err := reopenTender(ctx, db, req)
			if err != nil || !moved {
				return err
			}
			if err = revokeAccepted(ctx, db, req); err != nil {
				return err
			}
			if err = insertReoffers(ctx, db, req); err != nil {
				return err
			}
			reopened = true
			return nil
		})
		if errors.Is(err, errReopenLost) {
			return false, nil
		}
		if err != nil {
			return false, dberror.MapRetryableTransactionError(
				err, "The tender is busy. Retry the request.",
			)
		}
		return reopened, nil
	})
}

func reopenTender(
	ctx context.Context,
	db *keel.DB,
	req *tenderports.ReopenTenderRequest,
) (bool, error) {
	t := tenderdb.Tenders
	n, err := t.Where(
		t.ID.Eq(req.TenderID),
		t.AcceptedOfferID.Eq(req.RevokedOfferID),
		t.Status.In(tender.StatusAccepted, tender.StatusNeedsReview),
	).Update(
		ctx,
		db,
		t.Status.To(tender.StatusActive),
		t.BounceCount.Add(1),
		t.LastBouncedAt.To(req.At),
		t.AcceptedOfferID.ToNull(),
		t.AcceptedAt.ToNull(),
		t.CurrentRank.To(0),
		t.UpdatedAt.To(req.At),
	)
	if err != nil {
		return false, fmt.Errorf("reopen tender: %w", err)
	}

	return n > 0, nil
}

func revokeAccepted(
	ctx context.Context,
	db *keel.DB,
	req *tenderports.ReopenTenderRequest,
) error {
	o := tenderdb.TenderOffers
	sets := []keel.Assignment{
		o.Status.To(tender.OfferStatusRevoked),
		o.RevokedAt.To(req.At),
		o.RevokeReason.To(req.Reason),
		o.UpdatedAt.To(req.At),
	}
	if req.RevokedByID != nil {
		sets = append(sets, o.RevokedByID.To(*req.RevokedByID))
	}

	n, err := o.Where(
		o.ID.Eq(req.RevokedOfferID),
		o.TenderID.Eq(req.TenderID),
		o.Status.Eq(tender.OfferStatusAccepted),
	).Update(ctx, db, sets...)
	if err != nil {
		return fmt.Errorf("revoke accepted tender offer: %w", err)
	}
	if n == 0 {
		return errReopenLost
	}

	return nil
}

func insertReoffers(
	ctx context.Context,
	db *keel.DB,
	req *tenderports.ReopenTenderRequest,
) error {
	stampOffers(req.Reoffers, req.TenderID, req.TenantInfo)
	if err := tenderdb.TenderOffers.CreateMany(ctx, db, req.Reoffers); err != nil {
		return fmt.Errorf(
			"insert tender re-offers: %w",
			offerErrors.Map(err, req.TenderID.String()),
		)
	}

	return nil
}

func (r *repository) BulkUpdateOfferStatus(
	ctx context.Context,
	req *tenderports.BulkOfferStatusRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		o := tenderdb.TenderOffers
		n, err := o.Where(
			o.TenderID.Eq(req.TenderID),
			o.Status.In(req.FromStatus...),
			keel.When(!req.ExceptID.IsNil(), o.ID.Ne(req.ExceptID)),
		).Update(ctx, r.db.Keel(ctx), o.Status.To(req.ToStatus))
		if err != nil {
			r.l.Error("failed to bulk update tender offer status", zap.Error(err))
			return 0, err
		}

		return n, nil
	})
}

// RecordLateOfferResponse appends a late response to an already-terminal
// offer without touching its status.
func (r *repository) RecordLateOfferResponse(
	ctx context.Context,
	req *tenderports.RecordLateOfferResponseRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		o := tenderdb.TenderOffers
		_, err := o.Where(o.ID.Eq(req.OfferID)).Update(
			ctx,
			r.db.Keel(ctx),
			o.LateResponseAction.To(string(req.Action)),
			o.LateResponseAt.To(req.OccurredAt),
			o.ResponseSource.To(req.Source),
			o.Version.ToExpr(o.Version),
		)
		if err != nil {
			r.l.Error("failed to record late tender offer response", zap.Error(err))
		}
		return err
	})
}

func (r *repository) CreateToken(
	ctx context.Context,
	entity *tender.TenderOfferToken,
) (*tender.TenderOfferToken, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tender.TenderOfferToken, error) {
		if err := tenderdb.TenderOfferTokens.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create tender offer token", zap.Error(err))
			return nil, tokenErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// GetTokenByHash resolves a public token. It is deliberately not
// tenant-scoped: the caller only holds the opaque token, and the row itself
// carries the tenant that every follow-up query is scoped to.
func (r *repository) GetTokenByHash(
	ctx context.Context,
	tokenHash string,
) (*tender.TenderOfferToken, error) {
	ctx = dbscope.WithSystem(ctx, "resolve a public tender offer link before its tenant is known")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.TenderOfferToken, error) {
		tk := tenderdb.TenderOfferTokens
		entity, err := tk.AllTenants().Where(tk.TokenHash.Eq(tokenHash)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil token represents an optional absence
			}
			r.l.Error("failed to get tender offer token", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

// MarkTokenUsed is the single-use gate for public response links: the
// conditional update only wins for the first caller.
func (r *repository) MarkTokenUsed(
	ctx context.Context,
	req *tenderports.MarkTokenUsedRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		tk := tenderdb.TenderOfferTokens
		n, err := tk.Where(tk.ID.Eq(req.TokenID), tk.UsedAt.IsNull(), tk.RevokedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), tk.UsedAt.To(req.UsedAt))
		if err != nil {
			r.l.Error("failed to mark tender offer token used", zap.Error(err))
			return false, err
		}

		return n > 0, nil
	})
}

func (r *repository) RevokeTokensForOffer(
	ctx context.Context,
	_ pagination.TenantInfo,
	offerID pulid.ID,
	revokedAt int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tk := tenderdb.TenderOfferTokens
		_, err := tk.Where(tk.TenderOfferID.Eq(offerID), tk.RevokedAt.IsNull(), tk.UsedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx), tk.RevokedAt.To(revokedAt))
		if err != nil {
			r.l.Error("failed to revoke tender offer tokens", zap.Error(err))
		}
		return err
	})
}

// ListAcceptedMissingRateConfirmation is the recovery sweep's cross-tenant
// read: accepted tenders whose carrier is already covering the move but whose
// rate confirmation never landed. The NOT EXISTS arm deliberately treats any
// non-voided revision as covered, so a dispatcher-managed agreement is never
// re-entered, and the assignment arm excludes the needs-review family because
// those moves have no matching-carrier assignment at all.
func (r *repository) ListAcceptedMissingRateConfirmation(
	ctx context.Context,
	req tenderports.ListAcceptedMissingRateConfirmationRequest,
) ([]*tender.Tender, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list accepted tenders missing a rate confirmation across every organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tender.Tender, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultSweepLimit
		}

		t := tenderdb.Tenders
		o := tenderdb.TenderOffers
		ca := shipmentdb.CarrierAssignments
		rc := rateconfirmationdb.RateConfirmations

		acceptedOfferCarrier := o.AllTenants().
			Pick(o.CarrierID).
			Where(
				o.ID.EqCol(t.AcceptedOfferID),
				o.OrganizationID.EqCol(t.OrganizationID),
				o.BusinessUnitID.EqCol(t.BusinessUnitID),
			)

		standingRateConfirmation := rc.AllTenants().
			Pick(keel.Raw("1")).
			Where(
				rc.CarrierAssignmentID.EqCol(ca.ID),
				rc.OrganizationID.EqCol(ca.OrganizationID),
				rc.BusinessUnitID.EqCol(ca.BusinessUnitID),
				rc.Status.Ne(rateconfirmation.StatusVoided),
			)

		coveringAssignment := ca.AllTenants().
			Pick(keel.Raw("1")).
			Where(
				ca.ShipmentMoveID.EqCol(t.ShipmentMoveID),
				ca.OrganizationID.EqCol(t.OrganizationID),
				ca.BusinessUnitID.EqCol(t.BusinessUnitID),
				ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
				ca.CarrierID.EqCol(keel.Scalar(acceptedOfferCarrier)),
				keel.NotExists(standingRateConfirmation),
			)

		entities, err := t.AllTenants().
			Where(
				t.Status.Eq(tender.StatusAccepted),
				t.AcceptedOfferID.IsNotNull(),
				t.AcceptedAt.Gt(req.AcceptedAfter),
				t.AcceptedAt.Lte(req.AcceptedBefore),
				keel.Exists(coveringAssignment),
			).
			Order(t.AcceptedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list accepted tenders missing a rate confirmation", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

// PurgeDeadOfferTokens deletes offer links that can no longer authorize
// anything — used, revoked, or expired — once the retention window past their
// terminal moment has elapsed. A live link always coalesces to a future
// expires_at, so it can never match.
func (r *repository) PurgeDeadOfferTokens(
	ctx context.Context,
	req tenderports.PurgeDeadTokensRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "purge dead tender offer tokens across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultTokenPurgeLimit
		}

		tk := tenderdb.TenderOfferTokens
		dead := tk.As("dead")
		n, err := tk.AllTenants().
			Where(tk.ID.InQuery(dead.AllTenants().
				Pick(dead.ID).
				Where(keel.Raw(
					"coalesce(?, ?, ?) < ?",
					dead.UsedAt,
					dead.RevokedAt,
					dead.ExpiresAt,
					req.DeadBefore,
				)).
				Limit(limit))).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge dead tender offer tokens", zap.Error(err))
			return 0, err
		}

		return n, nil
	})
}
