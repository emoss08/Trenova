package volumeseed

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecalculation"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	shpMinute       = int64(60)
	shpHour         = int64(3600)
	shpDay          = int64(86400)
	shpChunkSize    = 1_000
	shpFutureDays   = 14
	shpMinimumPrice = 375.0
	shpMaxWeight    = 44_000
)

const (
	shpTableOrders             = "orders"
	shpTableShipments          = "shipments"
	shpTableMoves              = "shipment_moves"
	shpTableStops              = "stops"
	shpTableAssignments        = "assignments"
	shpTableCarrierAssignments = "carrier_assignments"
	shpTableCommodities        = "shipment_commodities"
	shpTableCharges            = "additional_charges"
	shpTableComments           = "shipment_comments"
	shpTableHolds              = "shipment_holds"
)

type shpUser struct {
	ID   pulid.ID
	Name string
}

type shpLane struct {
	origin *locationRecord
	dest   *locationRecord
}

type shpStopPlan struct {
	loc *locationRecord
	typ shipment.StopType
}

type shpCommodityPlan struct {
	commodity *commodityRecord
	pieces    int64
	weight    int64
}

type shpPlan struct {
	customer    *customerRecord
	moves       [][]shpStopPlan
	createdAt   int64
	pickupAt    int64
	canceled    bool
	serviceCode string
	team        bool
	flatbed     bool
	reefer      bool
	hazmat      bool
	relay       bool
	commodities []shpCommodityPlan
	pieces      int64
	weight      int64
	detention   bool
}

type shpLookups struct {
	tractorTypeID pulid.ID
	trailerTypes  map[trailerKind]pulid.ID
	anyTrailer    pulid.ID
	serviceTypes  map[string]pulid.ID
	shipmentTypes map[string]pulid.ID
	users         []shpUser
	holdReasons   []*holdreason.HoldReason
	templates     map[string]*formulatemplate.FormulaTemplate
}

type shpBatch struct {
	orders             []*order.Order
	shipments          []*shipment.Shipment
	moves              []*shipment.ShipmentMove
	stops              []*shipment.Stop
	assignments        []*shipment.Assignment
	carrierAssignments []*shipment.CarrierAssignment
	commodities        []*shipment.ShipmentCommodity
	charges            []*shipment.AdditionalCharge
	comments           []*shipment.ShipmentComment
	mentions           []*shipment.ShipmentCommentMention
	acks               []*shipment.ShipmentCommentAcknowledgment
	holds              []*shipment.ShipmentHold
	releases           []shpRelease
	checkCalls         []*checkcall.CheckCall
	stamps             map[string][]stamp
}

func (b *shpBatch) stamp(table string, id pulid.ID, created, updated int64) {
	b.stamps[table] = append(b.stamps[table], stamp{id: id, created: created, updated: updated})
}

type shpRelease struct {
	id pulid.ID
	at int64
	by pulid.ID
}

type shpBuilder struct {
	g               *generator
	now             int64
	coord           *shipmentstate.Coordinator
	look            *shpLookups
	customerWeights []int
	customerFactor  map[pulid.ID]float64
	lanes           map[pulid.ID][]shpLane
	shippers        []*locationRecord
	consignees      []*locationRecord
	crewFree        []int64
	crewsByKind     map[trailerKind][]int
	carrierPools    map[pulid.ID][]pulid.ID
	allCrews        []int
	zones           map[string]*time.Location
	counts          map[string]int
	statuses        map[shipment.Status]int
	externalRefs    map[string]struct{}
}

func (g *generator) seedShipments(ctx context.Context) error {
	if len(g.customers) == 0 || len(g.locations) < 2 || len(g.commodities) == 0 ||
		len(g.crews) == 0 {
		return fmt.Errorf(
			"%w: customers, locations, commodities and crews must be seeded before shipments",
			ErrMissingSeedData,
		)
	}

	look, err := g.shpLoadLookups(ctx)
	if err != nil {
		return err
	}

	b := g.shpNewBuilder(look)
	plans := b.planAll(g.opts.Shipments)

	proSlots := make([]numberSlot, len(plans))
	for i, p := range plans {
		proSlots[i] = numberSlot{At: p.createdAt, LocationCode: p.moves[0][0].loc.Code}
	}
	pros, err := g.numbers.allocate(ctx, tenant.SequenceTypeProNumber, proSlots)
	if err != nil {
		return err
	}
	orderNumbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeOrder, proSlots)
	if err != nil {
		return err
	}

	for begin := 0; begin < len(plans); begin += shpChunkSize {
		end := min(begin+shpChunkSize, len(plans))
		batch := &shpBatch{stamps: make(map[string][]stamp)}
		for i := begin; i < end; i++ {
			if err = b.build(batch, plans[i], pros[i], orderNumbers[i]); err != nil {
				return fmt.Errorf("build shipment %s: %w", pros[i], err)
			}
		}
		if err = g.inTx(ctx, func(ctx context.Context, tx bun.Tx) error {
			return b.write(ctx, tx, batch)
		}); err != nil {
			return err
		}
		g.progress("shipments", end, len(plans))
	}

	for _, entity := range []struct{ table, label string }{
		{shpTableOrders, "Orders"},
		{shpTableShipments, "Shipments"},
		{shpTableMoves, "Shipment moves"},
		{shpTableStops, "Stops"},
		{shpTableAssignments, "Driver assignments"},
		{shpTableCarrierAssignments, "Carrier assignments"},
		{shpTableCommodities, "Shipment commodities"},
		{shpTableCharges, "Additional charges"},
		{shpTableComments, "Shipment comments"},
		{"shipment_comment_mentions", "Comment mentions"},
		{"shipment_comment_acknowledgments", "Comment acknowledgments"},
		{shpTableHolds, "Shipment holds"},
		{"check_calls", "Check calls"},
	} {
		g.report.add(entity.label, b.counts[entity.table])
	}
	for _, status := range []shipment.Status{
		shipment.StatusNew, shipment.StatusPartiallyAssigned, shipment.StatusAssigned,
		shipment.StatusInTransit, shipment.StatusDelayed, shipment.StatusPartiallyCompleted,
		shipment.StatusCompleted, shipment.StatusReadyToInvoice, shipment.StatusInvoiced,
		shipment.StatusCanceled,
	} {
		g.opts.Logf("  • shipments %s: %d", status, b.statuses[status])
	}

	return nil
}

func (g *generator) shpLoadLookups(ctx context.Context) (*shpLookups, error) {
	look := &shpLookups{
		trailerTypes:  make(map[trailerKind]pulid.ID),
		serviceTypes:  make(map[string]pulid.ID),
		shipmentTypes: make(map[string]pulid.ID),
		templates:     make(map[string]*formulatemplate.FormulaTemplate),
	}

	var equipment []*equipmenttype.EquipmentType
	if err := g.db.NewSelect().Model(&equipment).
		Where("et.organization_id = ?", g.refs.orgID).
		Where("et.status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load equipment types: %w", err)
	}
	for _, et := range equipment {
		switch et.Class {
		case equipmenttype.ClassTractor:
			if look.tractorTypeID.IsNil() {
				look.tractorTypeID = et.ID
			}
		case equipmenttype.ClassTrailer:
			kind := trailerKindOf(et)
			if _, ok := look.trailerTypes[kind]; !ok {
				look.trailerTypes[kind] = et.ID
			}
			if look.anyTrailer.IsNil() {
				look.anyTrailer = et.ID
			}
		}
	}

	var services []*servicetype.ServiceType
	if err := g.db.NewSelect().Model(&services).
		Where("st.organization_id = ?", g.refs.orgID).
		Where("st.status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load service types: %w", err)
	}
	for _, s := range services {
		look.serviceTypes[strings.ToUpper(s.Code)] = s.ID
	}

	var types []*shipmenttype.ShipmentType
	if err := g.db.NewSelect().Model(&types).
		Where("sht.organization_id = ?", g.refs.orgID).
		Where("sht.status = ?", domaintypes.StatusActive).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load shipment types: %w", err)
	}
	for _, t := range types {
		look.shipmentTypes[strings.ToUpper(t.Code)] = t.ID
	}

	var users []*tenant.User
	if err := g.db.NewSelect().Model(&users).
		Where("usr.id IN (?)", bun.List(g.refs.userIDs)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	for _, u := range users {
		name := u.Name
		if name == "" {
			name = u.Username
		}
		look.users = append(look.users, shpUser{ID: u.ID, Name: name})
	}
	if len(look.users) == 0 {
		look.users = append(look.users, shpUser{ID: g.refs.adminID, Name: "Admin"})
	}

	if err := g.db.NewSelect().Model(&look.holdReasons).
		Where("hr.organization_id = ?", g.refs.orgID).
		Where("hr.active = ?", true).
		Where("hr.type = ?", holdreason.HoldTypeOperational).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load hold reasons: %w", err)
	}

	for _, t := range g.refs.templates {
		look.templates[t.Name] = t
	}

	return look, nil
}

func (g *generator) shpNewBuilder(look *shpLookups) *shpBuilder {
	b := &shpBuilder{
		g:     g,
		now:   unix(g.now),
		look:  look,
		coord: shipmentstate.NewCoordinatorWithClock(func() int64 { return unix(g.now) }),
		customerFactor: make(
			map[pulid.ID]float64,
			len(g.customers),
		),
		lanes:        make(map[pulid.ID][]shpLane, len(g.customers)),
		crewFree:     make([]int64, len(g.crews)),
		crewsByKind:  make(map[trailerKind][]int),
		carrierPools: make(map[pulid.ID][]pulid.ID, len(g.customers)),
		allCrews:     make([]int, len(g.crews)),
		zones:        make(map[string]*time.Location),
		counts:       make(map[string]int),
		statuses:     make(map[shipment.Status]int),
		externalRefs: make(map[string]struct{}),
	}
	for i, c := range g.crews {
		b.allCrews[i] = i
		b.crewsByKind[c.TrailerKind] = append(b.crewsByKind[c.TrailerKind], i)
	}
	b.customerWeights = make([]int, len(g.customers))
	for i, c := range g.customers {
		b.customerWeights[i] = max(c.Weight, 1)
		b.customerFactor[c.ID] = 0.88 + g.fake.float()*0.27
	}
	for _, loc := range g.locations {
		if loc.Shipper {
			b.shippers = append(b.shippers, loc)
		}
		if loc.Consignee {
			b.consignees = append(b.consignees, loc)
		}
	}
	if len(b.shippers) == 0 {
		b.shippers = g.locations
	}
	if len(b.consignees) == 0 {
		b.consignees = g.locations
	}

	return b
}

func (b *shpBuilder) zone(name string) *time.Location {
	if loc, ok := b.zones[name]; ok {
		return loc
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		loc = time.UTC
	}
	b.zones[name] = loc
	return loc
}

func (b *shpBuilder) planAll(n int) []*shpPlan {
	f := b.g.fake
	begin := unix(b.g.start)
	end := b.now + shpFutureDays*shpDay
	span := float64(end - begin)
	const low, slope = 0.7, 0.6

	plans := make([]*shpPlan, 0, n)
	for range n {
		u := f.float()
		x := (-low + math.Sqrt(low*low+2*slope*u*(low+slope/2))) / slope
		day := begin + int64(x*span)

		p := b.planRoute()
		zone := b.zone(p.moves[0][0].loc.City.Timezone)
		local := time.Unix(day, 0).In(zone)
		switch local.Weekday() {
		case time.Saturday:
			if f.chance(0.75) {
				local = local.AddDate(0, 0, -1)
			}
		case time.Sunday:
			if f.chance(0.75) {
				local = local.AddDate(0, 0, 1)
			}
		}
		pickup := time.Date(
			local.Year(), local.Month(), local.Day(),
			f.between(5, 15), pick(f, []int{0, 0, 0, 30}), 0, 0, zone,
		)
		p.pickupAt = pickup.Unix()
		p.createdAt = b.leadTime(p.pickupAt)
		p.canceled = f.chance(0.02)
		plans = append(plans, p)
	}

	sort.SliceStable(plans, func(i, j int) bool { return plans[i].pickupAt < plans[j].pickupAt })
	return plans
}

func (b *shpBuilder) leadTime(pickupAt int64) int64 {
	f := b.g.fake
	var lead int64
	switch f.weighted([]int{30, 45, 20, 5}) {
	case 0:
		lead = f.between64(2*shpHour, shpDay)
	case 1:
		lead = f.between64(shpDay, 4*shpDay)
	case 2:
		lead = f.between64(4*shpDay, 10*shpDay)
	default:
		lead = f.between64(10*shpDay, 21*shpDay)
	}
	created := pickupAt - lead
	if latest := b.now - f.between64(5*shpMinute, 2*shpDay); created > latest {
		created = latest
	}
	return created
}

func (b *shpBuilder) pickCustomer() *customerRecord {
	return b.g.customers[b.g.fake.weighted(b.customerWeights)]
}

func (b *shpBuilder) nearLocation(
	pool []*locationRecord,
	anchor *cityRecord,
	radius float64,
	exclude *locationRecord,
) *locationRecord {
	f := b.g.fake
	if anchor != nil {
		for range 25 {
			candidate := pick(f, pool)
			if candidate != exclude && haversineMiles(anchor, candidate.City) <= radius {
				return candidate
			}
		}
	}
	for range 10 {
		candidate := pick(f, pool)
		if candidate != exclude {
			return candidate
		}
	}
	return pick(f, b.g.locations)
}

func (b *shpBuilder) farLocation(pool []*locationRecord, from *locationRecord) *locationRecord {
	f := b.g.fake
	for range 25 {
		candidate := pick(f, pool)
		if candidate != from && haversineMiles(from.City, candidate.City) >= 40 {
			return candidate
		}
	}
	return b.nearLocation(pool, nil, 0, from)
}

func (b *shpBuilder) lane(c *customerRecord) shpLane {
	f := b.g.fake
	lanes, ok := b.lanes[c.ID]
	if !ok {
		count := f.between(3, 10)
		lanes = make([]shpLane, 0, count)
		for range count {
			if lane, ok := b.ownLane(c); ok {
				lanes = append(lanes, lane)
				continue
			}
			var origin *locationRecord
			if f.chance(0.7) {
				origin = b.nearLocation(b.shippers, c.Home, 450, nil)
			} else {
				origin = pick(f, b.shippers)
			}
			lanes = append(lanes, shpLane{origin: origin, dest: b.farLocation(b.consignees, origin)})
		}
		b.lanes[c.ID] = lanes
	}
	if f.chance(0.75) {
		return lanes[f.intn(len(lanes))]
	}
	origin := pick(f, b.shippers)
	return shpLane{origin: origin, dest: b.farLocation(b.consignees, origin)}
}

func (b *shpBuilder) ownLane(c *customerRecord) (shpLane, bool) {
	f := b.g.fake
	if len(c.Facilities) == 0 || !f.chance(0.8) {
		return shpLane{}, false
	}
	own := pick(f, c.Facilities)
	if own.Consignee && (!own.Shipper || f.chance(0.25)) {
		return shpLane{origin: b.farLocation(b.shippers, own), dest: own}, true
	}
	if own.Shipper {
		return shpLane{origin: own, dest: b.farLocation(b.consignees, own)}, true
	}
	return shpLane{}, false
}

func (b *shpBuilder) planRoute() *shpPlan {
	f := b.g.fake
	p := &shpPlan{customer: b.pickCustomer()}
	l := b.lane(p.customer)

	switch f.weighted([]int{82, 10, 8}) {
	case 0:
		p.moves = [][]shpStopPlan{{
			{loc: l.origin, typ: shipment.StopTypePickup},
			{loc: l.dest, typ: shipment.StopTypeDelivery},
		}}
	case 1:
		stops := []shpStopPlan{{loc: l.origin, typ: shipment.StopTypePickup}}
		if f.chance(0.3) {
			stops = append(stops, shpStopPlan{
				loc: b.nearLocation(b.shippers, l.origin.City, 120, l.origin),
				typ: shipment.StopTypePickup,
			})
		}
		prev := l.dest
		stops = append(stops, shpStopPlan{loc: prev, typ: shipment.StopTypeDelivery})
		for range f.between(1, 2) {
			next := b.nearLocation(b.consignees, prev.City, 220, prev)
			stops = append(stops, shpStopPlan{loc: next, typ: shipment.StopTypeDelivery})
			prev = next
		}
		p.moves = [][]shpStopPlan{stops}
	default:
		p.relay = true
		legs := f.between(2, 3)
		points := []*locationRecord{l.origin}
		for i := 1; i < legs; i++ {
			points = append(points, b.farLocation(b.g.locations, points[i-1]))
		}
		points = append(points, l.dest)
		for i := range legs {
			p.moves = append(p.moves, []shpStopPlan{
				{loc: points[i], typ: shipment.StopTypePickup},
				{loc: points[i+1], typ: shipment.StopTypeDelivery},
			})
		}
	}

	b.planFreight(p)
	return p
}

func (b *shpBuilder) planFreight(p *shpPlan) {
	f := b.g.fake
	count := []int{1, 1, 1, 1, 1, 1, 2, 2, 2, 3}[f.intn(10)]
	p.flatbed = f.chance(0.08)
	seen := make(map[pulid.ID]bool, count)
	for range count {
		c := pick(f, b.g.commodities)
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true
		pieces := int64(f.between(1, 26))
		perPiece := c.WeightPerPiece
		if perPiece <= 0 {
			perPiece = int64(f.between(300, 1800))
		}
		weight := int64(float64(pieces*perPiece) * (0.8 + f.float()*0.4))
		weight = max(weight, 50)
		p.commodities = append(p.commodities, shpCommodityPlan{
			commodity: c,
			pieces:    pieces,
			weight:    weight,
		})
		p.pieces += pieces
		p.weight += weight
		if c.TempMin != nil || c.TempMax != nil {
			p.reefer = true
		}
		if c.Hazmat {
			p.hazmat = true
		}
	}
	if p.weight > shpMaxWeight {
		scale := float64(shpMaxWeight) / float64(p.weight)
		p.weight = 0
		for i := range p.commodities {
			p.commodities[i].weight = max(int64(float64(p.commodities[i].weight)*scale), 50)
			p.weight += p.commodities[i].weight
		}
	}
	if p.reefer {
		p.flatbed = false
	}

	switch {
	case p.hazmat && f.chance(0.6):
		p.serviceCode = "HAZM"
	case p.reefer && f.chance(0.7):
		p.serviceCode = "TEMP"
	default:
		p.serviceCode = []string{"STD", "STD", "STD", "STD", "STD", "STD", "STD", "EXP", "TEAM", "DTP"}[f.intn(10)]
	}
	p.team = p.serviceCode == "TEAM"
	p.detention = f.chance(0.09)
}

func (b *shpBuilder) shipmentTypeCode(p *shpPlan) string {
	f := b.g.fake
	switch {
	case p.hazmat:
		return "HAZM"
	case p.reefer:
		return "REEF"
	case p.flatbed:
		return "FLAT"
	case p.weight < 12_000 && f.chance(0.6):
		return "LTL"
	case p.weight < 20_000 && f.chance(0.3):
		return "PART"
	case p.serviceCode == "EXP":
		return "EXP"
	default:
		return "FTL"
	}
}

func (b *shpBuilder) trailerKind(p *shpPlan) trailerKind {
	switch {
	case p.reefer:
		return trailerKindReefer
	case p.flatbed:
		return trailerKindFlatbed
	default:
		return trailerKindDry
	}
}

func (b *shpBuilder) trailerType(kind trailerKind) pulid.ID {
	if id, ok := b.look.trailerTypes[kind]; ok {
		return id
	}
	return b.look.anyTrailer
}

func pickID(f *faker, preferred map[string]pulid.ID, code string, fallback []pulid.ID) pulid.ID {
	if id, ok := preferred[code]; ok {
		return id
	}
	if len(fallback) == 0 {
		return pulid.Nil
	}
	return pick(f, fallback)
}

func driveSeconds(miles float64, team bool) int64 {
	hours := miles / 50
	if !team {
		hours += math.Floor(hours/11) * 10
	} else {
		hours += math.Floor(hours/11) * 0.5
	}
	return int64(hours * float64(shpHour))
}

type shpTimedStop struct {
	plan        shpStopPlan
	windowStart int64
	windowEnd   int64
	appointment bool
	arrival     int64
	departure   int64
	miles       float64
}

func (b *shpBuilder) schedule(p *shpPlan) [][]*shpTimedStop {
	f := b.g.fake
	moves := make([][]*shpTimedStop, len(p.moves))
	plannedDeparture := int64(0)
	actualDeparture := int64(0)
	detentionStop := -1
	if p.detention {
		detentionStop = f.intn(len(p.moves[len(p.moves)-1])-1) + 1
	}

	for m, stops := range p.moves {
		timed := make([]*shpTimedStop, len(stops))
		for s, sp := range stops {
			ts := &shpTimedStop{plan: sp, appointment: f.chance(0.45)}
			zone := b.zone(sp.loc.City.Timezone)
			var plannedArrival int64
			switch {
			case m == 0 && s == 0:
				ts.windowStart = p.pickupAt
				plannedArrival = ts.windowStart
			case s == 0:
				ts.windowStart = b.businessHours(
					plannedDeparture+f.between64(2*shpHour, 18*shpHour),
					zone,
					ts.appointment,
				)
				plannedArrival = ts.windowStart
			default:
				prev := timed[s-1]
				ts.miles = roadMiles(prev.plan.loc.City, sp.loc.City)
				plannedArrival = plannedDeparture + driveSeconds(ts.miles, p.team)
				ts.windowStart = b.businessHours(plannedArrival, zone, ts.appointment)
			}
			if ts.appointment {
				ts.windowEnd = ts.windowStart + shpHour
			} else {
				ts.windowEnd = ts.windowStart + f.between64(4, 8)*shpHour
			}

			switch {
			case s == 0 && m == 0:
				ts.arrival = ts.windowStart + b.lateness()
			case s == 0:
				ts.arrival = max(actualDeparture+f.between64(shpHour, 6*shpHour), ts.windowStart+b.lateness())
			default:
				drive := int64(float64(driveSeconds(ts.miles, p.team)) * (0.92 + f.float()*0.25))
				ts.arrival = max(actualDeparture+drive, ts.windowStart-f.between64(0, 45*shpMinute))
				if f.chance(0.08) {
					ts.arrival += f.between64(shpHour, 6*shpHour)
				}
			}
			dwell := f.between64(45*shpMinute, 3*shpHour)
			if m == len(p.moves)-1 && s == detentionStop {
				dwell = f.between64(3*shpHour+15*shpMinute, 6*shpHour)
			}
			ts.departure = ts.arrival + dwell
			plannedDeparture = max(ts.windowStart, plannedArrival) + shpHour
			actualDeparture = ts.departure
			timed[s] = ts
		}
		moves[m] = timed
	}

	return moves
}

func (b *shpBuilder) lateness() int64 {
	f := b.g.fake
	if f.chance(0.1) {
		return f.between64(shpHour, 6*shpHour)
	}
	return f.between64(-20*shpMinute, 40*shpMinute)
}

func (b *shpBuilder) businessHours(at int64, zone *time.Location, appointment bool) int64 {
	t := time.Unix(at, 0).In(zone)
	t = t.Truncate(30 * time.Minute).Add(30 * time.Minute)
	if appointment {
		switch {
		case t.Hour() < 5:
			t = time.Date(t.Year(), t.Month(), t.Day(), 7, 0, 0, 0, zone)
		case t.Hour() >= 18:
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 7, 0, 0, 0, zone)
		}
	}
	return t.Unix()
}

func (b *shpBuilder) pickCrew(start, end int64, p *shpPlan, kind trailerKind) *crew {
	f := b.g.fake
	pool := b.crewsByKind[kind]
	if len(pool) == 0 {
		pool = b.allCrews
	}
	fits := func(c *crew, strict bool) bool {
		if p.hazmat && !c.Hazmat {
			return false
		}
		return !strict || !p.team || c.SecondaryWorkerID != nil
	}

	best := -1
	for try := range 48 {
		i := pool[f.intn(len(pool))]
		if !fits(b.g.crews[i], try < 32) {
			continue
		}
		if b.crewFree[i] <= start {
			best = i
			break
		}
		if best < 0 || b.crewFree[i] < b.crewFree[best] {
			best = i
		}
	}
	if best < 0 {
		for _, i := range pool {
			if fits(b.g.crews[i], false) && (best < 0 || b.crewFree[i] < b.crewFree[best]) {
				best = i
			}
		}
	}
	if best < 0 {
		best = pool[f.intn(len(pool))]
	}
	b.crewFree[best] = max(b.crewFree[best], end)
	return b.g.crews[best]
}

func (b *shpBuilder) carrierFor(c *customerRecord) pulid.ID {
	f := b.g.fake
	pool, ok := b.carrierPools[c.ID]
	if !ok {
		for range f.between(2, 4) {
			pool = append(pool, pick(f, b.g.carriers))
		}
		b.carrierPools[c.ID] = pool
	}
	if f.chance(0.8) {
		return pick(f, pool)
	}
	return pick(f, b.g.carriers)
}

func (b *shpBuilder) carrierAssignment(
	move *shipment.ShipmentMove,
	p *shpPlan,
	assignedAt int64,
	started bool,
) *shipment.CarrierAssignment {
	f := b.g.fake
	miles := 0.0
	if move.Distance != nil {
		miles = *move.Distance
	}
	assigner := b.user()
	ca := &shipment.CarrierAssignment{
		ID:                    idAt(shipment.CarrierAssignmentIDPrefix, assignedAt),
		BusinessUnitID:        b.g.refs.buID,
		OrganizationID:        b.g.refs.orgID,
		ShipmentMoveID:        move.ID,
		CarrierID:             b.carrierFor(p.customer),
		Status:                shipment.CarrierAssignmentStatusConfirmed,
		RateMethod:            shipment.CarrierRateMethodFlat,
		CurrencyCode:          "USD",
		ProNumber:             f.letters(2) + f.digits(7),
		ExternalDriverName:    f.firstName() + " " + f.lastName(),
		ExternalDriverPhone:   f.phone(),
		ExternalTractorNumber: f.digits(4),
		ExternalTrailerNumber: "53" + f.digits(4),
		AssignedByID:          ptr(assigner.ID),
		Version:               1,
		CreatedAt:             assignedAt,
	}
	rpm := 1.7 + 120/(miles+40) + f.float()*0.35
	if f.chance(0.4) && miles >= 150 {
		ca.RateMethod = shipment.CarrierRateMethodPerMile
		ca.BaseRate = decimal.NewFromFloat(rpm).Round(2)
	} else {
		ca.BaseRate = decimal.NewFromFloat(math.Round(math.Max(miles*rpm, 300)/5) * 5)
	}
	ca.FuelSurcharge = decimal.NewFromFloat(miles * (0.38 + f.float()*0.2)).Round(2)
	ca.SyncTotals(move.Distance)
	if !started && f.chance(0.25) {
		ca.Status = shipment.CarrierAssignmentStatusPending
	} else {
		ca.ConfirmedAt = ptr(min(assignedAt+b.at(10*shpMinute, 6*shpHour), b.now))
	}
	return ca
}

func (b *shpBuilder) user() shpUser {
	return pick(b.g.fake, b.look.users)
}

func (b *shpBuilder) otherUser(not pulid.ID) (shpUser, bool) {
	for range 6 {
		u := b.user()
		if u.ID != not {
			return u, true
		}
	}
	return shpUser{}, false
}

func (b *shpBuilder) at(low, high int64) int64 {
	if high <= low {
		return low
	}
	return b.g.fake.between64(low, high)
}

//nolint:gocognit,funlen // one shipment's graph is built in one place so its parts agree
func (b *shpBuilder) build(batch *shpBatch, p *shpPlan, pro, orderNumber string) error {
	f := b.g.fake
	refs := b.g.refs
	orgID, buID := refs.orgID, refs.buID
	timeline := b.schedule(p)

	shipmentID := idAt("shp_", p.createdAt)
	orderID := idAt("ord_", p.createdAt)
	entered := b.user()
	owner := b.user()
	po := ""
	if f.chance(0.8) {
		po = b.poNumber(p.customer)
	}
	bol := f.digits(10)

	entity := &shipment.Shipment{
		ID:             shipmentID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		ServiceTypeID:  pickID(f, b.look.serviceTypes, p.serviceCode, refs.serviceTypes),
		ShipmentTypeID: pickID(f, b.look.shipmentTypes, b.shipmentTypeCode(p), refs.shipmentTypes),
		CustomerID:     p.customer.ID,
		TractorTypeID:  b.look.tractorTypeID,
		TrailerTypeID:  b.trailerType(b.trailerKind(p)),
		OwnerID:        owner.ID,
		EnteredByID:    entered.ID,
		OrderID:        orderID,
		EntryMethod:    shipment.EntryMethodManual,
		ProNumber:      pro,
		BOL:            bol,
		Pieces:         ptr(p.pieces),
		Weight:         ptr(p.weight),
		RatingUnit:     1,
		FreightTerms:   shipment.FreightTermsPrepaid,
		Version:        1,
	}
	if f.chance(0.08) {
		entity.FreightTerms = shipment.FreightTermsCollect
	}
	if f.chance(0.15) {
		entity.EntryMethod = shipment.EntryMethodEDI
		entity.ExternalReference = b.externalRef(p, func() string { return "LT" + f.digits(9) })
	} else if f.chance(0.2) {
		entity.ExternalReference = b.externalRef(p, func() string {
			return p.customer.Code + "-" + f.digits(6)
		})
	}
	for _, c := range p.commodities {
		if c.commodity.TempMin != nil {
			entity.TemperatureMin = c.commodity.TempMin
			entity.TemperatureMax = c.commodity.TempMax
			break
		}
	}

	var lastActual int64
	totalMiles := 0.0
	totalStops := 0
	for m, timed := range timeline {
		moveID := idAt("sm_", p.createdAt)
		move := &shipment.ShipmentMove{
			ID:             moveID,
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shipmentID,
			Status:         shipment.MoveStatusNew,
			CoverageType:   shipment.MoveCoverageTypeUnassigned,
			Loaded:         true,
			Sequence:       int64(m),
		}
		miles := 0.0
		perType := make(map[shipment.StopType]int64, 2)
		for _, ts := range timed {
			perType[ts.plan.typ]++
		}
		for s, ts := range timed {
			miles += ts.miles
			stopID := idAt("stp_", p.createdAt)
			pieces, weight := p.pieces, p.weight
			if n := perType[ts.plan.typ]; n > 1 {
				pieces = max(p.pieces/n, 1)
				weight = max(p.weight/n, 1)
			}
			stop := &shipment.Stop{
				ID:                   stopID,
				BusinessUnitID:       buID,
				OrganizationID:       orgID,
				ShipmentMoveID:       moveID,
				LocationID:           ts.plan.loc.ID,
				Type:                 ts.plan.typ,
				ScheduleType:         shipment.StopScheduleTypeOpen,
				Sequence:             int64(s),
				Pieces:               ptr(pieces),
				Weight:               ptr(weight),
				ScheduledWindowStart: ts.windowStart,
				ScheduledWindowEnd:   ptr(ts.windowEnd),
				AddressLine:          shpAddressLine(ts.plan.loc),
				Version:              1,
			}
			if ts.appointment {
				stop.ScheduleType = shipment.StopScheduleTypeAppointment
			}
			if !p.canceled && ts.arrival <= b.now {
				stop.ActualArrival = ptr(ts.arrival)
				lastActual = ts.arrival
				if ts.departure <= b.now {
					stop.ActualDeparture = ptr(ts.departure)
					lastActual = ts.departure
				}
			}
			move.Stops = append(move.Stops, stop)
			totalStops++
		}
		move.Distance = ptr(math.Round(miles*10) / 10)
		move.DistanceSource = distancecalculation.SourcePCMiler
		move.DistanceProvider = string(integration.TypePCMiler)
		move.DistanceRoutingType = distanceprofile.DefaultRoutingType
		move.DistanceUnits = distanceprofile.DefaultDistanceUnits
		move.DistanceDataVersion = distanceprofile.DefaultDataVersion
		move.DistanceCalculatedAt = ptr(p.createdAt)
		totalMiles += miles
		entity.Moves = append(entity.Moves, move)
	}
	totalMiles = math.Round(totalMiles*10) / 10

	started := false
	moveCrews := make(map[pulid.ID]*crew, len(entity.Moves))
	for m, move := range entity.Moves {
		timed := timeline[m]
		moveStarted := move.Stops[0].ActualArrival != nil
		if moveStarted {
			started = true
		}
		if p.canceled {
			continue
		}
		start := timed[0].windowStart
		end := timed[len(timed)-1].departure
		lead := start - b.now
		var assign bool
		switch {
		case moveStarted, started, start < b.now:
			assign = moveStarted || started || f.chance(0.95)
			if started && !moveStarted {
				assign = f.chance(0.85)
			}
		case lead <= 2*shpDay:
			assign = f.chance(0.8)
		case lead <= 7*shpDay:
			assign = f.chance(0.45)
		default:
			assign = f.chance(0.15)
		}
		if !assign {
			continue
		}
		assignedAt := min(b.at(p.createdAt+10*shpMinute, start-shpHour), b.now-shpMinute)
		assignedAt = max(assignedAt, p.createdAt)
		if len(b.g.carriers) > 0 && f.chance(0.12) {
			move.CarrierAssignment = b.carrierAssignment(move, p, assignedAt, moveStarted)
			move.CoverageType = shipment.MoveCoverageTypeCarrier
			continue
		}
		c := b.pickCrew(start-2*shpHour, end, p, b.trailerKind(p))
		moveCrews[move.ID] = c
		if move.Sequence == 0 || entity.Moves[0].Assignment == nil {
			entity.TractorTypeID = pulid.FirstNotNil(c.TractorTypeID, entity.TractorTypeID)
			entity.TrailerTypeID = pulid.FirstNotNil(c.TrailerTypeID, entity.TrailerTypeID)
		}
		a := &shipment.Assignment{
			ID:                idAt("asn_", assignedAt),
			BusinessUnitID:    buID,
			OrganizationID:    orgID,
			ShipmentMoveID:    move.ID,
			PrimaryWorkerID:   ptr(c.WorkerID),
			TractorID:         ptr(c.TractorID),
			TrailerID:         pulid.PtrOrNil(c.TrailerID),
			SecondaryWorkerID: c.SecondaryWorkerID,
			Status:            shipment.AssignmentStatusNew,
			AckStatus:         shipment.AssignmentAckPending,
			Version:           1,
			CreatedAt:         assignedAt,
		}
		if moveStarted || f.chance(0.75) {
			ackAt := min(assignedAt+b.at(5*shpMinute, 90*shpMinute), b.now)
			a.AckStatus = shipment.AssignmentAckAccepted
			a.AckAt = ptr(ackAt)
		}
		move.Assignment = a
		move.CoverageType = shipment.MoveCoverageTypeDriver
	}

	if p.canceled {
		entity.Status = shipment.StatusCanceled
		for _, move := range entity.Moves {
			move.Status = shipment.MoveStatusCanceled
			for _, stop := range move.Stops {
				stop.Status = shipment.StopStatusCanceled
			}
		}
		canceledAt := b.at(p.createdAt+shpHour, min(p.pickupAt, b.now-shpMinute))
		canceledBy := b.user()
		entity.CanceledAt = ptr(canceledAt)
		entity.CanceledByID = canceledBy.ID
		entity.CancelReason = pick(f, shpCancelReasons)
		lastActual = canceledAt
	} else if multiErr := b.coord.PrepareForCreateWithDelayThreshold(
		entity,
		shipmentstate.DefaultDelayThresholdMinutes,
	); multiErr != nil {
		return multiErr
	}

	for _, move := range entity.Moves {
		if move.Assignment == nil {
			continue
		}
		switch move.Status {
		case shipment.MoveStatusCompleted:
			move.Assignment.Status = shipment.AssignmentStatusCompleted
		case shipment.MoveStatusInTransit:
			move.Assignment.Status = shipment.AssignmentStatusInProgress
		default:
			move.Assignment.Status = shipment.AssignmentStatusNew
		}
	}

	rec := &shipmentRecord{
		ID:          shipmentID,
		OrderID:     orderID,
		ProNumber:   pro,
		BOL:         bol,
		OrderNumber: orderNumber,
		PONumber:    po,
		Customer:    p.customer,
		CreatedAt:   p.createdAt,
		PickupAt:    p.pickupAt,
		Miles:       totalMiles,
	}
	if entity.ActualDeliveryDate != nil {
		rec.DeliveredAt = *entity.ActualDeliveryDate
	}
	rec.Moves = shpMoveRecords(entity, timeline, moveCrews)

	b.planBilling(entity, rec)
	b.price(entity, rec, p, totalMiles, totalStops, batch)
	rec.Status = entity.Status
	b.statuses[entity.Status]++

	updated := max(p.createdAt, lastActual)
	switch rec.Stage {
	case billingStageInvoiced:
		updated = max(updated, rec.InvoiceDate)
	case billingStageQueued:
		updated = max(updated, rec.TransferredAt)
	case billingStageNone:
	}
	updated = min(updated, b.now)

	ord := &order.Order{
		ID:             orderID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
		CustomerID:     p.customer.ID,
		OwnerID:        owner.ID,
		EnteredByID:    entered.ID,
		Status:         shpOrderStatus(entity.Status),
		OrderNumber:    orderNumber,
		PONumber:       po,
		BOL:            bol,
		CurrencyCode:   "USD",
		BaseAmount:     entity.FreightChargeAmount,
		TotalAmount:    entity.TotalChargeAmount,
		Version:        1,
	}
	batch.orders = append(batch.orders, ord)
	batch.stamp(shpTableOrders, orderID, p.createdAt, updated)

	batch.shipments = append(batch.shipments, entity)
	batch.stamp(shpTableShipments, shipmentID, p.createdAt, updated)
	for _, move := range entity.Moves {
		batch.moves = append(batch.moves, move)
		batch.stamp(shpTableMoves, move.ID, p.createdAt, updated)
		for _, stop := range move.Stops {
			stopUpdated := p.createdAt
			if stop.ActualDeparture != nil {
				stopUpdated = *stop.ActualDeparture
			} else if stop.ActualArrival != nil {
				stopUpdated = *stop.ActualArrival
			}
			if entity.CanceledAt != nil {
				stopUpdated = *entity.CanceledAt
			}
			batch.stops = append(batch.stops, stop)
			batch.stamp(shpTableStops, stop.ID, p.createdAt, stopUpdated)
		}
		if a := move.Assignment; a != nil {
			assignUpdated := a.CreatedAt
			if a.AckAt != nil {
				assignUpdated = *a.AckAt
			}
			if move.Status == shipment.MoveStatusCompleted || move.Status == shipment.MoveStatusInTransit {
				assignUpdated = max(assignUpdated, lastActual)
			}
			batch.assignments = append(batch.assignments, a)
			batch.stamp(shpTableAssignments, a.ID, a.CreatedAt, min(assignUpdated, b.now))
		}
		if ca := move.CarrierAssignment; ca != nil {
			caUpdated := ca.CreatedAt
			if ca.ConfirmedAt != nil {
				caUpdated = *ca.ConfirmedAt
			}
			batch.carrierAssignments = append(batch.carrierAssignments, ca)
			batch.stamp(shpTableCarrierAssignments, ca.ID, ca.CreatedAt, caUpdated)
		}
	}
	for _, c := range p.commodities {
		sc := &shipment.ShipmentCommodity{
			ID:             idAt("sc_", p.createdAt),
			BusinessUnitID: buID,
			OrganizationID: orgID,
			ShipmentID:     shipmentID,
			CommodityID:    c.commodity.ID,
			Weight:         c.weight,
			Pieces:         c.pieces,
			Version:        1,
		}
		batch.commodities = append(batch.commodities, sc)
		batch.stamp(shpTableCommodities, sc.ID, p.createdAt, p.createdAt)
	}

	b.comments(batch, entity, rec, p, timeline)
	b.checkCalls(batch, entity, timeline)
	b.holds(batch, entity, rec)

	b.g.shipments = append(b.g.shipments, rec)
	return nil
}

func shpMoveRecords(
	entity *shipment.Shipment,
	timeline [][]*shpTimedStop,
	crews map[pulid.ID]*crew,
) []*moveRecord {
	records := make([]*moveRecord, 0, len(entity.Moves))
	for m, move := range entity.Moves {
		rec := &moveRecord{
			ID:                move.ID,
			Sequence:          move.Sequence,
			Status:            move.Status,
			Crew:              crews[move.ID],
			CarrierAssignment: move.CarrierAssignment,
		}
		if move.Distance != nil {
			rec.Miles = *move.Distance
		}
		if move.Assignment != nil {
			rec.AssignmentID = move.Assignment.ID
		}
		for _, ts := range timeline[m] {
			rec.Cities = append(rec.Cities, ts.plan.loc.City)
		}
		first, last := move.Stops[0], move.Stops[len(move.Stops)-1]
		if first.ActualArrival != nil {
			rec.StartedAt = *first.ActualArrival
		}
		if move.Status == shipment.MoveStatusCompleted && last.ActualDeparture != nil {
			rec.CompletedAt = *last.ActualDeparture
		}
		records = append(records, rec)
	}
	return records
}

// externalRef draws a reference unique per customer, as
// uq_shipments_external_reference requires of every shipment not canceled.
func (b *shpBuilder) externalRef(p *shpPlan, draw func() string) string {
	for {
		ref := draw()
		key := p.customer.ID.String() + "|" + strings.ToLower(ref)
		if _, taken := b.externalRefs[key]; !taken {
			b.externalRefs[key] = struct{}{}
			return ref
		}
	}
}

func (b *shpBuilder) poNumber(c *customerRecord) string {
	f := b.g.fake
	switch f.intn(4) {
	case 0:
		return "PO" + f.digits(8)
	case 1:
		return c.Code + "-" + f.digits(6)
	case 2:
		return f.digits(7)
	default:
		return fmt.Sprintf("%s%s-%s", f.letters(2), f.digits(5), f.digits(2))
	}
}

func shpAddressLine(loc *locationRecord) string {
	return fmt.Sprintf("%s, %s, %s", loc.Address, loc.City.Name, loc.City.State)
}

func shpOrderStatus(status shipment.Status) order.Status {
	switch status {
	case shipment.StatusNew, shipment.StatusAssigned, shipment.StatusPartiallyAssigned:
		return order.StatusConfirmed
	case shipment.StatusInTransit, shipment.StatusDelayed, shipment.StatusPartiallyCompleted:
		return order.StatusInProgress
	case shipment.StatusCompleted, shipment.StatusReadyToInvoice:
		return order.StatusCompleted
	case shipment.StatusInvoiced:
		return order.StatusBilled
	case shipment.StatusCanceled:
		return order.StatusCanceled
	default:
		return order.StatusConfirmed
	}
}

func (b *shpBuilder) planBilling(entity *shipment.Shipment, rec *shipmentRecord) {
	if entity.Status != shipment.StatusCompleted || rec.DeliveredAt == 0 {
		return
	}
	f := b.g.fake
	age := b.now - rec.DeliveredAt

	var weights []int
	switch {
	case age < 2*shpDay:
		weights = []int{60, 40, 0}
	case age < 21*shpDay:
		weights = []int{20, 25, 55}
	case age < 45*shpDay:
		weights = []int{4, 8, 88}
	default:
		weights = []int{1, 2, 97}
	}

	choice := f.weighted(weights)
	if choice == 2 {
		lag := f.between64(shpDay, 10*shpDay)
		room := b.now - shpHour - rec.DeliveredAt
		if room < 2*shpHour {
			choice = 1
		} else {
			lag = min(lag, room)
			rec.InvoiceDate = rec.DeliveredAt + lag
			rec.ReadyToBillAt = rec.DeliveredAt + b.at(shpHour, max(lag/3, shpHour))
			rec.TransferredAt = b.at(rec.ReadyToBillAt, rec.InvoiceDate-shpMinute)
			rec.Stage = billingStageInvoiced
			rec.QueueStatus = billingqueue.StatusPosted
			entity.Status = shipment.StatusInvoiced
			entity.BillingTransferStatus = shipment.BillingTransferPosted
			entity.MarkedReadyToBillAt = ptr(rec.ReadyToBillAt)
			entity.TransferredToBillingAt = ptr(rec.TransferredAt)
			entity.BilledAt = ptr(rec.InvoiceDate)
			return
		}
	}
	if choice != 1 {
		return
	}

	rec.ReadyToBillAt = min(rec.DeliveredAt+b.at(2*shpHour, 2*shpDay), b.now-2*shpMinute)
	rec.ReadyToBillAt = max(rec.ReadyToBillAt, rec.DeliveredAt)
	rec.TransferredAt = b.at(rec.ReadyToBillAt, min(rec.ReadyToBillAt+shpDay, b.now-shpMinute))
	rec.Stage = billingStageQueued
	rec.QueueStatus = []billingqueue.Status{
		billingqueue.StatusReadyForReview,
		billingqueue.StatusInReview,
		billingqueue.StatusOnHold,
		billingqueue.StatusException,
		billingqueue.StatusSentBackToOps,
		billingqueue.StatusApproved,
	}[f.weighted([]int{40, 25, 10, 10, 5, 10})]
	entity.Status = shipment.StatusReadyToInvoice
	entity.BillingTransferStatus = shipment.BillingTransferStatus(rec.QueueStatus)
	entity.MarkedReadyToBillAt = ptr(rec.ReadyToBillAt)
	entity.TransferredToBillingAt = ptr(rec.TransferredAt)
}

func (b *shpBuilder) template(names ...string) *formulatemplate.FormulaTemplate {
	for _, name := range names {
		if t, ok := b.look.templates[name]; ok {
			return t
		}
	}
	return nil
}

func (b *shpBuilder) fuelPercent(at int64) decimal.Decimal {
	phase := float64(at-unix(b.g.start)) / float64(365*shpDay) * 2 * math.Pi
	pct := 22 + 6*math.Sin(phase) + (b.g.fake.float()-0.5)*3
	return decimal.NewFromFloat(pct).Round(1)
}

//nolint:funlen // the rate, its method and every accessorial are priced together
func (b *shpBuilder) price(
	entity *shipment.Shipment,
	rec *shipmentRecord,
	p *shpPlan,
	miles float64,
	stops int,
	batch *shpBatch,
) {
	f := b.g.fake
	rpm := 2.15 + 160/(miles+40)
	rpm *= b.customerFactor[p.customer.ID] * (0.92 + f.float()*0.16)
	if p.reefer {
		rpm *= 1.14
	}
	if p.hazmat {
		rpm *= 1.09
	}
	if p.flatbed {
		rpm *= 1.1
	}
	if p.serviceCode == "EXP" {
		rpm *= 1.18
	}
	if p.team {
		rpm *= 1.12
	}
	target := math.Max(miles*rpm, shpMinimumPrice+f.float()*100)

	var tmpl *formulatemplate.FormulaTemplate
	var baseRate, freight decimal.Decimal
	cwt := decimal.NewFromInt((p.weight + 99) / 100)
	roll := f.float()
	switch {
	case p.weight < 12_000 && roll < 0.5 && b.template(formulatemplate.StandardPerCwt) != nil:
		tmpl = b.template(formulatemplate.StandardPerCwt)
		baseRate = decimal.NewFromFloat(target).Div(cwt).Round(2)
		freight = baseRate.Mul(cwt).Round(2)
	case stops > 2 && !p.relay && roll < 0.5 && b.template(formulatemplate.StandardPerStop) != nil:
		tmpl = b.template(formulatemplate.StandardPerStop)
		baseRate = decimal.NewFromFloat(target / float64(stops)).Round(2)
		freight = baseRate.Mul(decimal.NewFromInt(int64(stops))).Round(2)
	case miles*rpm >= shpMinimumPrice && roll < 0.6 && b.template(formulatemplate.StandardPerMile) != nil:
		tmpl = b.template(formulatemplate.StandardPerMile)
		baseRate = decimal.NewFromFloat(rpm).Round(2)
		freight = baseRate.Mul(decimal.NewFromFloat(miles)).Round(2)
	default:
		tmpl = b.template(formulatemplate.StandardFlatRate)
		baseRate = decimal.NewFromFloat(math.Round(target/5) * 5)
		freight = baseRate
	}
	if tmpl == nil {
		tmpl = pick(f, b.g.refs.templates)
		baseRate = decimal.NewFromFloat(math.Round(target/5) * 5)
		freight = baseRate
	}

	delivered := entity.Status == shipment.StatusCompleted ||
		entity.Status == shipment.StatusReadyToInvoice ||
		entity.Status == shipment.StatusInvoiced

	var charges []chargeRecord
	add := func(code string, amount decimal.Decimal, unit int64, at int64) {
		acc := b.g.refs.accessorial(code)
		if acc == nil || amount.LessThanOrEqual(decimal.Zero) {
			return
		}
		id := idAt("ac_", at)
		ac := &shipment.AdditionalCharge{
			ID:                  id,
			BusinessUnitID:      b.g.refs.buID,
			OrganizationID:      b.g.refs.orgID,
			ShipmentID:          entity.ID,
			AccessorialChargeID: acc.ID,
			Method:              acc.Method,
			Amount:              amount,
			Unit:                int16(unit), //nolint:gosec // bounded unit counts
			IsDetention:         strings.EqualFold(acc.Code, "DET"),
			Version:             1,
		}
		cr := chargeRecord{
			AccessorialChargeID: acc.ID,
			AdditionalChargeID:  id,
			Code:                acc.Code,
			Description:         acc.Description,
			Method:              acc.Method,
			Unit:                unit,
			UnitPrice:           amount,
		}
		switch acc.Method {
		case accessorialcharge.MethodPercentage:
			cr.Amount = freight.Mul(amount).Div(decimal.NewFromInt(100)).Round(2)
			cr.Unit = 1
			cr.Description = fmt.Sprintf("%s (%s%%)", acc.Description, amount.String())
		case accessorialcharge.MethodFlat:
			cr.Amount = amount.Mul(decimal.NewFromInt(max(unit, 1))).Round(2)
		default:
			cr.Amount = amount.Mul(decimal.NewFromInt(unit)).Round(2)
		}
		batch.charges = append(batch.charges, ac)
		batch.stamp(shpTableCharges, id, at, at)
		charges = append(charges, cr)
	}
	rate := func(code string, fallback float64) decimal.Decimal {
		if acc := b.g.refs.accessorial(code); acc != nil && acc.Amount.GreaterThan(decimal.Zero) {
			return acc.Amount
		}
		return decimal.NewFromFloat(fallback)
	}

	if !p.canceled {
		if f.chance(0.85) {
			add("FUEL", b.fuelPercent(p.pickupAt), 1, rec.CreatedAt)
		}
		if extra := stops - 2; extra > 0 && !p.relay {
			add("STOP", rate("STOP", 75), int64(extra), rec.CreatedAt)
		}
		if p.hazmat {
			add("HAZ", rate("HAZ", 350), 1, rec.CreatedAt)
		}
		if p.team {
			add("TEAM", rate("TEAM", 0.25), int64(math.Ceil(miles)), rec.CreatedAt)
		}
		if p.flatbed && f.chance(0.5) {
			add("TARP", rate("TARP", 75), 1, rec.CreatedAt)
		}
	} else if f.chance(0.3) {
		add("TONU", rate("TONU", 250), 1, *entity.CanceledAt)
	}
	if delivered {
		at := rec.DeliveredAt
		if p.detention {
			add("DET", rate("DET", 75), f.between64(1, 4), at)
		}
		if f.chance(0.07) {
			add("LUMP", b.g.fake.money(150, 450), 1, at)
		}
		if f.chance(0.01) {
			add("LAY", rate("LAY", 250), 1, at)
		}
		if f.chance(0.015) {
			add("RESDL", rate("RESDL", 125), 1, at)
		}
	}

	other := decimal.Zero
	for _, c := range charges {
		other = other.Add(c.Amount)
	}
	if p.canceled {
		freight = decimal.Zero
		if len(charges) == 0 {
			baseRate = decimal.Zero
		}
	}
	total := freight.Add(other)

	entity.FormulaTemplateID = tmpl.ID
	entity.BaseRate = decimal.NewNullDecimal(baseRate)
	entity.FreightChargeAmount = decimal.NewNullDecimal(freight)
	entity.OtherChargeAmount = decimal.NewNullDecimal(other)
	entity.TotalChargeAmount = decimal.NewNullDecimal(total)

	rec.FormulaTemplateID = tmpl.ID
	rec.FormulaName = tmpl.Name
	rec.Freight = freight
	rec.Charges = charges
	rec.Total = total
}

type shpCommentSpec struct {
	typ        shipment.CommentType
	visibility shipment.CommentVisibility
	priority   shipment.CommentPriority
	text       string
	at         int64
}

var shpCountWeights = []int{28, 22, 16, 11, 8, 5, 4, 3, 2, 1}

var shpCountValues = []int{0, 1, 2, 3, 4, 5, 6, 8, 12, 20}

//nolint:funlen // the comment catalogue reads best in one place
func (b *shpBuilder) comments(
	batch *shpBatch,
	entity *shipment.Shipment,
	rec *shipmentRecord,
	p *shpPlan,
	timeline [][]*shpTimedStop,
) {
	f := b.g.fake
	count := int(math.Round(float64(shpCountValues[f.weighted(shpCountWeights)]) * b.g.scale.commentRate / 2.4))
	if count == 0 {
		return
	}

	first := timeline[0][0]
	lastMove := timeline[len(timeline)-1]
	last := lastMove[len(lastMove)-1]
	origin := first.plan.loc
	dest := last.plan.loc
	lane := fmt.Sprintf("%s, %s → %s, %s", origin.City.Name, origin.City.State, dest.City.Name, dest.City.State)
	commodity := p.commodities[0].commodity.Name
	charged := make(map[string]bool, len(rec.Charges))
	for _, c := range rec.Charges {
		charged[strings.ToUpper(c.Code)] = true
	}
	pickupAppt := b.localTime(first.windowStart, origin)
	deliveryAppt := b.localTime(last.windowStart, dest)

	var specs []shpCommentSpec
	addSpec := func(
		typ shipment.CommentType,
		vis shipment.CommentVisibility,
		priority shipment.CommentPriority,
		low, high int64,
		text string,
	) {
		high = min(high, b.now-shpMinute)
		if high < low {
			return
		}
		specs = append(specs, shpCommentSpec{
			typ: typ, visibility: vis, priority: priority, text: text, at: b.at(low, high),
		})
	}

	preEnd := min(first.windowStart, b.now)
	if entity.CanceledAt != nil {
		preEnd = *entity.CanceledAt
	}
	pre := []func(){
		func() {
			if !first.appointment {
				return
			}
			addSpec(shipment.CommentTypeAppointment, shipment.CommentVisibilityOperations,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				fmt.Sprintf("Pickup appointment confirmed at %s for %s.%s",
					origin.Name, pickupAppt, shpPORef(" Reference PO ", rec.PONumber, ".")))
		},
		func() {
			if !last.appointment {
				return
			}
			addSpec(shipment.CommentTypeAppointment, shipment.CommentVisibilityOperations,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				fmt.Sprintf("%s set the delivery appointment for %s. Driver to call 30 minutes out.",
					dest.Name, deliveryAppt))
		},
		func() {
			addSpec(shipment.CommentTypePickupInstruction, shipment.CommentVisibilityDriver,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				fmt.Sprintf("PPE required at %s: vest, hard hat and steel toes. Check in at the guard shack%s.",
					origin.Name, shpPORef(" with PO ", rec.PONumber, "")))
		},
		func() {
			addSpec(shipment.CommentTypePickupInstruction, shipment.CommentVisibilityDriver,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				fmt.Sprintf("Shipper load, driver count: %d pieces, %d lbs of %s. Note any exceptions on the BOL.",
					p.pieces, p.weight, commodity))
		},
		func() {
			addSpec(shipment.CommentTypeDeliveryInstruction, shipment.CommentVisibilityDriver,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				shpDeliveryInstruction(f, dest, charged))
		},
		func() {
			addSpec(shipment.CommentTypeDispatch, shipment.CommentVisibilityOperations,
				shipment.CommentPriorityNormal, rec.CreatedAt, preEnd,
				fmt.Sprintf("Built from tender for %s: %s, %.0f miles.", p.customer.Name, lane, rec.Miles))
		},
		func() {
			addSpec(shipment.CommentTypeInternal, shipment.CommentVisibilityInternal,
				shipment.CommentPriorityLow, rec.CreatedAt, preEnd,
				fmt.Sprintf("%s wants proactive updates every 4 hours on this one.", p.customer.Name))
		},
		func() {
			if p.hazmat {
				addSpec(shipment.CommentTypeCompliance, shipment.CommentVisibilityDriver,
					shipment.CommentPriorityHigh, rec.CreatedAt, preEnd,
					"Hazmat load: placards on all four sides, H endorsement required, shipping papers within reach.")
			} else if p.reefer {
				addSpec(shipment.CommentTypePickupInstruction, shipment.CommentVisibilityDriver,
					shipment.CommentPriorityHigh, rec.CreatedAt, preEnd,
					"Pre-cool the trailer before arrival and run continuous, not cycle-sentry.")
			}
		},
	}

	var transit []func()
	var post []func()
	if entity.ActualShipDate != nil {
		departed := *entity.ActualShipDate
		transitEnd := b.now
		if entity.ActualDeliveryDate != nil {
			transitEnd = *entity.ActualDeliveryDate
		}
		transit = []func(){
			func() {
				addSpec(shipment.CommentTypeDriverUpdate, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityNormal, departed, departed+2*shpHour,
					fmt.Sprintf("Loaded and rolling from %s, seal %s.", origin.City.Name, f.digits(7)))
			},
			func() {
				addSpec(shipment.CommentTypeDriverUpdate, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityNormal, departed, transitEnd,
					fmt.Sprintf("Fuel stop, on schedule for %s.", deliveryAppt))
			},
			func() {
				addSpec(shipment.CommentTypeStatusUpdate, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityLow, departed, departed+shpHour,
					fmt.Sprintf("Departed %s at %s.", origin.Name, b.localTime(departed, origin)))
			},
			func() {
				addSpec(shipment.CommentTypeException, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityHigh, departed, transitEnd,
					fmt.Sprintf("Running %d hours behind on weather near %s. New ETA %s.",
						f.between(2, 6), pick(f, cities).Name, deliveryAppt))
			},
			func() {
				addSpec(shipment.CommentTypeException, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityUrgent, departed, transitEnd,
					pick(f, []string{
						"Tractor down with a DEF sensor fault; road service dispatched, ETA 2 hours.",
						"Driver out of hours short of the receiver; delivering first thing tomorrow.",
						fmt.Sprintf("Receiver at %s refused the appointment slot; working a reschedule.", dest.Name),
					}))
			},
			func() {
				addSpec(shipment.CommentTypeCustomerUpdate, shipment.CommentVisibilityCustomer,
					shipment.CommentPriorityNormal, departed, transitEnd,
					fmt.Sprintf("Update sent to %s: in transit, ETA %s.", p.customer.Name, deliveryAppt))
			},
		}
		if p.reefer {
			transit = append(transit, func() {
				addSpec(shipment.CommentTypeDriverUpdate, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityNormal, departed, transitEnd,
					fmt.Sprintf("Reefer holding %d°F, return air steady.", f.between(-5, 38)))
			})
		}
	}
	if entity.ActualDeliveryDate != nil {
		delivered := *entity.ActualDeliveryDate
		postEnd := delivered + 10*shpDay
		if rec.InvoiceDate > 0 {
			postEnd = rec.InvoiceDate + 20*shpDay
		}
		post = []func(){
			func() {
				addSpec(shipment.CommentTypeDocument, shipment.CommentVisibilityOperations,
					shipment.CommentPriorityNormal, delivered, delivered+shpDay,
					fmt.Sprintf("Clean POD signed by %s %s at %s.", f.firstName(), f.lastName(), dest.Name))
			},
			func() {
				addSpec(shipment.CommentTypeCustomerUpdate, shipment.CommentVisibilityCustomer,
					shipment.CommentPriorityNormal, delivered, delivered+4*shpHour,
					fmt.Sprintf("Delivered to %s at %s.", dest.Name, b.localTime(delivered, dest)))
			},
			func() {
				addSpec(shipment.CommentTypeBilling, shipment.CommentVisibilityAccounting,
					shipment.CommentPriorityNormal, delivered, postEnd,
					shpBillingNote(f, p.customer, rec.PONumber, charged))
			},
			func() {
				if charged["DET"] {
					addSpec(shipment.CommentTypeBilling, shipment.CommentVisibilityAccounting,
						shipment.CommentPriorityHigh, delivered, postEnd,
						fmt.Sprintf("Detention at %s approved by %s; added as an accessorial charge.",
							dest.Name, p.customer.Name))
				}
			},
			func() {
				addSpec(shipment.CommentTypeInternal, shipment.CommentVisibilityInternal,
					shipment.CommentPriorityLow, delivered, postEnd,
					fmt.Sprintf("Good lane, add %s to the backhaul list.", lane))
			},
		}
	}
	if entity.CanceledAt != nil {
		addSpec(shipment.CommentTypeDispatch, shipment.CommentVisibilityOperations,
			shipment.CommentPriorityHigh, *entity.CanceledAt, *entity.CanceledAt+shpHour,
			"Canceled: "+entity.CancelReason+".")
	}

	for range count {
		pool := pre
		switch roll := f.float(); {
		case roll < 0.35:
		case roll < 0.7 && len(transit) > 0:
			pool = transit
		case len(post) > 0:
			pool = post
		case len(transit) > 0:
			pool = transit
		}
		pick(f, pool)()
	}
	if len(specs) == 0 {
		return
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].at < specs[j].at })

	var topLevel []*shipment.ShipmentComment
	for i := range specs {
		spec := &specs[i]
		author := b.user()
		c := &shipment.ShipmentComment{
			ID:             idAt("shc_", spec.at),
			BusinessUnitID: b.g.refs.buID,
			OrganizationID: b.g.refs.orgID,
			ShipmentID:     entity.ID,
			UserID:         author.ID,
			Comment:        spec.text,
			Type:           spec.typ,
			Visibility:     spec.visibility,
			Priority:       spec.priority,
			Source:         shipment.CommentSourceUser,
			Metadata:       map[string]any{},
			Version:        1,
			CreatedAt:      spec.at,
		}
		if spec.typ == shipment.CommentTypeStatusUpdate && f.chance(0.6) {
			c.Source = shipment.CommentSourceSystem
			c.UserID = pulid.Nil
		}
		if spec.typ == shipment.CommentTypeDriverUpdate && f.chance(0.4) {
			c.Metadata[shipment.CommentMetadataOrigin] = shipment.CommentOriginDash
		}
		if entity.EntryMethod == shipment.EntryMethodEDI && spec.typ == shipment.CommentTypeAppointment &&
			f.chance(0.5) {
			c.Source = shipment.CommentSourceIntegration
			c.Metadata[shipment.CommentMetadataOrigin] = shipment.CommentOriginEDI
		}

		updated := spec.at
		if len(topLevel) > 0 && c.Source == shipment.CommentSourceUser && f.chance(0.2) {
			parent := pick(f, topLevel)
			c.ParentCommentID = ptr(parent.ID)
			c.Comment = pick(f, shpReplies)
			c.Type = parent.Type
			c.Visibility = parent.Visibility
			c.Priority = shipment.CommentPriorityNormal
		} else {
			topLevel = append(topLevel, c)
		}

		if c.UserID.IsNotNil() && c.ParentCommentID == nil && f.chance(0.12) {
			if mentioned, ok := b.otherUser(c.UserID); ok {
				ask := " please take a look."
				c.Body = map[string]any{
					"type": "doc",
					"content": []any{map[string]any{
						"type": "paragraph",
						"content": []any{
							map[string]any{"type": "text", "text": c.Comment + " "},
							map[string]any{
								"type":  "mention",
								"attrs": map[string]any{"id": mentioned.ID.String(), "label": mentioned.Name},
							},
							map[string]any{"type": "text", "text": ask},
						},
					}},
				}
				c.Comment = c.Comment + " @" + mentioned.Name + ask
				batch.mentions = append(batch.mentions, &shipment.ShipmentCommentMention{
					ID:              idAt("scm_", spec.at),
					CommentID:       c.ID,
					MentionedUserID: mentioned.ID,
					OrganizationID:  b.g.refs.orgID,
					BusinessUnitID:  b.g.refs.buID,
					ShipmentID:      entity.ID,
					CreatedAt:       spec.at,
				})
			}
		}

		if c.ParentCommentID == nil && f.chance(0.04) {
			pinnedAt := min(spec.at+b.at(shpMinute, 6*shpHour), b.now)
			pinner := b.user()
			c.PinnedAt = ptr(pinnedAt)
			c.PinnedByID = ptr(pinner.ID)
			updated = pinnedAt
		}
		if c.Type == shipment.CommentTypeException && c.ParentCommentID == nil {
			if spec.priority == shipment.CommentPriorityUrgent && c.UserID.IsNotNil() && f.chance(0.5) {
				c.RequiresAcknowledgment = true
				seen := map[pulid.ID]bool{c.UserID: true}
				for range f.between(1, 2) {
					u, ok := b.otherUser(c.UserID)
					if !ok || seen[u.ID] {
						continue
					}
					seen[u.ID] = true
					ackAt := spec.at + b.at(2*shpMinute, 4*shpHour)
					if ackAt > b.now {
						continue
					}
					batch.acks = append(batch.acks, &shipment.ShipmentCommentAcknowledgment{
						ID:             idAt("sck_", ackAt),
						CommentID:      c.ID,
						BusinessUnitID: b.g.refs.buID,
						OrganizationID: b.g.refs.orgID,
						ShipmentID:     entity.ID,
						UserID:         u.ID,
						AcknowledgedAt: ackAt,
					})
				}
			}
			if f.chance(0.6) {
				resolvedAt := spec.at + b.at(shpHour, shpDay)
				if resolvedAt <= b.now {
					resolver := b.user()
					c.ResolvedAt = ptr(resolvedAt)
					c.ResolvedByID = ptr(resolver.ID)
					updated = max(updated, resolvedAt)
				}
			}
		}

		batch.comments = append(batch.comments, c)
		batch.stamp(shpTableComments, c.ID, spec.at, updated)
	}
}

func shpDeliveryInstruction(f *faker, dest *locationRecord, charged map[string]bool) string {
	options := []string{
		fmt.Sprintf("Receiver at %s unloads doors 12-18 only. No deliveries after 14:00.", dest.Name),
		fmt.Sprintf("Seal must stay intact until the receiver at %s breaks it.", dest.Name),
	}
	if charged["LUMP"] {
		options = append(options,
			fmt.Sprintf("Lumper service at %s. Get a receipt; it bills back to the customer.", dest.Name))
	}
	if charged["RESDL"] {
		options = append(options, fmt.Sprintf("Liftgate needed at %s, no dock available.", dest.Name))
	}
	return pick(f, options)
}

func shpPORef(prefix, po, suffix string) string {
	if po == "" {
		return ""
	}
	return prefix + po + suffix
}

func shpBillingNote(f *faker, c *customerRecord, po string, charged map[string]bool) string {
	options := []string{"POD and BOL on file; ready to bill."}
	if po != "" {
		options = append(options, fmt.Sprintf("%s requires PO %s on the invoice.", c.Name, po))
	}
	if charged["LUMP"] {
		options = append(options, "Lumper paid at the receiver; bill back at cost.")
	}
	if charged["DET"] {
		options = append(options, "AP asked for the detention log before they release payment.")
	}
	return pick(f, options)
}

func (b *shpBuilder) localTime(at int64, loc *locationRecord) string {
	return time.Unix(at, 0).In(b.zone(loc.City.Timezone)).Format("Mon Jan 2 15:04 MST")
}

func (b *shpBuilder) checkCalls(
	batch *shpBatch,
	entity *shipment.Shipment,
	timeline [][]*shpTimedStop,
) {
	if entity.CanceledAt != nil || entity.ActualShipDate == nil {
		return
	}
	if entity.ActualDeliveryDate != nil && b.now-*entity.ActualDeliveryDate > 21*shpDay {
		return
	}
	f := b.g.fake
	add := func(move *shipment.ShipmentMove, stopID pulid.ID, kind checkcall.Kind, at int64, c *cityRecord) {
		if at > b.now {
			return
		}
		lat, lon := f.jitter(c)
		reporter := b.user()
		id := idAt("chkc_", at)
		received := min(at+f.between64(0, 5*shpMinute), b.now)
		cc := &checkcall.CheckCall{
			ID:             id,
			OrganizationID: b.g.refs.orgID,
			BusinessUnitID: b.g.refs.buID,
			ShipmentID:     entity.ID,
			ShipmentMoveID: move.ID,
			StopID:         stopID,
			Source:         checkcall.SourceDispatcher,
			SourceKey:      "volume:" + id.String(),
			Kind:           kind,
			ReportedAt:     at,
			ReceivedAt:     received,
			Latitude:       ptr(lat),
			Longitude:      ptr(lon),
			City:           c.Name,
			StateCode:      c.State,
			ReportedByID:   reporter.ID,
			CreatedAt:      received,
		}
		if f.chance(0.15) {
			cc.Source = checkcall.SourceAgent
			cc.ReportedByID = pulid.Nil
		}
		batch.checkCalls = append(batch.checkCalls, cc)
	}

	for m, move := range entity.Moves {
		for s, stop := range move.Stops {
			c := timeline[m][s].plan.loc.City
			if stop.ActualArrival != nil {
				add(move, stop.ID, checkcall.KindArrival, *stop.ActualArrival, c)
			}
			if stop.ActualDeparture != nil {
				add(move, stop.ID, checkcall.KindDeparture, *stop.ActualDeparture, c)
			}
			if s == 0 || stop.ActualArrival != nil && f.chance(0.5) {
				continue
			}
			prev := move.Stops[s-1]
			if prev.ActualDeparture == nil {
				continue
			}
			from := timeline[m][s-1].plan.loc.City
			legEnd := b.now
			if stop.ActualArrival != nil {
				legEnd = *stop.ActualArrival
			}
			for range f.between(1, 3) {
				at := b.at(*prev.ActualDeparture+shpHour, legEnd-shpHour)
				if at <= *prev.ActualDeparture || at >= legEnd {
					break
				}
				share := float64(at-*prev.ActualDeparture) / float64(max(legEnd-*prev.ActualDeparture, 1))
				point := shpNearestCity(from, c, share)
				add(move, pulid.Nil, checkcall.KindUpdate, at, point)
				if entity.Status == shipment.StatusDelayed && stop.ActualArrival == nil {
					last := batch.checkCalls[len(batch.checkCalls)-1]
					if last.ShipmentID == entity.ID && last.Kind == checkcall.KindUpdate && !last.Delayed {
						last.Delayed = true
						last.DelayReason = pick(f, shpDelayReasons)
						last.EstimatedArrival = ptr(b.now + f.between64(shpHour, 8*shpHour))
					}
				}
			}
		}
	}
}

func shpNearestCity(from, to *cityRecord, share float64) *cityRecord {
	lat := from.Lat + (to.Lat-from.Lat)*share
	lon := from.Lon + (to.Lon-from.Lon)*share
	point := &cityRecord{Lat: lat, Lon: lon}
	best := &cities[0]
	bestMiles := math.MaxFloat64
	for i := range cities {
		if d := haversineMiles(point, &cities[i]); d < bestMiles {
			best, bestMiles = &cities[i], d
		}
	}
	return best
}

func (b *shpBuilder) holds(batch *shpBatch, entity *shipment.Shipment, rec *shipmentRecord) {
	if len(b.look.holdReasons) == 0 {
		return
	}
	f := b.g.fake
	var started int64
	var released *int64
	switch {
	case entity.Status == shipment.StatusDelayed && f.chance(0.35):
		started = b.at(rec.PickupAt, b.now-shpMinute)
	case rec.DeliveredAt > 0 && b.now-rec.DeliveredAt < 60*shpDay && f.chance(0.03):
		started = b.at(rec.PickupAt, rec.DeliveredAt)
		released = ptr(b.at(started+shpHour, rec.DeliveredAt+shpHour))
	default:
		return
	}
	reason := pick(f, b.look.holdReasons)
	author := b.user()
	h := &shipment.ShipmentHold{
		ID:                idAt("shh_", started),
		ShipmentID:        entity.ID,
		BusinessUnitID:    b.g.refs.buID,
		OrganizationID:    b.g.refs.orgID,
		HoldReasonID:      ptr(reason.ID),
		Type:              reason.Type,
		Severity:          reason.DefaultSeverity,
		ReasonCode:        reason.Code,
		Notes:             pick(f, shpHoldNotes),
		Source:            shipment.HoldSourceUser,
		BlocksDispatch:    reason.DefaultBlocksDispatch,
		BlocksDelivery:    reason.DefaultBlocksDelivery,
		BlocksBilling:     reason.DefaultBlocksBilling,
		VisibleToCustomer: reason.DefaultVisibleToCustomer,
		StartedAt:         started,
		CreatedByID:       ptr(author.ID),
		Version:           1,
	}
	if released != nil {
		batch.releases = append(batch.releases, shpRelease{id: h.ID, at: *released, by: b.user().ID})
	}
	batch.holds = append(batch.holds, h)
	batch.stamp(shpTableHolds, h.ID, started, started)
}

func (b *shpBuilder) write(ctx context.Context, tx bun.Tx, batch *shpBatch) error {
	size := b.g.opts.BatchSize
	steps := []struct {
		table string
		rows  int
		run   func() error
	}{
		{shpTableOrders, len(batch.orders), func() error { return insertBatches(ctx, tx, batch.orders, size) }},
		{shpTableShipments, len(batch.shipments), func() error { return insertBatches(ctx, tx, batch.shipments, size) }},
		{shpTableMoves, len(batch.moves), func() error { return insertBatches(ctx, tx, batch.moves, size) }},
		{shpTableStops, len(batch.stops), func() error { return insertBatches(ctx, tx, batch.stops, size) }},
		{shpTableAssignments, len(batch.assignments), func() error {
			return insertBatches(ctx, tx, batch.assignments, size)
		}},
		{shpTableCarrierAssignments, len(batch.carrierAssignments), func() error {
			return insertBatches(ctx, tx, batch.carrierAssignments, size)
		}},
		{shpTableCommodities, len(batch.commodities), func() error {
			return insertBatches(ctx, tx, batch.commodities, size)
		}},
		{shpTableCharges, len(batch.charges), func() error { return insertBatches(ctx, tx, batch.charges, size) }},
		{shpTableComments, len(batch.comments), func() error { return insertBatches(ctx, tx, batch.comments, size) }},
		{"shipment_comment_mentions", len(batch.mentions), func() error {
			return insertBatches(ctx, tx, batch.mentions, size)
		}},
		{"shipment_comment_acknowledgments", len(batch.acks), func() error {
			return insertBatches(ctx, tx, batch.acks, size)
		}},
		{shpTableHolds, len(batch.holds), func() error { return insertBatches(ctx, tx, batch.holds, size) }},
		{"check_calls", len(batch.checkCalls), func() error {
			return insertBatches(ctx, tx, batch.checkCalls, size)
		}},
	}
	for _, step := range steps {
		if step.rows == 0 {
			continue
		}
		if err := step.run(); err != nil {
			return fmt.Errorf("insert %s: %w", step.table, err)
		}
		b.counts[step.table] += step.rows
	}

	tables := make([]string, 0, len(batch.stamps))
	for table := range batch.stamps {
		tables = append(tables, table)
	}
	slices.Sort(tables)
	for _, table := range tables {
		if err := backdate(ctx, tx, table, b.g.refs.orgID, batch.stamps[table]); err != nil {
			return err
		}
	}

	return b.releaseHolds(ctx, tx, batch.releases)
}

// releaseHolds releases holds after they are inserted and back-dated, because
// protect_shipment_holds rejects any update to a hold once it is released.
func (b *shpBuilder) releaseHolds(ctx context.Context, tx bun.Tx, releases []shpRelease) error {
	if len(releases) == 0 {
		return nil
	}
	ids := make([]string, len(releases))
	ats := make([]int64, len(releases))
	bys := make([]string, len(releases))
	for i, r := range releases {
		ids[i] = r.id.String()
		ats[i] = r.at
		bys[i] = r.by.String()
	}
	if _, err := tx.NewRaw(`
		UPDATE shipment_holds AS t
		SET released_at = d.released_at, released_by_id = d.released_by_id
		FROM unnest(?::varchar[], ?::bigint[], ?::varchar[]) AS d(id, released_at, released_by_id)
		WHERE t.id = d.id AND t.organization_id = ?`,
		pgdialect.Array(ids),
		pgdialect.Array(ats),
		pgdialect.Array(bys),
		b.g.refs.orgID,
	).Exec(ctx); err != nil {
		return fmt.Errorf("release shipment holds: %w", err)
	}

	return nil
}

var shpCancelReasons = []string{
	"Customer canceled the order",
	"Freight not ready, customer rebooked",
	"Duplicate tender",
	"Customer moved the load to another carrier",
	"Rate not accepted",
	"Consignee closed for inventory",
	"Weather closure at origin",
}

var shpReplies = []string{
	"Thanks, noted.",
	"Copy. Updating the customer now.",
	"Driver confirmed.",
	"Done.",
	"Following up with the receiver.",
	"Called the shipper; they confirmed the window.",
	"Approved.",
	"Can we get the lumper receipt uploaded?",
	"Customer is fine with the new ETA.",
	"On it.",
}

var shpDelayReasons = []string{
	"Weather",
	"Traffic",
	"Equipment breakdown",
	"Hours of service",
	"Shipper delay",
	"Receiver delay",
}

var shpHoldNotes = []string{
	"Holding until the receiver confirms a new appointment.",
	"Waiting on customer approval for the reconsignment.",
	"Load secured, waiting on road conditions to clear.",
	"Paperwork mismatch at pickup; waiting on a corrected BOL.",
	"Customer asked us to hold delivery until Monday.",
}
