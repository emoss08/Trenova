package operationsboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrollup"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionboard"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence/detentionevidencedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const stayMinute = 60

type DetentionStays struct{ db *postgres.Connection }

type Detention struct{ db *postgres.Connection }

type PastFree struct{ db *postgres.Connection }

type DriverPay struct{ db *postgres.Connection }

func NewDetentionStays(p Params) *DetentionStays { return &DetentionStays{db: p.DB} }

func NewDetention(p Params) *Detention { return &Detention{db: p.DB} }

func NewPastFree(p Params) *PastFree { return &PastFree{db: p.DB} }

func NewDriverPay(p Params) *DriverPay { return &DriverPay{db: p.DB} }

func (*DetentionStays) Kind() recordkind.Kind { return recordkind.DetentionOccurrence }

func (*Detention) Kind() recordkind.Kind { return recordkind.DetentionOccurrence }

func (*Detention) Measure() string { return detentionboard.MeasureDetention }

func (*PastFree) Kind() recordkind.Kind { return recordkind.DetentionOccurrence }

func (*PastFree) Measure() string { return detentionboard.MeasurePastFree }

func (*DriverPay) Kind() recordkind.Kind { return recordkind.DetentionOccurrence }

func (*DriverPay) Measure() string { return detentionboard.MeasureDriverPay }

type stayRow struct {
	ID                 pulid.ID
	ProNumber          string
	LocationName       string
	Place              string
	Status             string
	NotificationStatus string
	NoticeDeadlineAt   *int64
	IsOpen             bool
	ClockStartAt       int64
	FreeMinutes        int64
	DwellMinutes       int64
	PastFreeMinutes    int64
	BillableAmount     decimal.Decimal
	DriverPayAmount    decimal.NullDecimal
	Currency           string
	ArrivalByHand      bool
	DepartureByHand    bool
}

func readStays(
	ctx context.Context,
	db *postgres.Connection,
	ids []pulid.ID,
) ([]stayRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	return keel.Into[stayRow](ctx, db.Keel(ctx), stayQuery(ids))
}

func stayQuery(ids []pulid.ID) keel.Query[*detention.DetentionOccurrence] {
	o := detentiondb.DetentionOccurrences
	sp := shipmentdb.Shipments
	loc := locationdb.Locations
	st := usstatedb.UsStates
	ev := detentionevidencedb.DetentionEvidences
	byHand := func(kind detentionevidence.EvidenceKind) keel.Pred {
		return keel.Exists(ev.Pick(ev.ID).Where(
			ev.DetentionOccurrenceID.EqCol(o.ID),
			ev.Kind.Eq(kind),
			ev.Source.Eq(detentionevidence.EvidenceSourceManual),
		))
	}

	return o.Join(sp, sp.ID.EqCol(o.ShipmentID)).
		LeftJoin(loc, loc.ID.EqCol(o.LocationID)).
		LeftJoin(st, st.ID.EqCol(loc.StateID)).
		Pick(
			o.ID,
			text(sp.ProNumber),
			text(loc.Name),
			text(keel.ConcatWS(", ", loc.City, st.Abbreviation)),
			text(o.Status),
			text(o.NotificationStatus),
			o.NoticeDeadlineAt,
			o.IsOpen,
			o.ClockStartAt,
			keel.CastInt64(o.FreeMinutesGranted),
			keel.CastInt64(o.RawDwellMinutes),
			detentionrollup.PastFreeMinutes(),
			o.BillableAmount,
			o.DriverPayAmount,
			text(o.Currency),
			byHand(detentionevidence.EvidenceKindArrival),
			byHand(detentionevidence.EvidenceKindDeparture),
		).
		Where(o.ID.In(ids...))
}

func (r *DetentionStays) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := readStays(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = stay(ctx, &rows[idx])
			}

			return out, nil
		},
	)
}

func stay(ctx context.Context, row *stayRow) boardreading.Summary {
	label := strings.TrimSpace(row.ProNumber)
	if label == "" {
		label = i18n.T(ctx, "Detention stay")
	}
	subtitle := strings.TrimSpace(row.Place)
	if subtitle == "" {
		subtitle = strings.TrimSpace(row.LocationName)
	}

	status, tone := stayStatus(ctx, row.Status)

	return boardreading.Summary{
		Label:      label,
		Subtitle:   subtitle,
		Status:     status,
		StatusTone: tone,
	}
}

func stayStatus(ctx context.Context, status string) (string, boardcatalog.Tone) {
	switch detention.OccurrenceStatus(status) {
	case detention.OccurrenceStatusAccruing:
		return i18n.T(ctx, "At the dock"), boardcatalog.ToneInfo
	case detention.OccurrenceStatusPending:
		return i18n.T(ctx, "Waiting on approval"), boardcatalog.ToneWarn
	case detention.OccurrenceStatusApproved:
		return i18n.T(ctx, "Approved"), boardcatalog.ToneInfo
	case detention.OccurrenceStatusBilled:
		return i18n.T(ctx, "Billed"), boardcatalog.ToneOK
	case detention.OccurrenceStatusDisputed:
		return i18n.T(ctx, "Disputed"), boardcatalog.ToneWarn
	case detention.OccurrenceStatusWaived:
		return i18n.T(ctx, "Waived"), ""
	case detention.OccurrenceStatusNotBillable:
		return i18n.T(ctx, "Not billable"), ""
	}

	return status, ""
}

func (r *Detention) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readStays(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = billable(ctx, &rows[idx], req.Now)
			}

			return out, nil
		},
	)
}

func (r *PastFree) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readStays(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}

			return pastFree(ctx, rows, req.Now), nil
		},
	)
}

func pastFree(ctx context.Context, rows []stayRow, now int64) map[pulid.ID]boardreading.Reading {
	out := make(map[pulid.ID]boardreading.Reading, len(rows))
	for idx := range rows {
		row := &rows[idx]
		dwell, past := onSite(row, now)
		value := boardcatalog.Duration(past * stayMinute)
		on := boardcatalog.Duration(dwell * stayMinute)
		reading := boardreading.Reading{
			Value:    value,
			Standing: boardreading.Words{Text: slot, Value: &value},
			Fact:     &boardreading.Words{Text: i18n.T(ctx, "{0} on site"), Value: &on},
		}
		if past > 0 {
			reading.Tone = boardcatalog.ToneWarn
		}
		out[row.ID] = reading
	}

	return out
}

func (r *DriverPay) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if !req.Visible.Has(protectedfield.Margins) {
		return map[pulid.ID]boardreading.Reading{}, nil
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := readStays(ctx, r.db, req.IDs)
			if err != nil {
				return nil, err
			}

			return driverPay(rows, req.Visible), nil
		},
	)
}

func driverPay(
	rows []stayRow,
	visible protectedfield.GroupSet,
) map[pulid.ID]boardreading.Reading {
	out := make(map[pulid.ID]boardreading.Reading, len(rows))
	if !visible.Has(protectedfield.Margins) {
		return out
	}
	for idx := range rows {
		row := &rows[idx]
		if !row.DriverPayAmount.Valid {
			continue
		}
		value := boardcatalog.Amount(row.DriverPayAmount.Decimal, row.Currency)
		out[row.ID] = boardreading.Reading{
			Value:    value,
			Standing: boardreading.Words{Text: slot, Value: &value},
		}
	}

	return out
}

func onSite(row *stayRow, now int64) (dwell, past int64) {
	dwell = row.DwellMinutes
	past = row.PastFreeMinutes
	if row.IsOpen && row.ClockStartAt > 0 && now > row.ClockStartAt {
		dwell = (now - row.ClockStartAt) / stayMinute
		past = max(dwell-row.FreeMinutes, 0)
	}

	return dwell, past
}

func billable(ctx context.Context, row *stayRow, now int64) boardreading.Reading {
	dwell, _ := onSite(row, now)
	on := boardcatalog.Duration(dwell * stayMinute)
	bounded := &boardreading.BoundedFacts{
		Value: decimal.NewFromInt(dwell),
		Bound: decimal.NewFromInt(row.FreeMinutes),
	}
	if row.IsOpen {
		value := boardcatalog.Money(0, row.Currency)

		return boardreading.Reading{
			Value:    value,
			Standing: boardreading.Words{Text: i18n.T(ctx, "Still on site")},
			Fact:     &boardreading.Words{Text: i18n.T(ctx, "{0} on site so far"), Value: &on},
			Tone:     boardcatalog.ToneInfo,
			Bounded:  bounded,
		}
	}
	value := boardcatalog.Amount(row.BillableAmount, row.Currency)
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
		Fact:     &boardreading.Words{Text: i18n.T(ctx, "{0} on site"), Value: &on},
		Bounded:  bounded,
	}
	if held := heldBack(ctx, row, now); held != "" {
		reading.Fact = &boardreading.Words{Text: held}
		reading.Tone = boardcatalog.ToneWarn
	}

	return reading
}

func heldBack(ctx context.Context, row *stayRow, now int64) string {
	facts := detention.HoldBackFacts{
		Status:             detention.OccurrenceStatus(row.Status),
		NotificationStatus: detention.NotificationStatus(row.NotificationStatus),
		NoticeDeadlineAt:   row.NoticeDeadlineAt,
		ArrivalByHand:      row.ArrivalByHand,
		DepartureByHand:    row.DepartureByHand,
	}
	switch facts.HoldBack(now) {
	case detention.HoldBackNoticeMissed:
		return i18n.T(ctx, "Notice missed")
	case detention.HoldBackNoticeFailed:
		return i18n.T(ctx, "Notice never reached the customer")
	case detention.HoldBackNoticeLate:
		return i18n.T(ctx, "Notice sent late")
	case detention.HoldBackDepartureByHand:
		return i18n.T(ctx, "Departure entered by hand")
	case detention.HoldBackArrivalByHand:
		return i18n.T(ctx, "Arrival entered by hand")
	case detention.HoldBackAwaitingApproval, detention.HoldBackNone:
	}

	return ""
}
