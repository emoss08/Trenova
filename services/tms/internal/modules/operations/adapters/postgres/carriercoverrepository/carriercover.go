package carriercoverrepository

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) carriercoverports.Reader {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.carrier-cover-repository")}
}

type carrierRow struct {
	ID               pulid.ID
	Name             string
	Code             string
	Status           string
	ComplianceStatus string
}

type policyRow struct {
	CarrierID      pulid.ID
	PolicyType     string
	PolicyNumber   string
	ExpirationDate int64
}

type loadCountRow struct {
	CarrierID pulid.ID
	Open      int64
	Readable  int64
}

type loadRow struct {
	CarrierID  pulid.ID
	ShipmentID pulid.ID
	ProNumber  string
	DeliversAt *int64
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

func openAssignments(ids []pulid.ID) []keel.Pred {
	a := shipmentdb.CarrierAssignments
	m := shipmentdb.ShipmentMoves

	return []keel.Pred{
		a.CarrierID.In(ids...),
		a.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
		m.Status.NotIn(shipmentvocab.MoveStatusCompleted, shipmentvocab.MoveStatusCanceled),
	}
}

func carriersQuery(ids []pulid.ID) keel.Projection {
	c := carrierdb.Carriers

	return c.Pick(c.ID, text(c.Name), text(c.Code), text(c.Status), text(c.ComplianceStatus)).
		Where(c.ID.In(ids...))
}

func policiesQuery(ids []pulid.ID) keel.Projection {
	p := carrierdb.CarrierInsurancePolicies

	return p.Pick(p.CarrierID, text(p.PolicyType), text(p.PolicyNumber), p.ExpirationDate).
		DistinctOn(p.CarrierID, p.PolicyType).
		Where(
			p.CarrierID.In(ids...),
			p.PolicyType.In(carrier.RequiredInsurancePolicyTypes...),
		).
		Order(p.CarrierID.Asc(), p.PolicyType.Asc(), p.ExpirationDate.Desc())
}

func loadCountsQuery(ids []pulid.ID) keel.Projection {
	a := shipmentdb.CarrierAssignments
	m := shipmentdb.ShipmentMoves
	s := shipmentdb.Shipments

	return a.Join(m, m.ID.EqCol(a.ShipmentMoveID)).
		LeftJoin(s, s.ID.EqCol(m.ShipmentID)).
		Group(a.CarrierID).
		Pick(a.CarrierID, keel.Count(), keel.CountOf(s.ID)).
		Where(openAssignments(ids)...)
}

func (r *repository) Covers(
	ctx context.Context,
	req *carriercoverports.CoversRequest,
) (map[pulid.ID]*carriercoverports.Cover, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]*carriercoverports.Cover{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*carriercoverports.Cover, error) {
			return r.covers(ctx, req.IDs, req.Loads)
		},
	)
}

func (r *repository) covers(
	ctx context.Context,
	ids []pulid.ID,
	loads bool,
) (map[pulid.ID]*carriercoverports.Cover, error) {
	var carriers []carrierRow
	var policies []policyRow
	var counts []loadCountRow
	reads := []keel.BatchItem{
		keel.IntoSlice(carriersQuery(ids), &carriers),
		keel.IntoSlice(policiesQuery(ids), &policies),
	}
	if loads {
		reads = append(reads, keel.IntoSlice(loadCountsQuery(ids), &counts))
	}
	if err := keel.Batch(ctx, r.db.Keel(ctx), reads...); err != nil {
		r.l.Error("failed to read carrier cover", zap.Error(err))
		return nil, err
	}

	return assemble(carriers, policies, counts, loads), nil
}

func assemble(
	carriers []carrierRow,
	policies []policyRow,
	counts []loadCountRow,
	loadsRead bool,
) map[pulid.ID]*carriercoverports.Cover {
	held := make(map[pulid.ID][]*carrier.CarrierInsurancePolicy, len(carriers))
	for idx := range policies {
		row := &policies[idx]
		held[row.CarrierID] = append(held[row.CarrierID], &carrier.CarrierInsurancePolicy{
			CarrierID:      row.CarrierID,
			PolicyType:     carrier.InsurancePolicyType(row.PolicyType),
			PolicyNumber:   row.PolicyNumber,
			ExpirationDate: row.ExpirationDate,
		})
	}
	loads := make(map[pulid.ID]loadCountRow, len(counts))
	for _, row := range counts {
		loads[row.CarrierID] = row
	}

	out := make(map[pulid.ID]*carriercoverports.Cover, len(carriers))
	for idx := range carriers {
		row := &carriers[idx]
		required := carrier.SoonestRequiredCover(held[row.ID])
		count := loads[row.ID]
		out[row.ID] = &carriercoverports.Cover{
			CarrierID:        row.ID,
			Name:             strings.TrimSpace(row.Name),
			Code:             strings.TrimSpace(row.Code),
			Status:           carrier.Status(row.Status),
			ComplianceStatus: carrier.ComplianceStatus(row.ComplianceStatus),
			Policy:           required.Policy,
			Missing:          required.Missing,
			LoadsRead:        loadsRead,
			OpenLoads:        count.Open,
			ReadableLoads:    count.Readable,
		}
	}

	return out
}

func candidatesQuery(req *carriercoverports.EndingRequest) keel.Projection {
	p := carrierdb.CarrierInsurancePolicies
	soonest := p.ExpirationDate.Min()

	return p.Group(p.CarrierID).
		Pick(p.CarrierID).
		Where(
			p.ExpirationDate.Between(req.Now-carriercoverports.LapsedLookbackSeconds, req.Until),
			p.PolicyType.In(carrier.RequiredInsurancePolicyTypes...),
			p.Carrier().Status.Eq(carrier.StatusActive),
		).
		Order(keel.Asc(soonest), p.CarrierID.Asc()).
		Limit(carriercoverports.CandidateCarriers)
}

func (r *repository) Ending(
	ctx context.Context,
	req *carriercoverports.EndingRequest,
) ([]*carriercoverports.Cover, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*carriercoverports.Cover, error) {
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), candidatesQuery(req))
		if err != nil {
			r.l.Error("failed to find carriers whose cover ends", zap.Error(err))
			return nil, err
		}
		if len(ids) == 0 {
			return []*carriercoverports.Cover{}, nil
		}
		covers, err := r.covers(ctx, ids, req.Loads)
		if err != nil {
			return nil, err
		}

		return soonestEnding(covers, req), nil
	})
}

func soonestEnding(
	covers map[pulid.ID]*carriercoverports.Cover,
	req *carriercoverports.EndingRequest,
) []*carriercoverports.Cover {
	out := make([]*carriercoverports.Cover, 0, len(covers))
	from := req.Now - carriercoverports.LapsedLookbackSeconds
	for _, cover := range covers {
		if cover.Status != carrier.StatusActive || cover.Policy == nil {
			continue
		}
		if ends := cover.EndsAt(); ends < from || ends > req.Until {
			continue
		}
		out = append(out, cover)
	}
	slices.SortFunc(out, func(a, b *carriercoverports.Cover) int {
		return cmp.Or(
			cmp.Compare(a.EndsAt(), b.EndsAt()),
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.CarrierID.String(), b.CarrierID.String()),
		)
	})
	limit := req.Limit
	if limit <= 0 || limit > carriercoverports.MaxCarriers {
		limit = carriercoverports.DefaultCarriers
	}

	return out[:min(len(out), limit)]
}

func loadsQuery(req *carriercoverports.LoadsRequest) keel.Projection {
	a := shipmentdb.CarrierAssignments
	m := shipmentdb.ShipmentMoves
	s := shipmentdb.Shipments
	last := shipmentdb.Stops.As("last_stop")
	delivers := last.ScheduledWindowEnd.Coalesce(last.ScheduledWindowStart)
	place := keel.RowNumber().Over(
		[]keel.Expr{a.CarrierID},
		keel.Asc(delivers).NullsLast(),
		s.ID.Asc(),
	)
	body := a.Join(m, m.ID.EqCol(a.ShipmentMoveID)).
		Join(s, s.ID.EqCol(m.ShipmentID)).
		LeftJoinLateral(
			last.Pick(last.ScheduledWindowStart, last.ScheduledWindowEnd).
				Where(last.ShipmentMoveID.EqCol(m.ID)).
				Order(last.Sequence.Desc()).
				Limit(1),
		).
		Pick(a.CarrierID, s.ID, text(s.ProNumber), delivers, place).
		Where(openAssignments(req.CarrierIDs)...)
	loads := keel.CTE(
		"open_carrier_loads",
		[]string{"carrier_id", "shipment_id", "pro_number", "delivers_at", "place"},
		body,
	)

	return keel.RowsOf(loads.TableRef()).
		With(loads).
		Pick(
			loads.Col("carrier_id"),
			loads.Col("shipment_id"),
			loads.Col("pro_number"),
			loads.Col("delivers_at"),
		).
		Where(loads.Col("place").Lte(req.PerCarrier)).
		Order(keel.Asc(loads.Col("carrier_id")), keel.Asc(loads.Col("place"))).
		Limit(len(req.CarrierIDs) * req.PerCarrier)
}

func (r *repository) Loads(
	ctx context.Context,
	req *carriercoverports.LoadsRequest,
) (map[pulid.ID][]carriercoverports.Load, error) {
	if len(req.CarrierIDs) == 0 || req.PerCarrier <= 0 {
		return map[pulid.ID][]carriercoverports.Load{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]carriercoverports.Load, error) {
			rows, err := keel.Into[loadRow](ctx, r.db.Keel(ctx), loadsQuery(req))
			if err != nil {
				r.l.Error("failed to read carriers' open loads", zap.Error(err))
				return nil, err
			}
			out := make(map[pulid.ID][]carriercoverports.Load, len(req.CarrierIDs))
			for idx := range rows {
				row := &rows[idx]
				load := carriercoverports.Load{
					CarrierID:  row.CarrierID,
					ShipmentID: row.ShipmentID,
					ProNumber:  strings.TrimSpace(row.ProNumber),
				}
				if row.DeliversAt != nil {
					load.DeliversAt = *row.DeliversAt
				}
				out[row.CarrierID] = append(out[row.CarrierID], load)
			}

			return out, nil
		},
	)
}
