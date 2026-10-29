package recordsummaryrepository

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordsummaryports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const minorExponent = -2

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Repository struct {
	db *postgres.Connection
}

func New(p Params) recordsummaryports.RecordSummarizer {
	return &Repository{db: p.DB}
}

type projection interface {
	keel.Projection
	ToSQL(ctx context.Context, db *keel.DB) (string, []any, error)
}

type summaryRow struct {
	ID       pulid.ID
	Label    string
	Subtitle string
	Status   string
}

type shipmentRow struct {
	ID      pulid.ID
	Label   string
	Status  string
	Revenue decimal.NullDecimal
}

type stopRow struct {
	ShipmentID   pulid.ID
	MoveSequence int64
	Sequence     int64
	Type         string
	Status       string
	WindowStart  int64
	WindowEnd    *int64
	Arrival      *int64
	Estimate     *int64
	City         string
	State        string
}

type invoiceRow struct {
	ID          pulid.ID
	Label       string
	Subtitle    string
	Status      string
	Currency    string
	TotalMinor  int64
	BalanceDue  int64
	InvoiceDate int64
	DueDate     *int64
}

type entryRow struct {
	ID        pulid.ID
	Label     string
	Subtitle  string
	Status    string
	Debit     int64
	CreatedAt int64
}

func text(e keel.Expr) keel.StringExpr {
	return keel.CastText(e).Coalesce("")
}

//nolint:exhaustive // only the resources a board draws as rows are summarized
var plain = map[permission.Resource]func(ids []pulid.ID) projection{
	permission.ResourceCarrier: func(ids []pulid.ID) projection {
		t := carrierdb.Carriers
		mc := keel.Text("MC ").Cat(t.MCNumber.NullIf(""))
		dot := keel.Text("DOT ").Cat(t.DOTNumber.NullIf(""))
		return t.Pick(t.ID, text(t.Name), text(mc.Coalesce(dot)), text(t.ComplianceStatus)).
			Where(t.ID.In(ids...))
	},
	permission.ResourceCustomer: func(ids []pulid.ID) projection {
		t := customerdb.Customers
		return t.Pick(t.ID, text(t.Name), text(t.Code), text(t.Status)).Where(t.ID.In(ids...))
	},
	permission.ResourceWorker: func(ids []pulid.ID) projection {
		t := workerdb.Workers
		return t.Pick(
			t.ID, text(keel.ConcatWS(" ", t.FirstName, t.LastName)), text(t.Type), text(t.Status),
		).Where(t.ID.In(ids...))
	},
	permission.ResourceTractor: func(ids []pulid.ID) projection {
		t := tractordb.Tractors
		return t.Pick(t.ID, text(t.Code), text(keel.ConcatWS(" ", t.Make, t.Model)), text(t.Status)).
			Where(t.ID.In(ids...))
	},
	permission.ResourceTrailer: func(ids []pulid.ID) projection {
		t := trailerdb.Trailers
		return t.Pick(t.ID, text(t.Code), text(keel.ConcatWS(" ", t.Make, t.Model)), text(t.Status)).
			Where(t.ID.In(ids...))
	},
	permission.ResourceLocation: func(ids []pulid.ID) projection {
		t := locationdb.Locations
		st := usstatedb.UsStates
		return t.LeftJoin(st, st.ID.EqCol(t.StateID)).
			Pick(t.ID, text(t.Name), text(keel.ConcatWS(", ", t.City, st.Abbreviation)), text(t.Status)).
			Where(t.ID.In(ids...))
	},
}

func shipmentsQuery(ids []pulid.ID) projection {
	sp := shipmentdb.Shipments

	return sp.Pick(sp.ID, text(sp.ProNumber), text(sp.Status), sp.TotalChargeAmount).
		Where(sp.ID.In(ids...))
}

func stopsQuery(ids []pulid.ID) projection {
	stp := shipmentdb.Stops
	sm := shipmentdb.ShipmentMoves
	loc := locationdb.Locations
	st := usstatedb.UsStates
	eta := stopetadb.StopETAs

	return stp.Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
		LeftJoin(loc, loc.ID.EqCol(stp.LocationID)).
		LeftJoin(st, st.ID.EqCol(loc.StateID)).
		LeftJoin(eta, eta.StopID.EqCol(stp.ID)).
		Pick(
			sm.ShipmentID, sm.Sequence, stp.Sequence, text(stp.Type), text(stp.Status),
			stp.ScheduledWindowStart, stp.ScheduledWindowEnd, stp.ActualArrival,
			eta.EstimatedArrival, text(loc.City), text(st.Abbreviation),
		).
		Where(sm.ShipmentID.In(ids...), stp.Status.Ne(shipmentvocab.StopStatusCanceled))
}

func invoicesQuery(ids []pulid.ID) projection {
	t := invoicedb.Invoices

	return t.Pick(
		t.ID, text(t.Number), text(t.BillToName), text(t.Status), text(t.CurrencyCode),
		t.TotalAmountMinor, t.BalanceDueMinor, t.InvoiceDate, t.DueDate,
	).Where(t.ID.In(ids...))
}

func entriesQuery(ids []pulid.ID) projection {
	t := journalentrydb.JournalEntries

	return t.Pick(
		t.ID, text(t.EntryNumber), text(t.Description), text(t.Status), t.TotalDebit, t.CreatedAt,
	).Where(t.ID.In(ids...))
}

func (r *Repository) Summaries(
	ctx context.Context,
	req *recordsummaryports.RecordSummaryRequest,
) (recordsummaryports.RecordSummaries, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (recordsummaryports.RecordSummaries, error) {
			out := make(recordsummaryports.RecordSummaries, len(req.Refs))
			for resource, ids := range req.Refs {
				ids = distinct(ids)
				if len(ids) == 0 {
					continue
				}
				read, err := r.read(ctx, resource, ids, req.Now)
				if err != nil {
					return nil, fmt.Errorf("read %s summaries: %w", resource, err)
				}
				if read != nil {
					out[resource] = read
				}
			}

			return out, nil
		},
	)
}

func (r *Repository) read(
	ctx context.Context,
	resource permission.Resource,
	ids []pulid.ID,
	now int64,
) (map[pulid.ID]boardassemble.Summary, error) {
	if resource == permission.ResourceShipment {
		return r.shipments(ctx, ids, now)
	}
	if resource == permission.ResourceInvoice {
		return r.invoices(ctx, ids, now)
	}
	if resource == permission.ResourceJournalEntry {
		return r.entries(ctx, ids, now)
	}
	query, ok := plain[resource]
	if !ok {
		return map[pulid.ID]boardassemble.Summary{}, nil
	}
	rows, err := keel.Into[summaryRow](ctx, r.db.Keel(ctx), query(ids))
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardassemble.Summary, len(rows))
	for _, row := range rows {
		out[row.ID] = boardassemble.Summary{
			Label:    strings.TrimSpace(row.Label),
			Subtitle: strings.TrimSpace(row.Subtitle),
			Status:   row.Status,
		}
	}

	return out, nil
}

func (r *Repository) shipments(
	ctx context.Context,
	ids []pulid.ID,
	now int64,
) (map[pulid.ID]boardassemble.Summary, error) {
	rows, err := keel.Into[shipmentRow](ctx, r.db.Keel(ctx), shipmentsQuery(ids))
	if err != nil {
		return nil, err
	}
	stops, err := keel.Into[stopRow](ctx, r.db.Keel(ctx), stopsQuery(ids))
	if err != nil {
		return nil, err
	}
	byShipment := make(map[pulid.ID][]stopRow, len(rows))
	for _, stop := range stops {
		byShipment[stop.ShipmentID] = append(byShipment[stop.ShipmentID], stop)
	}

	out := make(map[pulid.ID]boardassemble.Summary, len(rows))
	for _, row := range rows {
		route := byShipment[row.ID]
		slices.SortFunc(route, func(a, b stopRow) int {
			return cmp.Or(
				cmp.Compare(a.MoveSequence, b.MoveSequence),
				cmp.Compare(a.Sequence, b.Sequence),
			)
		})
		measures := make(map[board.Measure]board.RowMeasure, 2)
		if row.Revenue.Valid {
			measures[board.MeasureRevenue] = board.RowMeasure{
				Measure: board.MeasureRevenue,
				Value:   board.Value{Kind: board.ValueMoney, Number: row.Revenue.Decimal},
			}
		}
		if late, ok := lateness(route, now); ok {
			measures[board.MeasureLateness] = late
		}
		out[row.ID] = boardassemble.Summary{
			Label:    strings.TrimSpace(row.Label),
			Subtitle: lane(route),
			Status:   row.Status,
			Measures: measures,
		}
	}

	return out, nil
}

func pickup(stop *stopRow) bool {
	return stop.Type == string(shipmentvocab.StopTypePickup) ||
		stop.Type == string(shipmentvocab.StopTypeSplitPickup)
}

func place(stop *stopRow) string {
	return strings.Join(slices.DeleteFunc([]string{
		strings.TrimSpace(stop.City), strings.TrimSpace(stop.State),
	}, func(part string) bool { return part == "" }), ", ")
}

func lane(route []stopRow) string {
	var from, to string
	for idx := range route {
		if pickup(&route[idx]) {
			if from == "" {
				from = place(&route[idx])
			}
			continue
		}
		to = place(&route[idx])
	}
	switch {
	case from != "" && to != "":
		return from + " → " + to
	default:
		return cmp.Or(from, to)
	}
}

func windowEnd(stop *stopRow) int64 {
	if stop.WindowEnd != nil && *stop.WindowEnd > 0 {
		return *stop.WindowEnd
	}

	return stop.WindowStart
}

func lateness(route []stopRow, now int64) (board.RowMeasure, bool) {
	measure := board.RowMeasure{
		Measure: board.MeasureLateness,
		Value:   board.Value{Kind: board.ValueDuration, Number: decimal.Zero},
	}
	for idx := range route {
		stop := &route[idx]
		if stop.Status == string(shipmentvocab.StopStatusCompleted) {
			continue
		}
		due := windowEnd(stop)
		if due <= 0 {
			return board.RowMeasure{}, false
		}
		measure.At = due
		measure.Place = place(stop)
		switch {
		case now <= due:
			ahead(&measure, stop.Estimate, due)
		case stop.Arrival != nil && *stop.Arrival > 0:
			measure.State = board.MeasureHeld
			measure.Tone = board.ToneWarn
			measure.Value.Number = decimal.NewFromInt(now - due)
		default:
			measure.State = board.MeasureLate
			measure.Tone = board.ToneWarn
			measure.Value.Number = decimal.NewFromInt(now - due)
		}
		return measure, true
	}

	for idx := len(route) - 1; idx >= 0; idx-- {
		stop := &route[idx]
		if pickup(stop) || stop.Arrival == nil || *stop.Arrival <= 0 {
			continue
		}
		due := windowEnd(stop)
		measure.State = board.MeasureDelivered
		measure.At = *stop.Arrival
		measure.Place = place(stop)
		measure.Tone = board.ToneOK
		if due > 0 && *stop.Arrival > due {
			measure.Value.Number = decimal.NewFromInt(*stop.Arrival - due)
			measure.Tone = board.ToneInfo
		}
		return measure, true
	}

	return board.RowMeasure{}, false
}

func ahead(measure *board.RowMeasure, estimate *int64, due int64) {
	switch {
	case estimate == nil || *estimate <= 0:
		measure.State = board.MeasureDue
	case *estimate > due:
		measure.State = board.MeasureProjected
		measure.Tone = board.ToneWarn
		measure.Value.Number = decimal.NewFromInt(*estimate - due)
	default:
		measure.State = board.MeasureOnTime
		measure.Tone = board.ToneOK
	}
}

func (r *Repository) invoices(
	ctx context.Context,
	ids []pulid.ID,
	now int64,
) (map[pulid.ID]boardassemble.Summary, error) {
	rows, err := keel.Into[invoiceRow](ctx, r.db.Keel(ctx), invoicesQuery(ids))
	if err != nil {
		return nil, err
	}

	out := make(map[pulid.ID]boardassemble.Summary, len(rows))
	for _, row := range rows {
		due := board.RowMeasure{
			Measure: board.MeasureAmountDue,
			Value:   money(row.BalanceDue, row.Currency),
			State:   board.MeasureOpen,
		}
		switch {
		case row.BalanceDue <= 0:
			due.State = board.MeasureSettled
			due.Tone = board.ToneOK
		case row.DueDate != nil && *row.DueDate > 0 && *row.DueDate < now:
			due.Tone = board.ToneWarn
			due.At = *row.DueDate
		case row.DueDate != nil:
			due.At = *row.DueDate
		}
		measures := make(map[board.Measure]board.RowMeasure, 3)
		measures[board.MeasureAmountDue] = due
		measures[board.MeasureTotal] = board.RowMeasure{
			Measure: board.MeasureTotal,
			Value:   money(row.TotalMinor, row.Currency),
		}
		if row.InvoiceDate > 0 {
			measures[board.MeasureAge] = board.RowMeasure{
				Measure: board.MeasureAge,
				Value:   duration(now - row.InvoiceDate),
				At:      row.InvoiceDate,
			}
		}
		out[row.ID] = boardassemble.Summary{
			Label:    strings.TrimSpace(row.Label),
			Subtitle: strings.TrimSpace(row.Subtitle),
			Status:   row.Status,
			Measures: measures,
		}
	}

	return out, nil
}

func (r *Repository) entries(
	ctx context.Context,
	ids []pulid.ID,
	now int64,
) (map[pulid.ID]boardassemble.Summary, error) {
	rows, err := keel.Into[entryRow](ctx, r.db.Keel(ctx), entriesQuery(ids))
	if err != nil {
		return nil, err
	}

	out := make(map[pulid.ID]boardassemble.Summary, len(rows))
	for _, row := range rows {
		pending := board.RowMeasure{
			Measure: board.MeasurePendingFor,
			Value:   duration(0),
			State:   board.MeasureSettled,
			At:      row.CreatedAt,
			Tone:    board.ToneOK,
		}
		if waiting(row.Status) {
			pending.Value = duration(now - row.CreatedAt)
			pending.State = board.MeasureOpen
			pending.Tone = board.ToneWarn
		}
		measures := make(map[board.Measure]board.RowMeasure, 2)
		measures[board.MeasureAmount] = board.RowMeasure{
			Measure: board.MeasureAmount,
			Value:   money(row.Debit, ""),
		}
		measures[board.MeasurePendingFor] = pending
		out[row.ID] = boardassemble.Summary{
			Label:    strings.TrimSpace(row.Label),
			Subtitle: strings.TrimSpace(row.Subtitle),
			Status:   row.Status,
			Measures: measures,
		}
	}

	return out, nil
}

func waiting(status string) bool {
	return status == string(journalentry.StatusDraft) ||
		status == string(journalentry.StatusPending) ||
		status == string(journalentry.StatusApproved)
}

func money(minor int64, currency string) board.Value {
	return board.Value{
		Kind:     board.ValueMoney,
		Number:   decimal.New(minor, minorExponent),
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

func duration(seconds int64) board.Value {
	return board.Value{Kind: board.ValueDuration, Number: decimal.NewFromInt(max(seconds, 0))}
}

func distinct(ids []pulid.ID) []pulid.ID {
	out := make([]pulid.ID, 0, min(len(ids), board.MaxBoardRecords))
	for _, id := range ids {
		if id.IsNotNil() && !slices.Contains(out, id) {
			out = append(out, id)
		}
		if len(out) == board.MaxBoardRecords {
			break
		}
	}

	return out
}
