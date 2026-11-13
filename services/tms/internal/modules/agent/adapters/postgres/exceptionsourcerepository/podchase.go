package exceptionsourcerepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

// NewPODChaseFacts is the read the proof-of-delivery chase plans from.
func NewPODChaseFacts(p Params) podchaseports.FactsReader {
	return &repository{db: p.DB}
}

type chaseMove struct {
	ID         pulid.ID
	ShipmentID pulid.ID
	Sequence   int64
}

type chaseAssignment struct {
	ShipmentMoveID  pulid.ID
	PrimaryWorkerID pulid.ID
}

type chaseCarrierAssignment struct {
	ShipmentMoveID pulid.ID
	CarrierID      pulid.ID
}

type chaseWorker struct {
	ID        pulid.ID
	FirstName string
	LastName  string
	UserID    pulid.ID
	Email     string
	Phone     string
}

type chaseCarrier struct {
	ID    pulid.ID
	Name  string
	Email string
}

// ListPODChaseFacts reads the shipments among those asked for that are still
// delivered with no proof of delivery on file, and who hauled each one's
// last move: three round trips for the whole page, whatever its size.
func (r *repository) ListPODChaseFacts(
	ctx context.Context,
	req *podchaseports.ListFactsRequest,
) ([]*podchaseports.Facts, error) {
	if len(req.ShipmentIDs) == 0 {
		return []*podchaseports.Facts{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*podchaseports.Facts, error) {
		db := r.db.Keel(ctx)
		mv := shipmentdb.ShipmentMoves
		sp := shipmentdb.Shipments

		var missing []watchtowerports.MissingPOD
		var bols []shipmentBOL
		var moves []chaseMove
		if err := keel.Batch(ctx, db,
			keel.IntoSlice(missingPODsQuery(&watchtowerports.ListMissingPODsRequest{
				TenantInfo:      req.TenantInfo,
				DeliveredBefore: req.DeliveredBefore,
				ShipmentIDs:     req.ShipmentIDs,
				Limit:           len(req.ShipmentIDs),
			}), &missing),
			keel.IntoSlice(sp.Pick(sp.ID, sp.BOL.Coalesce("")).
				Where(sp.ID.In(req.ShipmentIDs...)).
				Limit(len(req.ShipmentIDs)), &bols),
			keel.IntoSlice(mv.Pick(mv.ID, mv.ShipmentID, mv.Sequence).
				Where(mv.ShipmentID.In(req.ShipmentIDs...)).
				Order(mv.ShipmentID.Asc(), mv.Sequence.Desc()).
				Limit(maxChaseMoves(len(req.ShipmentIDs))), &moves),
		); err != nil {
			return nil, fmt.Errorf("read the shipments missing a proof of delivery: %w", err)
		}
		if len(missing) == 0 {
			return []*podchaseports.Facts{}, nil
		}

		lastMove := lastMoveByShipment(moves)
		haulers, err := r.haulers(ctx, lastMove)
		if err != nil {
			return nil, err
		}

		bolByShipment := make(map[pulid.ID]string, len(bols))
		for _, row := range bols {
			bolByShipment[row.ID] = row.BOL
		}
		facts := make([]*podchaseports.Facts, 0, len(missing))
		for i := range missing {
			row := &missing[i]
			fact := &podchaseports.Facts{
				ShipmentID:   row.ShipmentID,
				ProNumber:    row.ProNumber,
				BOL:          bolByShipment[row.ShipmentID],
				CustomerName: row.CustomerName,
				DeliveredAt:  row.DeliveredAt,
			}
			if move, ok := lastMove[row.ShipmentID]; ok {
				fact.Driver = haulers.drivers[move]
				fact.Carrier = haulers.carriers[move]
			}
			facts = append(facts, fact)
		}

		return facts, nil
	})
}

type shipmentBOL struct {
	ID  pulid.ID
	BOL string
}

// maxChaseMoves bounds the move read: a shipment rarely has more than a few.
func maxChaseMoves(shipments int) int {
	return shipments * 8
}

func lastMoveByShipment(moves []chaseMove) map[pulid.ID]pulid.ID {
	last := make(map[pulid.ID]pulid.ID, len(moves))
	for _, move := range moves {
		if _, seen := last[move.ShipmentID]; !seen {
			last[move.ShipmentID] = move.ID
		}
	}

	return last
}

type chaseHaulers struct {
	drivers  map[pulid.ID]*podchaseports.Driver
	carriers map[pulid.ID]*podchaseports.Carrier
}

// haulers reads who hauled each move: its driver, whether they are on Dash
// and their email, or its carrier and the carrier's email.
func (r *repository) haulers(
	ctx context.Context,
	lastMove map[pulid.ID]pulid.ID,
) (*chaseHaulers, error) {
	out := &chaseHaulers{
		drivers:  make(map[pulid.ID]*podchaseports.Driver),
		carriers: make(map[pulid.ID]*podchaseports.Carrier),
	}
	moveIDs := make([]pulid.ID, 0, len(lastMove))
	for _, move := range lastMove {
		moveIDs = append(moveIDs, move)
	}
	if len(moveIDs) == 0 {
		return out, nil
	}

	db := r.db.Keel(ctx)
	as := shipmentdb.Assignments
	ca := shipmentdb.CarrierAssignments
	var assignments []chaseAssignment
	var carrierAssignments []chaseCarrierAssignment
	if err := keel.Batch(ctx, db,
		keel.IntoSlice(as.Pick(as.ShipmentMoveID, as.PrimaryWorkerID).
			Where(
				as.ShipmentMoveID.In(moveIDs...),
				as.Status.Ne(shipmentvocab.AssignmentStatusCanceled),
				as.PrimaryWorkerID.IsNotNull(),
			).
			Limit(len(moveIDs)*2), &assignments),
		keel.IntoSlice(ca.Pick(ca.ShipmentMoveID, ca.CarrierID).
			Where(
				ca.ShipmentMoveID.In(moveIDs...),
				ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
			).
			Limit(len(moveIDs)*2), &carrierAssignments),
	); err != nil {
		return nil, fmt.Errorf("read who hauled the delivering moves: %w", err)
	}

	workerIDs := make([]pulid.ID, 0, len(assignments))
	for _, a := range assignments {
		workerIDs = append(workerIDs, a.PrimaryWorkerID)
	}
	carrierIDs := make([]pulid.ID, 0, len(carrierAssignments))
	for _, a := range carrierAssignments {
		carrierIDs = append(carrierIDs, a.CarrierID)
	}

	var workers []chaseWorker
	var carriers []chaseCarrier
	wk := workerdb.Workers
	cr := carrierdb.Carriers
	items := make([]keel.BatchItem, 0, 2)
	if len(workerIDs) > 0 {
		items = append(items, keel.IntoSlice(
			wk.Pick(
				wk.ID, wk.FirstName, wk.LastName,
				wk.UserID.Coalesce(""), wk.Email.Coalesce(""), wk.PhoneNumber.Coalesce(""),
			).
				Where(wk.ID.In(workerIDs...)).
				Limit(len(workerIDs)), &workers))
	}
	if len(carrierIDs) > 0 {
		items = append(items, keel.IntoSlice(
			cr.Pick(cr.ID, cr.Name, cr.Email.Coalesce("")).
				Where(cr.ID.In(carrierIDs...)).
				Limit(len(carrierIDs)), &carriers))
	}
	if len(items) > 0 {
		if err := keel.Batch(ctx, db, items...); err != nil {
			return nil, fmt.Errorf("read the drivers and carriers who hauled the moves: %w", err)
		}
	}

	workerByID := make(map[pulid.ID]*chaseWorker, len(workers))
	for i := range workers {
		workerByID[workers[i].ID] = &workers[i]
	}
	carrierByID := make(map[pulid.ID]*chaseCarrier, len(carriers))
	for i := range carriers {
		carrierByID[carriers[i].ID] = &carriers[i]
	}
	for _, a := range assignments {
		if w, ok := workerByID[a.PrimaryWorkerID]; ok {
			out.drivers[a.ShipmentMoveID] = &podchaseports.Driver{
				WorkerID: w.ID,
				Name:     strings.TrimSpace(w.FirstName + " " + w.LastName),
				OnDash:   w.UserID.IsNotNil(),
				Email:    strings.TrimSpace(w.Email),
				HasPhone: strings.TrimSpace(w.Phone) != "",
			}
		}
	}
	for _, a := range carrierAssignments {
		if c, ok := carrierByID[a.CarrierID]; ok {
			out.carriers[a.ShipmentMoveID] = &podchaseports.Carrier{
				CarrierID: c.ID,
				Name:      c.Name,
				Email:     strings.TrimSpace(c.Email),
			}
		}
	}

	return out, nil
}
