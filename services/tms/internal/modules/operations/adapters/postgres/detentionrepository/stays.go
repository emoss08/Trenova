package detentionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence/detentionevidencedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type stayRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewStayRepository(p OccurrenceParams) detentionports.DetentionStayRepository {
	return &stayRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.detention-stay-repository"),
	}
}

type byHandRow struct {
	OccurrenceID pulid.ID
	Kind         string
}

func stayStage(stage detentionports.StayStage) []keel.Pred {
	o := detentiondb.DetentionOccurrences
	switch stage {
	case detentionports.StayStageToBill:
		return []keel.Pred{
			o.IsOpen.IsFalse(),
			o.Status.In(detention.OccurrenceStatusPending, detention.OccurrenceStatusApproved),
		}
	case detentionports.StayStageRunning:
		return []keel.Pred{o.IsOpen.IsTrue()}
	case detentionports.StayStageBilled:
		return []keel.Pred{o.IsOpen.IsFalse(), o.Status.Eq(detention.OccurrenceStatusBilled)}
	case detentionports.StayStageDisputed:
		return []keel.Pred{o.IsOpen.IsFalse(), o.Status.Eq(detention.OccurrenceStatusDisputed)}
	case detentionports.StayStageWaived:
		return []keel.Pred{o.IsOpen.IsFalse(), o.Status.Eq(detention.OccurrenceStatusWaived)}
	case detentionports.StayStageNotBillable:
		return []keel.Pred{
			o.IsOpen.IsFalse(),
			o.Status.Eq(detention.OccurrenceStatusNotBillable),
		}
	case detentionports.StayStageAll:
	}

	return nil
}

func stayWhere(req *detentionports.ListDetentionStaysRequest) []keel.Pred {
	o := detentiondb.DetentionOccurrences
	sp := shipmentdb.Shipments

	return append([]keel.Pred{
		keel.Exists(sp.Pick(sp.ID).Where(sp.ID.EqCol(o.ShipmentID))),
		o.ClockStartAt.Gte(req.From),
		o.ClockStartAt.Lte(req.To),
		keel.When(req.LocationID.IsNotNil(), o.LocationID.Eq(req.LocationID)),
	}, stayStage(req.Stage)...)
}

func (r *stayRepository) ListStays(
	ctx context.Context,
	req *detentionports.ListDetentionStaysRequest,
) (*detentionports.DetentionStayPage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionports.DetentionStayPage, error) {
			o := detentiondb.DetentionOccurrences
			where := stayWhere(req)

			total, err := detentiondb.DetentionOccurrences.Query().Where(where...).
				Count(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to count detention stays", zap.Error(err))
				return nil, err
			}
			occurrences, err := namedOccurrences().
				Where(where...).
				Order(o.BillableAmount.Desc(), o.ClockStartAt.Desc(), o.ID.Asc()).
				Limit(req.Limit).
				Offset(req.Offset).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list detention stays", zap.Error(err))
				return nil, err
			}

			stays, err := r.withTimesByHand(ctx, occurrences)
			if err != nil {
				return nil, err
			}

			return &detentionports.DetentionStayPage{Items: stays, Total: int(total)}, nil
		},
	)
}

func (r *stayRepository) withTimesByHand(
	ctx context.Context,
	occurrences []*detention.DetentionOccurrence,
) ([]*detentionports.DetentionStay, error) {
	stays := make([]*detentionports.DetentionStay, 0, len(occurrences))
	if len(occurrences) == 0 {
		return stays, nil
	}
	ids := make([]pulid.ID, len(occurrences))
	for idx, occurrence := range occurrences {
		ids[idx] = occurrence.ID
	}
	ev := detentionevidencedb.DetentionEvidences
	rows, err := keel.Into[byHandRow](ctx, r.db.Keel(ctx),
		ev.Pick(ev.DetentionOccurrenceID, keel.CastText(ev.Kind)).
			Where(
				ev.DetentionOccurrenceID.In(ids...),
				ev.Source.Eq(detentionevidence.EvidenceSourceManual),
				ev.Kind.In(
					detentionevidence.EvidenceKindArrival,
					detentionevidence.EvidenceKindDeparture,
				),
			).
			Distinct())
	if err != nil {
		r.l.Error("failed to read detention times entered by hand", zap.Error(err))
		return nil, err
	}
	byHand := make(map[pulid.ID]*detentionports.DetentionStay, len(occurrences))
	for _, occurrence := range occurrences {
		stay := &detentionports.DetentionStay{Occurrence: occurrence}
		byHand[occurrence.ID] = stay
		stays = append(stays, stay)
	}
	for _, row := range rows {
		stay, ok := byHand[row.OccurrenceID]
		if !ok {
			continue
		}
		switch detentionevidence.EvidenceKind(row.Kind) {
		case detentionevidence.EvidenceKindArrival:
			stay.ArrivalByHand = true
		case detentionevidence.EvidenceKindDeparture:
			stay.DepartureByHand = true
		case detentionevidence.EvidenceKindAppointment, detentionevidence.EvidenceKindNotice,
			detentionevidence.EvidenceKindDocument, detentionevidence.EvidenceKindStatusChange,
			detentionevidence.EvidenceKindRecalculated, detentionevidence.EvidenceKindWaiver,
			detentionevidence.EvidenceKindDispute:
		}
	}

	return stays, nil
}
