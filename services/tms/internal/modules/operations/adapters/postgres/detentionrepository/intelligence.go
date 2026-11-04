package detentionrepository

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrollup"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy/detentionpolicydb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxPolicyUses bounds the policy-change read: it lists only facilities that used more than one
// policy in the window, and a facility changes policy rarely.
const maxPolicyUses = 500

type IntelligenceParams struct {
	fx.In

	DB       *postgres.Connection
	Logger   *zap.Logger
	Invoices detentionports.InvoiceSettlementColumns
}

// intelligenceRepository reads detention intelligence from the detention rollups, which add the
// settled stops up by organization-local day, so a read costs the days of its window rather than
// its stops. Two figures a sum cannot carry are read live from the stops, each through an index
// on its one status: the distinct people who waived, and whether a billed charge reached a posted
// or paid invoice. A read narrowed to one facility reads its customers live as well, from that
// facility's window alone.
type intelligenceRepository struct {
	db       *postgres.Connection
	l        *zap.Logger
	invoices detentionports.InvoiceSettlementColumns
}

func NewIntelligenceRepository(
	p IntelligenceParams,
) detentionports.DetentionIntelligenceRepository {
	return &intelligenceRepository{
		db:       p.DB,
		l:        p.Logger.Named("postgres.detention-intelligence-repository"),
		invoices: p.Invoices,
	}
}

func (r *intelligenceRepository) OrganizationTimezone(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		org := tenantdb.Organizations
		var zone string
		err := keel.Batch(ctx, r.db.Keel(ctx), keel.IntoValue(
			org.Pick(org.Timezone).
				Where(org.ID.Eq(tenant.OrgID), org.BusinessUnitID.Eq(tenant.BuID)),
			&zone,
		))
		if err != nil {
			r.l.Error("failed to read the organization's time zone", zap.Error(err))
			return "", err
		}

		return zone, nil
	})
}

func (r *intelligenceRepository) Intelligence(
	ctx context.Context,
	req *detentionports.IntelligenceRequest,
) (*detentionports.IntelligenceRollup, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionports.IntelligenceRollup, error) {
			out := &detentionports.IntelligenceRollup{}
			current := span{IntelligenceWindow: req.Current, location: req.LocationID}
			previous := span{IntelligenceWindow: req.Previous, location: req.LocationID}
			figures := figuresTable(req.LocationID)

			var facilities []facilityRow
			var splits splitsRow
			var arrivals []arrivalRow
			var recovery recoveryRow
			var invoiced invoicedRow
			var uses []policyUseRow
			items := []keel.BatchItem{
				keel.IntoValue(r.totals(figures, current), &out.Totals),
				keel.IntoValue(r.totals(figures, previous), &out.Previous),
				keel.IntoSlice(r.facilities(current, req.Limit), &facilities),
				keel.IntoSlice(r.facilityMinutes(previous, req.Limit), &out.PreviousFacilities),
				keel.IntoSlice(r.waivers(current), &out.Waivers),
				keel.IntoValue(r.recovery(figures, current), &recovery),
				keel.IntoValue(r.invoiced(current), &invoiced),
				r.arrivals(current, req.Timezone, &arrivals, &out.Arrivals),
				keel.IntoValue(r.splits(figures, current), &splits),
				keel.IntoSlice(r.days(figures, current), &out.Days),
				keel.IntoSlice(r.policyUses(current), &uses),
			}
			if req.LocationID.IsNotNil() {
				items = append(
					items,
					keel.IntoSlice(r.liveCustomers(current, req.Limit), &out.Customers),
				)
			} else {
				items = append(items, keel.IntoSlice(r.customers(current, req.Limit), &out.Customers))
			}

			if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
				r.l.Error("failed to read detention intelligence", zap.Error(err))
				return nil, err
			}

			out.Facilities = facilityIntelligence(facilities)
			out.Recovery = recovery.buckets(&invoiced)
			out.Splits = splits.splits()
			if req.LocationID.IsNil() {
				out.Arrivals = arrivalCells(arrivals)
			}
			out.PolicyUses = policyUses(uses)

			return out, nil
		},
	)
}

// span is one window of whole organization-local days, narrowed to a facility when location is
// set.
type span struct {
	detentionports.IntelligenceWindow
	location pulid.ID
}

// days keeps the daily rollup rows of the span's days: a row's bucket is its day's local midnight,
// so the last day's bucket falls before the span's last instant.
func (s span) days(t *detentionrollup.Table) []keel.Pred {
	return append([]keel.Pred{
		t.Bucket().Gte(time.Unix(s.From, 0)),
		t.Bucket().Lte(time.Unix(s.To, 0)),
	}, s.at(t)...)
}

// at narrows a rollup that keeps facilities to the span's one, when it names one.
func (s span) at(t *detentionrollup.Table) []keel.Pred {
	if s.location.IsNil() || !t.Has(detentionrollup.Location) {
		return nil
	}

	return []keel.Pred{keel.StringOf(t.Col(detentionrollup.Location)).Eq(s.location.String())}
}

// view reads the span's whole months from monthly and the days around them from daily, as one
// table of columns.
func (s span) view(
	name string,
	monthly, daily *detentionrollup.Table,
	columns ...string,
) *detentionrollup.View {
	return detentionrollup.Union(name, monthly, daily, s.IntelligenceWindow, columns, s.at)
}

// settled keeps the settled stops whose clock started in the span: the live reads.
// idx_detention_occurrences_settled_status_window serves a read of one status and
// idx_detention_occurrences_location_window a facility's.
func (s span) settled(o *detentiondb.DetentionOccurrencesTable) []keel.Pred {
	return []keel.Pred{
		o.IsOpen.IsFalse(),
		o.ClockStartAt.Gte(s.From),
		o.ClockStartAt.Lte(s.To),
		keel.When(s.location.IsNotNil(), o.LocationID.Eq(s.location)),
	}
}

// figuresTable is where a read's figures come from: the organization's days, or one facility's.
func figuresTable(location pulid.ID) *detentionrollup.Table {
	if location.IsNotNil() {
		return detentionrollup.FacilityDailyTable
	}

	return detentionrollup.DailyTable
}

// figureColumns are the rollup columns figurePicks sums.
var figureColumns = []string{
	detentionrollup.Stops,
	detentionrollup.DetainedStops,
	detentionrollup.PastFree,
	detentionrollup.DockMinutes,
	detentionrollup.OpenMinutes,
	detentionrollup.EarlyMinutes,
	detentionrollup.LateMinutes,
	detentionrollup.Billable,
	detentionrollup.DriverPay,
	detentionrollup.Net,
	detentionrollup.Waived,
	detentionrollup.Disputes,
	detentionrollup.Suppressed,
	detentionrollup.NoticesDue,
	detentionrollup.NoticesKept,
}

func figurePicks(t detentionrollup.Source) []keel.Expr {
	return []keel.Expr{
		t.Count(detentionrollup.Stops),
		t.Count(detentionrollup.DetainedStops),
		t.Count(detentionrollup.PastFree),
		t.Count(detentionrollup.DockMinutes),
		t.Count(detentionrollup.OpenMinutes),
		t.Count(detentionrollup.EarlyMinutes),
		t.Count(detentionrollup.LateMinutes),
		t.Amount(detentionrollup.Billable),
		t.Amount(detentionrollup.DriverPay),
		t.Amount(detentionrollup.Net),
		t.Amount(detentionrollup.Waived),
		t.Count(detentionrollup.Disputes),
		t.Count(detentionrollup.Suppressed),
		t.Count(detentionrollup.NoticesDue),
		t.Count(detentionrollup.NoticesKept),
	}
}

func (r *intelligenceRepository) totals(t *detentionrollup.Table, s span) keel.Projection {
	return keel.RowsOf(t.Ref).Pick(figurePicks(t)...).Where(s.days(t)...)
}

// facilityRow is a facility's figures with its dwell and free-time histograms, from which its
// quantiles are read.
type facilityRow struct {
	LocationID        pulid.ID
	LocationName      string
	LocationCity      string
	LocationState     string
	StopCount         int
	DetainedCount     int
	PastFreeMinutes   int64
	DockMinutes       int64
	OpenMinutes       int64
	EarlyMinutes      int64
	LateMinutes       int64
	BilledAmount      decimal.Decimal
	DriverPayAmount   decimal.NullDecimal
	NetMargin         decimal.NullDecimal
	WaivedAmount      decimal.Decimal
	DisputeCount      int
	SuppressedCount   int
	NoticeDueCount    int
	NoticeOnTimeCount int
	DwellMinutes      int64
	DwellBins         []int64
	FreeBins          []int64
}

func bins(t detentionrollup.Source, name func(int) string, n int) keel.Int64ArrayExpr {
	sums := make([]keel.Int64Value, n)
	for i := range n {
		sums[i] = t.Count(name(i))
	}

	return keel.ArrayOfInt64(sums...)
}

func names(name func(int) string, n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = name(i)
	}

	return out
}

// facilities ranks the window's facilities by their hours past free time on their plain sums,
// then sums the dwell and free-time bins and joins the names for the ranked ones alone: the
// histograms are most of the columns, and a list shows a hundred facilities of thousands.
func (r *intelligenceRepository) facilities(s span, limit int) keel.Projection {
	columns := append([]string{detentionrollup.Location}, figureColumns...)
	columns = append(columns, detentionrollup.DwellTotal)
	columns = append(columns, names(detentionrollup.DwellBin, len(detentionrollup.DwellBounds))...)
	columns = append(columns, names(detentionrollup.FreeBin, len(detentionrollup.FreeBounds))...)
	v := s.view("detention_facility_window",
		detentionrollup.FacilityMonthlyTable, detentionrollup.FacilityDailyTable, columns...)
	location := v.Col(detentionrollup.Location)

	rankedColumns := append([]string{detentionrollup.Location}, figureColumns...)
	rankedColumns = append(rankedColumns, detentionrollup.DwellTotal)
	rankedPicks := append([]keel.Expr{location}, figurePicks(v)...)
	rankedPicks = append(rankedPicks, v.Count(detentionrollup.DwellTotal))
	rank := v.Rows().
		Group(location).
		Pick(rankedPicks...).
		Order(keel.Desc(v.Count(detentionrollup.PastFree)), location.Asc())
	if limit > 0 {
		rank = rank.Limit(limit)
	}
	ranked := keel.CTE("detention_facility_ranked", rankedColumns, rank)
	rankedAt := ranked.Col(detentionrollup.Location)

	histograms := keel.CTE("detention_facility_bins",
		[]string{detentionrollup.Location, "dwell_bins", "free_bins"},
		v.Rows().
			Group(location).
			Pick(
				location,
				bins(v, detentionrollup.DwellBin, len(detentionrollup.DwellBounds)),
				bins(v, detentionrollup.FreeBin, len(detentionrollup.FreeBounds)),
			).
			Where(location.InQuery(ranked.Query().Pick(rankedAt))),
	)

	loc := locationdb.Locations
	st := usstatedb.UsStates
	picks := []keel.Expr{
		rankedAt,
		loc.Name.Coalesce(""),
		loc.City.Coalesce(""),
		st.Abbreviation.Coalesce(""),
	}
	for _, name := range rankedColumns[1:] {
		picks = append(picks, ranked.Col(name))
	}
	picks = append(picks, histograms.Col("dwell_bins"), histograms.Col("free_bins"))

	return ranked.Query().
		With(v.CTE(), ranked, histograms).
		LeftJoin(histograms, histograms.Col(detentionrollup.Location).EqCol(rankedAt)).
		LeftJoin(loc, loc.ID.EqCol(rankedAt)).
		LeftJoin(st, st.ID.EqCol(loc.StateID)).
		Pick(picks...).
		Order(keel.Desc(ranked.Col(detentionrollup.PastFree)), rankedAt.Asc())
}

func facilityIntelligence(rows []facilityRow) []detentionports.FacilityIntelligence {
	out := make([]detentionports.FacilityIntelligence, len(rows))
	for i := range rows {
		row := &rows[i]
		avg := 0
		if row.StopCount > 0 {
			avg = int((row.DwellMinutes + int64(row.StopCount)/2) / int64(row.StopCount))
		}
		out[i] = detentionports.FacilityIntelligence{
			LocationID:         row.LocationID,
			LocationName:       row.LocationName,
			LocationCity:       row.LocationCity,
			LocationState:      row.LocationState,
			StopCount:          row.StopCount,
			DetainedCount:      row.DetainedCount,
			PastFreeMinutes:    row.PastFreeMinutes,
			DockMinutes:        row.DockMinutes,
			OpenMinutes:        row.OpenMinutes,
			EarlyMinutes:       row.EarlyMinutes,
			LateMinutes:        row.LateMinutes,
			AvgDwellMinutes:    avg,
			MedianDwellMinutes: rollup.Quantile(row.DwellBins, detentionrollup.DwellBounds, 0.5),
			P90DwellMinutes:    rollup.Quantile(row.DwellBins, detentionrollup.DwellBounds, 0.9),
			MedianFreeMinutes:  rollup.Quantile(row.FreeBins, detentionrollup.FreeBounds, 0.5),
			BilledAmount:       row.BilledAmount,
			DriverPayAmount:    row.DriverPayAmount,
			NetMargin:          row.NetMargin,
			WaivedAmount:       row.WaivedAmount,
			DisputeCount:       row.DisputeCount,
			SuppressedCount:    row.SuppressedCount,
			NoticeDueCount:     row.NoticeDueCount,
			NoticeOnTimeCount:  row.NoticeOnTimeCount,
		}
	}

	return out
}

func (r *intelligenceRepository) facilityMinutes(s span, limit int) keel.Projection {
	v := s.view("detention_facility_before",
		detentionrollup.FacilityMonthlyTable, detentionrollup.FacilityDailyTable,
		detentionrollup.Location, detentionrollup.PastFree)
	location := v.Col(detentionrollup.Location)
	minutes := v.Count(detentionrollup.PastFree)
	q := v.Query().
		Group(location).
		Pick(location, minutes).
		Order(keel.Desc(minutes), location.Asc())
	if limit > 0 {
		// The previous window can rank facilities differently, so read deeper than the current
		// list to find most of the same facilities.
		q = q.Limit(limit * 2)
	}

	return q
}

func (r *intelligenceRepository) customers(s span, limit int) keel.Projection {
	v := s.view("detention_customer_window",
		detentionrollup.CustomerMonthlyTable, detentionrollup.CustomerDailyTable,
		append([]string{detentionrollup.Customer}, figureColumns...)...)
	cus := customerdb.Customers
	customer := v.Col(detentionrollup.Customer)

	q := v.Query().
		LeftJoin(cus, cus.ID.EqCol(customer)).
		Group(customer, cus.Name).
		Pick(
			customer,
			cus.Name.Coalesce(""),
			v.Count(detentionrollup.Stops),
			v.Count(detentionrollup.DetainedStops),
			v.Count(detentionrollup.PastFree),
			v.Amount(detentionrollup.Billable),
			v.Amount(detentionrollup.DriverPay),
			v.Amount(detentionrollup.Net),
			v.Amount(detentionrollup.Waived),
			v.Count(detentionrollup.Disputes),
			v.Count(detentionrollup.NoticesDue),
			v.Count(detentionrollup.NoticesKept),
		).
		Order(keel.Desc(v.Count(detentionrollup.PastFree)), customer.Asc())
	if limit > 0 {
		q = q.Limit(limit)
	}

	return q
}

func minutesWhere(p keel.Pred) keel.Int64Expr {
	agg := keel.Sum(detentionrollup.PastFreeMinutes())
	if p != nil {
		agg = agg.Filter(p)
	}
	return keel.CastInt64(keel.NumericOf(agg).Coalesce(0))
}

// liveCustomers is one facility's customers, read from its stops: the customer rollup is not
// kept by facility, and one facility's window is a short read through its location index.
func (r *intelligenceRepository) liveCustomers(s span, limit int) keel.Projection {
	o := detentiondb.DetentionOccurrences
	cus := customerdb.Customers

	q := o.LeftJoin(cus, cus.ID.EqCol(o.CustomerID)).
		Group(o.CustomerID, cus.Name).
		Pick(
			o.CustomerID,
			cus.Name.Coalesce(""),
			keel.Count(),
			keel.Count().Filter(detentionrollup.Detained()),
			minutesWhere(nil),
			sumOrZero(o.BillableAmount),
			sumOrZero(o.DriverPayAmount),
			sumOrZero(o.NetMargin),
			sumOrZero(o.WaivedAmount),
			keel.Count().Filter(o.Status.Eq(detention.OccurrenceStatusDisputed)),
			keel.Count().Filter(detentionrollup.NoticeDue()),
			keel.Count().Filter(detentionrollup.NoticeOnTime()),
		).
		Where(s.settled(o)...).
		Order(keel.Desc(minutesWhere(nil)), o.CustomerID.Asc())
	if limit > 0 {
		q = q.Limit(limit)
	}

	return q
}

func (r *intelligenceRepository) waivers(s span) keel.Projection {
	o := detentiondb.DetentionOccurrences
	return o.Group(o.WaiverReason).
		Pick(
			keel.CastText(o.WaiverReason).Coalesce(""),
			keel.Count(),
			o.WaivedByID.CountDistinct(),
			sumOrZero(o.WaivedAmount),
		).
		Where(append(s.settled(o), o.Status.Eq(detention.OccurrenceStatusWaived))...).
		Order(keel.Desc(sumOrZero(o.WaivedAmount)))
}

// recoveryRow holds, for each recovery bucket in detentionrollup.RecoveryBuckets order, its stops,
// gross and billable amounts.
type recoveryRow struct {
	Stops    []int64
	Gross    []decimal.Decimal
	Billable []decimal.Decimal
}

func (r *intelligenceRepository) recovery(t *detentionrollup.Table, s span) keel.Projection {
	n := len(detentionrollup.RecoveryBuckets)
	stops := make([]keel.Int64Value, n)
	gross := make([]keel.NumericValue, n)
	billable := make([]keel.NumericValue, n)
	for i, b := range detentionrollup.RecoveryBuckets {
		stops[i] = t.Count(detentionrollup.RecoveryCount(b))
		gross[i] = t.Amount(detentionrollup.RecoveryGross(b))
		billable[i] = t.Amount(detentionrollup.RecoveryBillable(b))
	}

	return keel.RowsOf(t.Ref).
		Pick(
			keel.ArrayOfInt64(stops...),
			keel.ArrayOfNumeric(gross...),
			keel.ArrayOfNumeric(billable...),
		).
		Where(s.days(t)...)
}

func (row *recoveryRow) buckets(invoiced *invoicedRow) []detentionports.RecoveryBucket {
	out := make([]detentionports.RecoveryBucket, 0, len(detentionrollup.RecoveryBuckets))
	for i, b := range detentionrollup.RecoveryBuckets {
		if i >= len(row.Stops) || row.Stops[i] == 0 {
			continue
		}
		bucket := detentionports.RecoveryBucket{
			Bucket:          b,
			OccurrenceCount: int(row.Stops[i]),
			GrossAmount:     row.Gross[i],
			BillableAmount:  row.Billable[i],
		}
		if b == detentionports.RecoveryBilled {
			bucket.InvoicedAmount = invoiced.Invoiced
			bucket.CollectedAmount = invoiced.Collected
		}
		out = append(out, bucket)
	}

	return out
}

// invoicedRow is what of the window's billed charges reached a posted invoice and a paid one.
type invoicedRow struct {
	Invoiced  decimal.Decimal
	Collected decimal.Decimal
}

// invoiced reads the invoice state of the window's billed charges live: an invoice is posted and
// paid long after its charge settled, so no rollup of the charge could keep it. The state is read
// once per charge, in a CTE limited to the window's billed charges, and joined: every table under
// row-level security pays its policy checks per reference, and invoices carry the heaviest.
func (r *intelligenceRepository) invoiced(s span) keel.Projection {
	o := detentiondb.DetentionOccurrences
	inv := r.invoices
	billed := append(s.settled(o),
		o.Status.Eq(detention.OccurrenceStatusBilled),
		o.AdditionalChargeID.IsNotNull(),
	)
	scoped := o.As("billed")
	charges := keel.CTE(
		"detention_invoice_state",
		[]string{"charge_id", "posted_lines", "paid_lines"},
		keel.RowsOf(inv.LineChargeID.Ref).
			Join(keel.TableBase{Ref: inv.InvoiceID.Ref}, inv.InvoiceID.EqCol(inv.LineInvoiceID)).
			Group(inv.LineChargeID).
			Pick(
				inv.LineChargeID,
				keel.Count().Filter(inv.InvoicePosted),
				keel.Count().Filter(inv.InvoicePaid),
			).
			Where(inv.LineChargeID.InQuery(
				scoped.Pick(scoped.AdditionalChargeID).Where(append(
					s.settled(scoped),
					scoped.Status.Eq(detention.OccurrenceStatusBilled),
					scoped.AdditionalChargeID.IsNotNull(),
				)...),
			)),
	)

	return o.Query().
		With(charges).
		Join(charges, charges.Col("charge_id").EqCol(o.AdditionalChargeID)).
		Pick(
			amountWhere(o.BillableAmount, charges.Col("posted_lines").Gt(0)),
			amountWhere(o.BillableAmount, charges.Col("paid_lines").Gt(0)),
		).
		Where(billed...)
}

func amountWhere(column keel.Expr, p keel.Pred) keel.NumericExpr {
	return keel.NumericOf(keel.Sum(column).Filter(p)).Coalesce(0)
}

// arrivalRow is one local weekday's arrivals by hour.
type arrivalRow struct {
	Weekday  int
	Arrivals []int64
	Detained []int64
}

// arrivals counts stops by the local weekday and hour they arrived. A read of the organization
// sums its arrival days by weekday; a read of one facility counts that facility's stops, a short
// read through its location index, since the arrival rollup is not kept by facility.
func (r *intelligenceRepository) arrivals(
	s span,
	zone string,
	days *[]arrivalRow,
	cells *[]detentionports.ArrivalCell,
) keel.BatchItem {
	if s.location.IsNotNil() {
		return keel.IntoSlice(r.liveArrivals(s, zone), cells)
	}

	t := detentionrollup.ArrivalDailyTable
	weekday := keel.CastInt64(keel.DatePart("isodow", t.Bucket().AtTimeZone(keel.Text(zone))))
	return keel.IntoSlice(keel.RowsOf(t.Ref).
		Group(weekday).
		Pick(
			weekday,
			bins(t, detentionrollup.ArrivalsAt, detentionrollup.HoursPerDay),
			bins(t, detentionrollup.DetainedArrivalsAt, detentionrollup.HoursPerDay),
		).
		Where(s.days(t)...).
		Order(weekday.Asc()), days)
}

func (r *intelligenceRepository) liveArrivals(s span, zone string) keel.Projection {
	o := detentiondb.DetentionOccurrences
	local := keel.ToTimestamp(o.ArrivedAt).AtTimeZone(keel.Text(zone))
	weekday := keel.CastInt64(keel.DatePart("isodow", local))
	hour := keel.CastInt64(keel.DatePart("hour", local))
	return o.Group(weekday, hour).
		Pick(weekday, hour, keel.Count(), keel.Count().Filter(detentionrollup.Detained())).
		Where(
			o.IsOpen.IsFalse(),
			o.LocationID.Eq(s.location),
			o.ArrivedAt.Gte(s.From),
			o.ArrivedAt.Lte(s.To),
		).
		Order(weekday.Asc(), hour.Asc())
}

// arrivalCells spreads each weekday's hours into the grid's cells, leaving out a cell no stop
// arrived in.
func arrivalCells(rows []arrivalRow) []detentionports.ArrivalCell {
	out := make([]detentionports.ArrivalCell, 0, len(rows)*detentionrollup.HoursPerDay)
	for _, row := range rows {
		for hour := range min(len(row.Arrivals), len(row.Detained)) {
			if row.Arrivals[hour] == 0 {
				continue
			}
			out = append(out, detentionports.ArrivalCell{
				Weekday:       row.Weekday,
				Hour:          hour,
				StopCount:     int(row.Arrivals[hour]),
				DetainedCount: int(row.Detained[hour]),
			})
		}
	}

	return out
}

// splitsRow holds, for each split in detentionrollup.Splits order, its stops, detained stops and
// minutes past free time.
type splitsRow struct {
	Stops    []int64
	Detained []int64
	Minutes  []int64
}

func (r *intelligenceRepository) splits(t *detentionrollup.Table, s span) keel.Projection {
	return keel.RowsOf(t.Ref).
		Pick(
			bins(
				t,
				func(i int) string { return detentionrollup.SplitStops(detentionrollup.Splits[i]) },
				len(detentionrollup.Splits),
			),
			bins(
				t,
				func(i int) string { return detentionrollup.SplitDetained(detentionrollup.Splits[i]) },
				len(detentionrollup.Splits),
			),
			bins(
				t,
				func(i int) string { return detentionrollup.SplitMinutes(detentionrollup.Splits[i]) },
				len(detentionrollup.Splits),
			),
		).
		Where(s.days(t)...)
}

func (row *splitsRow) splits() detentionports.Splits {
	figure := func(split string) detentionports.SplitFigure {
		for i, name := range detentionrollup.Splits {
			if name == split && i < len(row.Stops) {
				return detentionports.SplitFigure{
					StopCount:       int(row.Stops[i]),
					DetainedCount:   int(row.Detained[i]),
					PastFreeMinutes: row.Minutes[i],
				}
			}
		}
		return detentionports.SplitFigure{}
	}

	return detentionports.Splits{
		Pickup:      figure(detentionrollup.SplitPickup),
		Delivery:    figure(detentionrollup.SplitDelivery),
		Appointment: figure(detentionrollup.SplitAppointment),
		Open:        figure(detentionrollup.SplitOpen),
		Live:        figure(detentionrollup.SplitLive),
		Drop:        figure(detentionrollup.SplitDrop),
	}
}

func (r *intelligenceRepository) days(t *detentionrollup.Table, s span) keel.Projection {
	day := t.Bucket()
	return keel.RowsOf(t.Ref).
		Group(day).
		Pick(
			day,
			t.Count(detentionrollup.Stops),
			t.Count(detentionrollup.DetainedStops),
			t.Count(detentionrollup.PastFree),
		).
		Where(s.days(t)...).
		Order(day.Asc())
}

// policyUseRow is one policy's use at one facility, with how many policies the facility used.
type policyUseRow struct {
	LocationID      pulid.ID
	PolicyID        pulid.ID
	PolicyName      string
	FirstClockStart int64
	StopCount       int
	Policies        int
}

// policyUses lists, for each facility that used more than one detention policy in the window,
// when each policy first applied, so a change in the figures can be laid against it. Each row
// counts its facility's policies, and the facilities that changed come first, so the bound keeps
// them.
func (r *intelligenceRepository) policyUses(s span) keel.Projection {
	v := s.view("detention_policy_window",
		detentionrollup.PolicyMonthlyTable, detentionrollup.PolicyDailyTable,
		detentionrollup.Location, detentionrollup.Policy,
		detentionrollup.Stops, detentionrollup.FirstClockStart)
	pol := detentionpolicydb.DetentionPolicies
	location := v.Col(detentionrollup.Location)
	policy := v.Col(detentionrollup.Policy)
	first := keel.CastInt64(v.Col(detentionrollup.FirstClockStart).Min())
	policies := keel.CastInt64(keel.Count().Over([]keel.Expr{location}))

	return v.Query().
		LeftJoin(pol, pol.ID.EqCol(policy)).
		Group(location, policy, pol.Name).
		Pick(location, policy, pol.Name.Coalesce(""), first, v.Count(detentionrollup.Stops), policies).
		Order(keel.Desc(policies), location.Asc(), keel.Asc(first)).
		Limit(maxPolicyUses)
}

func policyUses(rows []policyUseRow) []detentionports.PolicyUse {
	out := make([]detentionports.PolicyUse, 0, len(rows))
	for _, row := range rows {
		if row.Policies < 2 {
			break
		}
		out = append(out, detentionports.PolicyUse{
			LocationID:      row.LocationID,
			PolicyID:        row.PolicyID,
			PolicyName:      row.PolicyName,
			FirstClockStart: row.FirstClockStart,
			StopCount:       row.StopCount,
		})
	}

	return out
}
