package volume

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	fuelDay         = int64(86400)
	fuelHour        = int64(3600)
	fuelMinute      = int64(60)
	fuelSampleMiles = 25.0
	fuelHeldWindow  = 30 * fuelDay
	fuelRateNote    = "Volume seed: illustrative rates, not published IFTA rates"
	fuelMilesScale  = ifta.MilesScale
	fuelGallonScale = 3
	fuelMPGScale    = 2
	fuelMoneyScale  = 2
)

var (
	fuelProviders       = []fuelpurchase.CardProvider{fuelpurchase.CardProviderComdata, fuelpurchase.CardProviderEFS, fuelpurchase.CardProviderWEX}
	fuelProviderWeights = []int{60, 25, 15}
	fuelProviderPrefix  = map[fuelpurchase.CardProvider]string{
		fuelpurchase.CardProviderComdata: "CMD",
		fuelpurchase.CardProviderEFS:     "EFS",
		fuelpurchase.CardProviderWEX:     "WEX",
	}
	fuelChains = []string{
		"Pilot Travel Center", "Flying J Travel Center", "Love's Travel Stop", "TA Travel Center",
		"Petro Stopping Center", "Kwik Trip", "Speedway", "Sapp Bros. Travel Center",
		"Road Ranger", "Bosselman Travel Center", "Buc-ee's", "Casey's",
	}
	fuelDieselPrice = map[string]float64{
		"CA": 5.45, "WA": 4.85, "OR": 4.55, "NV": 4.35, "AZ": 3.95, "UT": 3.95, "ID": 3.9,
		"MT": 3.75, "WY": 3.65, "CO": 3.75, "NM": 3.65, "TX": 3.45, "OK": 3.4, "KS": 3.5,
		"NE": 3.55, "SD": 3.6, "ND": 3.6, "MN": 3.7, "IA": 3.55, "MO": 3.45, "AR": 3.45,
		"LA": 3.5, "MS": 3.45, "AL": 3.55, "TN": 3.55, "KY": 3.7, "IL": 3.95, "IN": 3.85,
		"OH": 3.85, "MI": 3.85, "WI": 3.75, "PA": 4.25, "NY": 4.35, "NJ": 3.95, "CT": 4.25,
		"MA": 4.15, "RI": 4.1, "NH": 4.0, "ME": 4.2, "VT": 4.2, "MD": 3.95, "DE": 3.9,
		"VA": 3.8, "WV": 3.85, "NC": 3.75, "SC": 3.6, "GA": 3.6, "FL": 3.75,
	}
	fuelDieselRate = map[string]float64{
		"AL": 0.29, "AZ": 0.26, "AR": 0.285, "CA": 0.97, "CO": 0.2875, "CT": 0.492,
		"DE": 0.22, "FL": 0.388, "GA": 0.364, "ID": 0.32, "IL": 0.744, "IN": 0.59,
		"IA": 0.325, "KS": 0.26, "KY": 0.246, "LA": 0.20, "ME": 0.312, "MD": 0.465,
		"MA": 0.24, "MI": 0.497, "MN": 0.30, "MS": 0.18, "MO": 0.245, "MT": 0.2975,
		"NE": 0.284, "NV": 0.27, "NH": 0.222, "NJ": 0.519, "NM": 0.21, "NY": 0.437,
		"NC": 0.405, "ND": 0.23, "OH": 0.47, "OK": 0.19, "OR": 0.0, "PA": 0.785,
		"RI": 0.37, "SC": 0.28, "SD": 0.28, "TN": 0.27, "TX": 0.20, "UT": 0.365,
		"VT": 0.32, "VA": 0.358, "WA": 0.494, "WV": 0.357, "WI": 0.329, "WY": 0.24,
	}
	fuelCancelReasons = []string{
		"Driver separated; card collected and cancelled",
		"Card reported lost by the driver; replacement issued",
		"Card retired when the unit was sold",
	}
)

type fuelTractor struct {
	id        pulid.ID
	code      string
	fuelType  domaintypes.IFTAFuelType
	qualified bool
	mpg       float64
	odometer  float64
	sinceFill float64
	fillAt    float64
	fills     int
	card      *fuelpurchase.FuelCard
	mateCard  *fuelpurchase.FuelCard
	lastCity  *cityRecord
	lastAt    int64
}

func (t *fuelTractor) milesEligible() bool {
	return t.qualified && t.fuelType.CountsForIFTA()
}

type fuelShare struct {
	state string
	share float64
}

type fuelMileRow struct {
	tractorID      pulid.ID
	jurisdictionID pulid.ID
	miles          decimal.Decimal
	loaded         bool
	year           int
	quarter        int
}

type fuelRouteMove struct {
	tractorID   pulid.ID
	completedAt int64
	miles       decimal.Decimal
	rows        []fuelMileRow
	overridden  bool
}

type fuelUse struct {
	tractorID      pulid.ID
	jurisdictionID pulid.ID
	fuelType       domaintypes.IFTAFuelType
	gallons        decimal.Decimal
	taxPaid        bool
	at             int64
}

type fuelFeedRun struct {
	key       string
	provider  fuelpurchase.CardProvider
	runAt     int64
	batch     *fuelpurchase.ImportBatch
	purchases []*fuelpurchase.FuelPurchase
	held      []*fuelpurchase.ImportRow
	dupes     int
}

type fuelImport struct {
	batches     []*fuelpurchase.ImportBatch
	rows        []*fuelpurchase.ImportRow
	batchStamps []stamp
	rowStamps   []stamp
}

type fuelLegs struct {
	cities []*cityRecord
	miles  []float64
	total  float64
}

type fuelSeeder struct {
	g             *generator
	now           int64
	loc           *time.Location
	baseState     string
	jurisdictions map[string]*ifta.Jurisdiction
	jurisByID     map[pulid.ID]*ifta.Jurisdiction
	tractors      map[pulid.ID]*fuelTractor
	splits        map[[2]*cityRecord][]fuelShare
	lastFours     map[fuelpurchase.CardProvider]map[string]struct{}
	refTag        string
	refSeq        int
	cardOwner     map[pulid.ID]pulid.ID

	cards          []*fuelpurchase.FuelCard
	cardStamps     []stamp
	purchases      []*fuelpurchase.FuelPurchase
	purchaseStamps []stamp
	runs           map[string]*fuelFeedRun
	routeMiles     []*shipment.ShipmentMoveJurisdictionMile
	routeStamps    []stamp
	entries        []*ifta.JurisdictionMileageEntry
	entryStamps    []stamp

	routeMoves []*fuelRouteMove
	manual     []fuelMileRow
	uses       []fuelUse
	periods    []ifta.Period
	hasReturn  map[ifta.Period]bool
	firstStart int64
}

func (g *generator) seedFuel(ctx context.Context) error {
	fs, err := g.fuelLoad(ctx)
	if err != nil {
		return err
	}
	if len(fs.tractors) == 0 || len(fs.jurisdictions) == 0 {
		g.opts.Logf("  ! no tractors or IFTA jurisdictions; fuel and IFTA skipped")
		return nil
	}

	fs.buildCards()
	fs.buildRoutes()
	fs.buildHeldRows()
	imp := fs.finishRuns()

	if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return fs.writeFuel(ctx, tx, imp)
	}); err != nil {
		return err
	}
	g.progress("fuel purchases", len(fs.purchases), len(fs.purchases))

	if err = g.inTx(ctx, fs.writeMiles); err != nil {
		return err
	}
	g.progress("jurisdiction miles", len(fs.routeMiles), len(fs.routeMiles))

	if err = fs.seedReturns(ctx); err != nil {
		return err
	}

	g.report.add("Fuel cards", len(fs.cards))
	g.report.add("Fuel feed runs", len(imp.batches))
	g.report.add("Fuel import rows", len(imp.rows))
	g.report.add("Fuel purchases", len(fs.purchases))
	g.report.add("Move jurisdiction miles", len(fs.routeMiles))
	g.report.add("Jurisdiction mileage entries", len(fs.entries))

	return nil
}

func (g *generator) fuelLoad(ctx context.Context) (*fuelSeeder, error) {
	fs := &fuelSeeder{
		g:             g,
		now:           unix(g.now),
		loc:           time.UTC,
		jurisdictions: make(map[string]*ifta.Jurisdiction),
		jurisByID:     make(map[pulid.ID]*ifta.Jurisdiction),
		tractors:      make(map[pulid.ID]*fuelTractor),
		splits:        make(map[[2]*cityRecord][]fuelShare),
		lastFours:     make(map[fuelpurchase.CardProvider]map[string]struct{}),
		refTag:        fmt.Sprintf("%05d", g.now.Unix()%100_000),
		cardOwner:     make(map[pulid.ID]pulid.ID),
		runs:          make(map[string]*fuelFeedRun),
		hasReturn:     make(map[ifta.Period]bool),
	}

	o := tenantdb.Organizations
	org, err := o.Pick(o.Timezone, o.StateID).Where(o.ID.Eq(g.refs.orgID)).One(ctx, g.db)
	if err != nil {
		return nil, fmt.Errorf("load organization timezone: %w", err)
	}
	if loc, err := time.LoadLocation(org.Timezone); err == nil {
		fs.loc = loc
	}
	fs.baseState = g.refs.stateAbbr[org.StateID]

	ij := iftadb.Jurisdictions
	jurisdictions, err := ij.Where(ij.Status.Eq(ifta.JurisdictionStatusActive)).All(ctx, g.db)
	if err != nil {
		return nil, fmt.Errorf("load ifta jurisdictions: %w", err)
	}
	for _, j := range jurisdictions {
		fs.jurisByID[j.ID] = j
		if j.CountryCode == ifta.CountryCodeUS {
			fs.jurisdictions[j.Code] = j
		}
	}

	tr := tractordb.Tractors
	tractors, err := tr.Pick(tr.ID, tr.Code, tr.FuelType, tr.IFTAQualified).All(ctx, g.db)
	if err != nil {
		return nil, fmt.Errorf("load tractors: %w", err)
	}
	for _, t := range tractors {
		ft := &fuelTractor{
			id:        t.ID,
			code:      t.Code,
			fuelType:  t.FuelType,
			qualified: t.IFTAQualified,
			odometer:  float64(g.fake.between(120_000, 650_000)),
		}
		if ft.fuelType == "" {
			ft.fuelType = domaintypes.IFTAFuelTypeDiesel
		}
		ft.mpg = 6.1 + g.fake.float()*1.5
		if ft.fuelType == domaintypes.IFTAFuelTypeCNG {
			ft.mpg = 5.0 + g.fake.float()*0.9
		}
		ft.fillAt = fs.nextFillAt()
		fs.tractors[t.ID] = ft
	}

	fc := fuelpurchasedb.FuelCards
	keys, err := fc.Pick(fc.Provider, fc.LastFour).All(ctx, g.db)
	if err != nil {
		return nil, fmt.Errorf("load fuel card numbers: %w", err)
	}
	for _, p := range fuelProviders {
		fs.lastFours[p] = make(map[string]struct{})
	}
	for _, k := range keys {
		if fs.lastFours[k.Provider] == nil {
			fs.lastFours[k.Provider] = make(map[string]struct{})
		}
		fs.lastFours[k.Provider][k.LastFour] = struct{}{}
	}

	first := ifta.PeriodOf(unix(g.start), fs.loc)
	current := ifta.PeriodOf(fs.now, fs.loc)
	for p := first; !current.Before(p); p = p.Next() {
		fs.periods = append(fs.periods, p)
	}
	fs.firstStart, _ = first.Bounds(fs.loc)

	ir := iftadb.Returns
	existing, err := ir.Pick(ir.Year, ir.Quarter).All(ctx, g.db)
	if err != nil {
		return nil, fmt.Errorf("load ifta returns: %w", err)
	}
	for _, e := range existing {
		fs.hasReturn[ifta.NewPeriod(int(e.Year), int(e.Quarter))] = true
	}

	if err := fs.loadExistingActivity(ctx); err != nil {
		return nil, err
	}

	return fs, nil
}

// loadExistingActivity reads the purchases and mileage entries already in the
// window, so a return computed here matches what a recompute in the app would
// produce.
func (fs *fuelSeeder) loadExistingActivity(ctx context.Context) error {
	g := fs.g
	fp := fuelpurchasedb.FuelPurchases
	purchases, err := fp.Pick(fp.TractorID, fp.JurisdictionID, fp.FuelType, fp.Gallons, fp.TaxPaid, fp.PurchasedAt).
		Where(fp.PurchasedAt.Gte(fs.firstStart)).
		All(ctx, g.db)
	if err != nil {
		return fmt.Errorf("load existing fuel purchases: %w", err)
	}
	for _, p := range purchases {
		fs.uses = append(fs.uses, fuelUse{
			tractorID:      p.TractorID,
			jurisdictionID: p.JurisdictionID,
			fuelType:       p.FuelType,
			gallons:        p.Gallons,
			taxPaid:        p.TaxPaid,
			at:             p.PurchasedAt,
		})
	}

	me := iftadb.JurisdictionMileageEntries
	entries, err := me.Pick(me.TractorID, me.JurisdictionID, me.Miles, me.Loaded, me.Year, me.Quarter).
		Where(me.TraveledAt.Gte(fs.firstStart)).
		All(ctx, g.db)
	if err != nil {
		return fmt.Errorf("load existing mileage entries: %w", err)
	}
	for _, e := range entries {
		fs.manual = append(fs.manual, fuelMileRow{
			tractorID:      e.TractorID,
			jurisdictionID: e.JurisdictionID,
			miles:          e.Miles,
			loaded:         e.Loaded,
			year:           int(e.Year),
			quarter:        int(e.Quarter),
		})
	}

	return nil
}

func (fs *fuelSeeder) nextFillAt() float64 {
	return 650 + fs.g.fake.float()*650
}

func (fs *fuelSeeder) lastFour(provider fuelpurchase.CardProvider) string {
	taken := fs.lastFours[provider]
	for {
		candidate := fs.g.fake.digits(4)
		if _, ok := taken[candidate]; !ok {
			taken[candidate] = struct{}{}
			return candidate
		}
	}
}

func (fs *fuelSeeder) reference(provider fuelpurchase.CardProvider) string {
	fs.refSeq++
	prefix, ok := fuelProviderPrefix[provider]
	if !ok {
		prefix = "MAN"
	}
	return fmt.Sprintf("%s%s%07d", prefix, fs.refTag, fs.refSeq)
}

func (fs *fuelSeeder) newCard(
	provider fuelpurchase.CardProvider,
	workerID pulid.ID,
	ft *fuelTractor,
	created int64,
) *fuelpurchase.FuelCard {
	g := fs.g
	last := fs.lastFour(provider)
	expires := fs.now + g.fake.between64(200, 1_100)*fuelDay
	card := &fuelpurchase.FuelCard{
		ID:             idAt("fcard_", created),
		OrganizationID: g.refs.orgID,
		BusinessUnitID: g.refs.buID,
		Provider:       provider,
		LastFour:       last,
		ExternalCardID: fmt.Sprintf("%s-%s-%s", fuelProviderPrefix[provider], g.fake.digits(4), last),
		Status:         fuelpurchase.CardStatusActive,
		ExpiresAt:      &expires,
		Label:          provider.Label() + " " + last,
	}
	if workerID.IsNotNil() {
		card.AssignedWorkerID = ptr(workerID)
		fs.cardOwner[card.ID] = workerID
	}
	if ft != nil {
		card.AssignedTractorID = ptr(ft.id)
		card.Label = fmt.Sprintf("%s •••• %s — Unit %s", provider.Label(), last, ft.code)
	}
	card.Normalize()
	fs.cards = append(fs.cards, card)
	fs.cardStamps = append(fs.cardStamps, stamp{id: card.ID, created: created, updated: created})
	return card
}

func (fs *fuelSeeder) buildCards() {
	g := fs.g
	start := unix(g.start)
	hired := make(map[pulid.ID]int64, len(g.drivers))
	for _, d := range g.drivers {
		hired[d.WorkerID] = d.HireDate
	}
	cardCreated := func(workerID pulid.ID) int64 {
		at := start - g.fake.between64(fuelDay, 400*fuelDay)
		if h, ok := hired[workerID]; ok && h > at {
			at = min(h+g.fake.between64(fuelDay, 10*fuelDay), start)
		}
		return at
	}

	crews := append([]*crew(nil), g.crews...)
	sort.Slice(crews, func(i, j int) bool { return crews[i].TractorID < crews[j].TractorID })
	discovered := clamp(len(crews)/40, 2, 25)
	for _, c := range crews {
		ft := fs.tractors[c.TractorID]
		if ft == nil {
			continue
		}
		provider := fuelProviders[g.fake.weighted(fuelProviderWeights)]
		created := cardCreated(c.WorkerID)
		ft.card = fs.newCard(provider, c.WorkerID, ft, created)
		if discovered > 0 && g.fake.chance(0.08) {
			discovered--
			ft.card.DiscoveredAt = ptr(created)
			ft.card.Notes = fmt.Sprintf(
				"First seen on the %s feed; assigned to unit %s.", provider.Label(), ft.code,
			)
		}
		if c.SecondaryWorkerID != nil {
			ft.mateCard = fs.newCard(provider, *c.SecondaryWorkerID, ft, cardCreated(*c.SecondaryWorkerID))
		}
	}

	for _, d := range g.drivers {
		if d.Active || d.TractorID.IsNil() {
			continue
		}
		ft := fs.tractors[d.TractorID]
		created := cardCreated(d.WorkerID)
		card := fs.newCard(fuelProviders[g.fake.weighted(fuelProviderWeights)], d.WorkerID, ft, created)
		cancelled := max(created+30*fuelDay, fs.now-g.fake.between64(5, 300)*fuelDay)
		card.Cancel(min(cancelled, fs.now-fuelHour), pick(g.fake, fuelCancelReasons))
		fs.cardStamps[len(fs.cardStamps)-1].updated = *card.CancelledAt
	}
}

func (fs *fuelSeeder) split(a, b *cityRecord) []fuelShare {
	key := [2]*cityRecord{a, b}
	if shares, ok := fs.splits[key]; ok {
		return shares
	}
	if a.State == b.State {
		shares := []fuelShare{{state: a.State, share: 1}}
		fs.splits[key] = shares
		return shares
	}
	n := max(2, int(haversineMiles(a, b)/fuelSampleMiles))
	counts := make(map[string]int, 4)
	order := make([]string, 0, 4)
	for i := range n {
		state := shpNearestCity(a, b, (float64(i)+0.5)/float64(n)).State
		if counts[state] == 0 {
			order = append(order, state)
		}
		counts[state]++
	}
	shares := make([]fuelShare, 0, len(order))
	for _, state := range order {
		shares = append(shares, fuelShare{state: state, share: float64(counts[state]) / float64(n)})
	}
	fs.splits[key] = shares
	return shares
}

func fuelRoute(cities []*cityRecord) fuelLegs {
	legs := fuelLegs{cities: cities, miles: make([]float64, 0, len(cities))}
	for i := 1; i < len(cities); i++ {
		m := roadMiles(cities[i-1], cities[i])
		legs.miles = append(legs.miles, m)
		legs.total += m
	}
	return legs
}

// stateMiles splits a route across the states it passes through, scaled so the
// rows add up to the move's recorded distance.
func (fs *fuelSeeder) stateMiles(legs fuelLegs, distance float64) ([]string, map[string]float64) {
	order := make([]string, 0, 4)
	byState := make(map[string]float64, 4)
	if legs.total <= 0 || distance <= 0 {
		return order, byState
	}
	scale := distance / legs.total
	for i, m := range legs.miles {
		for _, s := range fs.split(legs.cities[i], legs.cities[i+1]) {
			if _, ok := byState[s.state]; !ok {
				order = append(order, s.state)
			}
			byState[s.state] += m * s.share * scale
		}
	}
	return order, byState
}

func (fs *fuelSeeder) buildRoutes() {
	g := fs.g
	type driverMove struct {
		rec  *shipmentRecord
		move *moveRecord
	}
	byTractor := make(map[pulid.ID][]driverMove)

	for _, rec := range g.shipments {
		if rec.Status == shipment.StatusCanceled {
			continue
		}
		for _, mv := range rec.Moves {
			if mv.Miles <= 0 || len(mv.Cities) < 2 || mv.Status == shipment.MoveStatusCanceled {
				continue
			}
			legs := fuelRoute(mv.Cities)
			order, byState := fs.stateMiles(legs, mv.Miles)
			completed := mv.Status == shipment.MoveStatusCompleted && mv.CompletedAt > 0
			route := &fuelRouteMove{completedAt: mv.CompletedAt}
			if mv.Crew != nil {
				route.tractorID = mv.Crew.TractorID
			}
			calculated := rec.CreatedAt + g.fake.between64(5, 90)
			for seq, state := range order {
				j := fs.jurisdictions[state]
				miles := math.Round(byState[state]*100) / 100
				if j == nil || miles <= 0 {
					continue
				}
				row := &shipment.ShipmentMoveJurisdictionMile{
					ID:               idAt("smjm_", calculated),
					BusinessUnitID:   g.refs.buID,
					OrganizationID:   g.refs.orgID,
					ShipmentMoveID:   mv.ID,
					ShipmentID:       rec.ID,
					CountryCode:      j.CountryCode,
					JurisdictionCode: j.Code,
					Sequence:         seq,
					Distance:         miles,
					DistanceUnits:    shipment.JurisdictionDistanceUnitsMiles,
					Loaded:           true,
					Source:           shipment.JurisdictionMileSourceRouteCalculation,
					CalculatedAt:     calculated,
					Version:          1,
				}
				fs.routeMiles = append(fs.routeMiles, row)
				fs.routeStamps = append(fs.routeStamps, stamp{id: row.ID, created: calculated, updated: calculated})
				dec := decimal.NewFromFloat(miles)
				route.miles = route.miles.Add(dec)
				route.rows = append(route.rows, fuelMileRow{jurisdictionID: j.ID, miles: dec, loaded: true})
			}
			if !completed || len(route.rows) == 0 {
				continue
			}
			fs.routeMoves = append(fs.routeMoves, route)
			if mv.Crew != nil && mv.StartedAt > 0 && mv.CompletedAt > mv.StartedAt {
				byTractor[mv.Crew.TractorID] = append(byTractor[mv.Crew.TractorID], driverMove{rec: rec, move: mv})
				if g.fake.chance(0.03) {
					fs.telematics(route, mv, order, byState)
				}
			}
		}
	}

	ids := make([]pulid.ID, 0, len(byTractor))
	for id := range byTractor {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		ft := fs.tractors[id]
		if ft == nil {
			continue
		}
		moves := byTractor[id]
		sort.Slice(moves, func(i, j int) bool { return moves[i].move.StartedAt < moves[j].move.StartedAt })
		for _, dm := range moves {
			fs.driveMove(ft, dm.rec, dm.move)
		}
	}
}

// telematics replaces a move's route miles with what the ELD measured, the way a
// telematics import overrides the calculated route for IFTA.
func (fs *fuelSeeder) telematics(
	route *fuelRouteMove,
	mv *moveRecord,
	order []string,
	byState map[string]float64,
) {
	g := fs.g
	if mv.Crew == nil {
		return
	}
	traveled := mv.CompletedAt - g.fake.between64(fuelMinute, fuelHour)
	created := min(mv.CompletedAt+g.fake.between64(10*fuelMinute, 6*fuelHour), fs.now)
	added := 0
	for _, state := range order {
		j := fs.jurisdictions[state]
		miles := decimal.NewFromFloat(byState[state] * (0.97 + g.fake.float()*0.06)).Round(fuelMilesScale)
		if j == nil || !miles.IsPositive() {
			continue
		}
		entry := &ifta.JurisdictionMileageEntry{
			ID:             idAt("ifme_", created),
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			TractorID:      mv.Crew.TractorID,
			JurisdictionID: j.ID,
			TraveledAt:     traveled,
			Miles:          miles,
			Loaded:         true,
			Source:         ifta.MileageSourceTelematics,
			ShipmentMoveID: ptr(mv.ID),
			Notes:          "ELD odometer miles for the move, replacing the calculated route",
			Version:        1,
		}
		entry.AssignPeriod(fs.loc)
		entry.Normalize()
		fs.addEntry(entry, created)
		added++
	}
	if added > 0 {
		route.overridden = true
	}
}

func (fs *fuelSeeder) addEntry(entry *ifta.JurisdictionMileageEntry, created int64) {
	fs.entries = append(fs.entries, entry)
	fs.entryStamps = append(fs.entryStamps, stamp{id: entry.ID, created: created, updated: created})
	fs.manual = append(fs.manual, fuelMileRow{
		tractorID:      entry.TractorID,
		jurisdictionID: entry.JurisdictionID,
		miles:          entry.Miles,
		loaded:         entry.Loaded,
		year:           entry.Year,
		quarter:        entry.Quarter,
	})
}

func (fs *fuelSeeder) driveMove(ft *fuelTractor, rec *shipmentRecord, mv *moveRecord) {
	g := fs.g
	origin := mv.Cities[0]
	if ft.lastCity != nil && ft.lastCity != origin && haversineMiles(ft.lastCity, origin) > 25 &&
		mv.StartedAt-ft.lastAt > 3*fuelHour {
		fs.deadhead(ft, origin, mv.StartedAt, rec)
	}

	legs := fuelRoute(mv.Cities)
	span := mv.CompletedAt - mv.StartedAt
	covered := 0.0
	for i, miles := range legs.miles {
		begin := mv.StartedAt + int64(float64(span)*covered/legs.total)
		covered += miles
		end := mv.StartedAt + int64(float64(span)*covered/legs.total)
		fs.walk(ft, mv.Crew, legs.cities[i], legs.cities[i+1], miles, begin, end)
	}

	if mv.Crew.TrailerKind == trailerKindReefer {
		fs.purchase(ft, mv.Crew, origin, mv.StartedAt+g.fake.between64(10*fuelMinute, 50*fuelMinute),
			domaintypes.IFTAFuelTypeReefer, decimal.NewFromFloat(15+g.fake.float()*35).Round(3), false)
	}

	ft.lastCity = mv.Cities[len(mv.Cities)-1]
	ft.lastAt = mv.CompletedAt
}

func (fs *fuelSeeder) deadhead(ft *fuelTractor, to *cityRecord, nextStart int64, rec *shipmentRecord) {
	g := fs.g
	from := ft.lastCity
	miles := roadMiles(from, to)
	gap := nextStart - ft.lastAt
	start := ft.lastAt + g.fake.between64(fuelHour, max(fuelHour, min(gap/3, 12*fuelHour)))
	end := min(start+int64(miles/48*float64(fuelHour)), nextStart-30*fuelMinute)
	if end <= start || start >= fs.now {
		return
	}
	fs.walk(ft, nil, from, to, miles, start, end)

	legs := fuelLegs{cities: []*cityRecord{from, to}, miles: []float64{miles}, total: miles}
	order, byState := fs.stateMiles(legs, miles)
	created := min(end+g.fake.between64(fuelHour, 3*fuelDay), fs.now)
	for _, state := range order {
		j := fs.jurisdictions[state]
		amount := decimal.NewFromFloat(byState[state]).Round(fuelMilesScale)
		if j == nil || !amount.IsPositive() {
			continue
		}
		entry := &ifta.JurisdictionMileageEntry{
			ID:             idAt("ifme_", created),
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			TractorID:      ft.id,
			JurisdictionID: j.ID,
			TraveledAt:     start,
			Miles:          amount,
			Loaded:         false,
			Source:         ifta.MileageSourceManual,
			Notes: fmt.Sprintf("Deadhead %s, %s to %s, %s ahead of %s",
				from.Name, from.State, to.Name, to.State, rec.ProNumber),
			CreatedByID: pick(g.fake, g.refs.userIDs),
			Version:     1,
		}
		entry.AssignPeriod(fs.loc)
		entry.Normalize()
		fs.addEntry(entry, created)
	}
}

func (fs *fuelSeeder) walk(
	ft *fuelTractor,
	c *crew,
	from, to *cityRecord,
	miles float64,
	begin, end int64,
) {
	if miles <= 0 {
		return
	}
	remaining := miles
	covered := 0.0
	for ft.sinceFill+remaining >= ft.fillAt {
		need := ft.fillAt - ft.sinceFill
		covered += need
		remaining -= need
		frac := covered / miles
		at := begin + int64(float64(end-begin)*frac)
		city := shpNearestCity(from, to, frac)
		used := ft.fillAt
		ft.sinceFill = 0
		ft.fillAt = fs.nextFillAt()
		fs.fill(ft, c, city, at, used, ft.odometer+covered)
	}
	ft.sinceFill += remaining
	ft.odometer += miles
}

func (fs *fuelSeeder) fill(ft *fuelTractor, c *crew, city *cityRecord, at int64, used, odometer float64) {
	g := fs.g
	gallons := math.Min(math.Max(used/ft.mpg, 40), 290)
	p := fs.purchase(ft, c, city, at, ft.fuelType, decimal.NewFromFloat(gallons).Round(3), true)
	if p == nil {
		return
	}
	reading := int64(odometer)
	p.Odometer = &reading
	ft.fills++
	if ft.fuelType == domaintypes.IFTAFuelTypeDiesel && ft.fills%3 == 0 {
		fs.purchase(ft, c, city, at+g.fake.between64(2*fuelMinute, 8*fuelMinute),
			domaintypes.IFTAFuelTypeDEF, decimal.NewFromFloat(3+g.fake.float()*9).Round(3), false)
	}
}

func (fs *fuelSeeder) price(state string, at int64, fuelType domaintypes.IFTAFuelType) decimal.Decimal {
	g := fs.g
	base, ok := fuelDieselPrice[state]
	if !ok {
		base = 3.8
	}
	phase := float64(at-unix(g.start)) / float64(365*fuelDay) * 2 * math.Pi
	value := base + 0.28*math.Sin(phase) + (g.fake.float()-0.5)*0.24 - g.fake.float()*0.3
	switch fuelType {
	case domaintypes.IFTAFuelTypeDEF:
		value = 3.0 + g.fake.float()*0.9
	case domaintypes.IFTAFuelTypeCNG:
		value = base*0.72 + g.fake.float()*0.2
	case domaintypes.IFTAFuelTypeReefer:
		value = base - 0.55
	default:
	}
	return decimal.NewFromFloat(math.Max(value, 1.5)).Round(3)
}

// purchase records one fill. Card fills arrive through the provider's nightly
// feed run, so a fill whose run has not happened yet is not in the system.
func (fs *fuelSeeder) purchase(
	ft *fuelTractor,
	c *crew,
	city *cityRecord,
	at int64,
	fuelType domaintypes.IFTAFuelType,
	quantity decimal.Decimal,
	taxPaid bool,
) *fuelpurchase.FuelPurchase {
	g := fs.g
	j := fs.jurisdictions[city.State]
	if j == nil || at <= 0 || at >= fs.now || !quantity.IsPositive() {
		return nil
	}

	card := ft.card
	if c != nil && c.SecondaryWorkerID != nil && ft.mateCard != nil && g.fake.chance(0.35) {
		card = ft.mateCard
	}
	manual := card == nil || card.Status != fuelpurchase.CardStatusActive || g.fake.chance(0.03)

	unitPrice := fs.price(city.State, at, fuelType)
	vendor := pick(g.fake, fuelChains)
	if fuelType == domaintypes.IFTAFuelTypeCNG {
		vendor = "Clean Energy Fuels"
	}
	p := &fuelpurchase.FuelPurchase{
		OrganizationID: g.refs.orgID,
		BusinessUnitID: g.refs.buID,
		TractorID:      ft.id,
		JurisdictionID: j.ID,
		PurchasedAt:    at,
		Vendor:         vendor,
		VendorCity:     city.Name,
		FuelType:       fuelType,
		Quantity:       quantity,
		QuantityUnit:   fuelpurchase.QuantityUnitGallon,
		UnitPrice:      decimal.NewNullDecimal(unitPrice),
		CurrencyCode:   money.DefaultCurrencyCode,
		TaxPaid:        taxPaid,
		Version:        1,
		Source:         fuelpurchase.PurchaseSourceManual,
	}
	p.TotalAmountMinor = money.MinorUnits(quantity.Mul(unitPrice).Round(fuelMoneyScale))

	var created int64
	if manual {
		created = at + g.fake.between64(fuelDay, 4*fuelDay)
		if created >= fs.now {
			return nil
		}
		p.Source = fuelpurchase.PurchaseSourceManual
		p.TransactionReference = fs.reference("")
		p.CreatedByID = pick(g.fake, g.refs.userIDs)
		p.Notes = "Paid at the pump; receipt turned in with the trip envelope"
		if c != nil {
			p.WorkerID = ptr(c.WorkerID)
		}
	} else {
		run := fs.run(card.Provider, at)
		if run == nil {
			return nil
		}
		created = run.runAt
		p.Source = fuelpurchase.PurchaseSourceCardImport
		p.ImportBatchID = ptr(run.batch.ID)
		p.FuelCardID = ptr(card.ID)
		p.CardLastFour = card.LastFour
		p.TransactionReference = fs.reference(card.Provider)
		if owner, ok := fs.cardOwner[card.ID]; ok {
			p.WorkerID = ptr(owner)
		}
		run.purchases = append(run.purchases, p)
	}
	p.ID = idAt("fpur_", created)
	p.Normalize()

	fs.purchases = append(fs.purchases, p)
	fs.purchaseStamps = append(fs.purchaseStamps, stamp{id: p.ID, created: created, updated: created})
	fs.uses = append(fs.uses, fuelUse{
		tractorID:      ft.id,
		jurisdictionID: j.ID,
		fuelType:       fuelType,
		gallons:        p.Gallons,
		taxPaid:        p.TaxPaid,
		at:             at,
	})
	return p
}

func (fs *fuelSeeder) localDay(at int64) time.Time {
	t := time.Unix(at, 0).In(fs.loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, fs.loc)
}

func (fs *fuelSeeder) run(provider fuelpurchase.CardProvider, at int64) *fuelFeedRun {
	day := fs.localDay(at)
	key := string(provider) + "|" + day.Format("2006-01-02")
	if r, ok := fs.runs[key]; ok {
		return r
	}
	g := fs.g
	runAt := day.AddDate(0, 0, 1).Unix() + 4*fuelHour + g.fake.between64(0, 50*fuelMinute)
	if runAt >= fs.now {
		return nil
	}
	batch := &fuelpurchase.ImportBatch{
		ID:              idAt("fpib_", runAt),
		OrganizationID:  g.refs.orgID,
		BusinessUnitID:  g.refs.buID,
		Provider:        provider,
		Origin:          fuelpurchase.ImportOriginFeed,
		FeedReference:   fmt.Sprintf("%s/transactions/%s/%s.csv", strings.ToLower(provider.Label()), day.Format("2006-01-02"), fs.refTag),
		FileName:        fmt.Sprintf("%s_transactions_%s.csv", strings.ToLower(provider.Label()), day.Format("20060102")),
		SourceFormat:    fuelpurchase.SourceFormatCSV,
		DefaultFuelType: domaintypes.IFTAFuelTypeDiesel,
		DefaultCurrency: money.DefaultCurrencyCode,
		Version:         1,
		Status:          fuelpurchase.ImportStatusPending,
	}
	r := &fuelFeedRun{key: key, provider: provider, runAt: runAt, batch: batch}
	fs.runs[key] = r
	return r
}

// buildHeldRows reproduces cards the feed discovered and nobody has assigned yet:
// suspended, unassigned, with their transactions held in open runs because no
// tractor could be determined for them.
func (fs *fuelSeeder) buildHeldRows() {
	g := fs.g
	count := clamp(len(g.crews)/30, 4, 40)
	for range count {
		provider := fuelProviders[g.fake.weighted(fuelProviderWeights)]
		firstSeen := fs.now - g.fake.between64(2*fuelDay, fuelHeldWindow)
		run := fs.run(provider, firstSeen)
		if run == nil {
			continue
		}
		discoveredAt := run.runAt
		last := fs.lastFour(provider)
		card := &fuelpurchase.FuelCard{
			ID:             idAt("fcard_", discoveredAt),
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			Provider:       provider,
			LastFour:       last,
			Label:          provider.Label() + " " + last,
			ExternalCardID: fmt.Sprintf("%s-%s-%s", fuelProviderPrefix[provider], g.fake.digits(4), last),
			Status:         fuelpurchase.CardStatusSuspended,
			DiscoveredAt:   ptr(discoveredAt),
		}
		card.Normalize()
		fs.cards = append(fs.cards, card)
		fs.cardStamps = append(fs.cardStamps, stamp{id: card.ID, created: discoveredAt, updated: discoveredAt})

		at := firstSeen
		for range g.fake.between(2, 6) {
			r := fs.run(provider, at)
			if r == nil {
				break
			}
			city := g.fake.city()
			j := fs.jurisdictions[city.State]
			if j == nil {
				continue
			}
			gallons := decimal.NewFromFloat(60 + g.fake.float()*170).Round(3)
			unitPrice := fs.price(city.State, at, domaintypes.IFTAFuelTypeDiesel)
			parsed := &fuelpurchase.FuelPurchase{
				OrganizationID:       g.refs.orgID,
				BusinessUnitID:       g.refs.buID,
				JurisdictionID:       j.ID,
				FuelCardID:           ptr(card.ID),
				CardLastFour:         last,
				PurchasedAt:          at,
				Vendor:               pick(g.fake, fuelChains),
				VendorCity:           city.Name,
				FuelType:             domaintypes.IFTAFuelTypeDiesel,
				Quantity:             gallons,
				QuantityUnit:         fuelpurchase.QuantityUnitGallon,
				UnitPrice:            decimal.NewNullDecimal(unitPrice),
				TotalAmountMinor:     money.MinorUnits(gallons.Mul(unitPrice).Round(fuelMoneyScale)),
				CurrencyCode:         money.DefaultCurrencyCode,
				TransactionReference: fs.reference(provider),
				Source:               fuelpurchase.PurchaseSourceCardImport,
				ImportBatchID:        ptr(r.batch.ID),
				TaxPaid:              true,
			}
			parsed.Normalize()
			r.held = append(r.held, &fuelpurchase.ImportRow{
				OrganizationID:         g.refs.orgID,
				BusinessUnitID:         g.refs.buID,
				ImportBatchID:          r.batch.ID,
				Parsed:                 parsed,
				TransactionReference:   parsed.TransactionReference,
				Status:                 fuelpurchase.ImportRowStatusError,
				Error:                  "No tractor could be determined for card ending " + last,
				ResolvedFuelCardID:     ptr(card.ID),
				ResolvedJurisdictionID: ptr(j.ID),
			})
			at += g.fake.between64(18*fuelHour, 3*fuelDay)
			if at >= fs.now {
				break
			}
		}
	}
}

func (fs *fuelSeeder) cells(p *fuelpurchase.FuelPurchase, unit string) []string {
	state := ""
	if j := fs.jurisByID[p.JurisdictionID]; j != nil {
		state = j.Code
	}
	price := ""
	if p.UnitPrice.Valid {
		price = p.UnitPrice.Decimal.StringFixed(3)
	}
	return []string{
		time.Unix(p.PurchasedAt, 0).In(fs.loc).Format("01/02/2006 15:04"),
		p.CardLastFour,
		unit,
		p.Vendor,
		p.VendorCity,
		state,
		string(p.FuelType),
		p.Quantity.StringFixed(3),
		price,
		money.DecimalFromMinor(p.TotalAmountMinor).StringFixed(2),
		p.TransactionReference,
	}
}

// finishRuns turns each feed run into its batch and rows, in the state the feed
// leaves them: committed when every row posted, still open while rows are held
// for review, discarded when an older run's leftovers were dismissed.
func (fs *fuelSeeder) finishRuns() *fuelImport {
	g := fs.g
	keys := make([]string, 0, len(fs.runs))
	for key := range fs.runs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	unitOf := make(map[pulid.ID]string, len(fs.tractors))
	for id, t := range fs.tractors {
		unitOf[id] = t.code
	}

	out := &fuelImport{
		batches: make([]*fuelpurchase.ImportBatch, 0, len(keys)),
		rows:    make([]*fuelpurchase.ImportRow, 0, len(fs.purchases)+64),
	}
	for _, key := range keys {
		r := fs.runs[key]
		if len(r.purchases) == 0 && len(r.held) == 0 {
			continue
		}
		sort.SliceStable(r.purchases, func(i, j int) bool {
			return r.purchases[i].PurchasedAt < r.purchases[j].PurchasedAt
		})
		b := r.batch
		summary := &fuelpurchase.ImportSummary{
			ByFuelType:     map[string]string{},
			ByJurisdiction: map[string]string{},
		}
		byFuel := make(map[string]decimal.Decimal, 3)
		byJurisdiction := make(map[string]decimal.Decimal, 8)
		total := decimal.Zero
		number := 0
		runRows := make([]*fuelpurchase.ImportRow, 0, len(r.purchases)+len(r.held)+1)
		for _, p := range r.purchases {
			number++
			parsed := *p
			parsed.ID = pulid.Nil
			parsed.CreatedAt = 0
			parsed.UpdatedAt = 0
			row := &fuelpurchase.ImportRow{
				OrganizationID:         g.refs.orgID,
				BusinessUnitID:         g.refs.buID,
				ImportBatchID:          b.ID,
				RowNumber:              number,
				Cells:                  fs.cells(p, unitOf[p.TractorID]),
				Parsed:                 &parsed,
				TransactionReference:   p.TransactionReference,
				Status:                 fuelpurchase.ImportRowStatusCommitted,
				ResolvedTractorID:      ptr(p.TractorID),
				ResolvedFuelCardID:     p.FuelCardID,
				ResolvedJurisdictionID: ptr(p.JurisdictionID),
				FuelPurchaseID:         ptr(p.ID),
			}
			runRows = append(runRows, row)
			total = total.Add(p.Gallons)
			summary.TotalAmountMinor += p.TotalAmountMinor
			byFuel[string(p.FuelType)] = byFuel[string(p.FuelType)].Add(p.Gallons)
			jkey := fs.jurisByID[p.JurisdictionID].Key()
			byJurisdiction[jkey] = byJurisdiction[jkey].Add(p.Gallons)
			if summary.EarliestPurchasedAt == 0 || p.PurchasedAt < summary.EarliestPurchasedAt {
				summary.EarliestPurchasedAt = p.PurchasedAt
			}
			summary.LatestPurchasedAt = max(summary.LatestPurchasedAt, p.PurchasedAt)
		}
		if len(r.purchases) > 0 && len(r.held) == 0 && g.fake.chance(0.02) {
			src := runRows[g.fake.intn(len(runRows))]
			number++
			parsed := *src.Parsed
			runRows = append(runRows, &fuelpurchase.ImportRow{
				OrganizationID:       g.refs.orgID,
				BusinessUnitID:       g.refs.buID,
				ImportBatchID:        b.ID,
				RowNumber:            number,
				Cells:                src.Cells,
				Parsed:               &parsed,
				TransactionReference: src.TransactionReference,
				Status:               fuelpurchase.ImportRowStatusDuplicateInFile,
				Error:                fmt.Sprintf("Duplicate of row %d", src.RowNumber),
			})
			r.dupes++
		}
		for _, held := range r.held {
			number++
			held.RowNumber = number
			held.Cells = fs.cells(held.Parsed, "")
			runRows = append(runRows, held)
		}
		for _, row := range runRows {
			row.ID = idAt("fpir_", r.runAt)
			out.rowStamps = append(out.rowStamps, stamp{id: row.ID, created: r.runAt})
		}

		summary.RowCount = len(runRows)
		summary.ErrorCount = len(r.held)
		summary.DuplicateInFileCount = r.dupes
		summary.TotalGallons = total.StringFixed(3)
		for k, v := range byFuel {
			summary.ByFuelType[k] = v.StringFixed(3)
		}
		for k, v := range byJurisdiction {
			summary.ByJurisdiction[k] = v.StringFixed(3)
		}

		staged := r.runAt + g.fake.between64(10, 90)
		b.Status = fuelpurchase.ImportStatusParsed
		b.Summary = summary
		b.RowCount = summary.RowCount
		b.ErrorCount = summary.ErrorCount
		b.CommittedCount = len(r.purchases)
		b.StagedAt = ptr(staged)
		updated := staged + g.fake.between64(5, 120)
		switch {
		case len(r.held) == 0 && r.dupes == 0:
			b.Status = fuelpurchase.ImportStatusCommitted
			b.CommittedAt = ptr(updated)
		case r.runAt < fs.now-fuelHeldWindow:
			b.Status = fuelpurchase.ImportStatusDiscarded
			updated = min(r.runAt+g.fake.between64(fuelDay, 6*fuelDay), fs.now-fuelHour)
		}
		b.Normalize()

		out.batches = append(out.batches, b)
		out.rows = append(out.rows, runRows...)
		out.batchStamps = append(out.batchStamps, stamp{id: b.ID, created: r.runAt, updated: updated})
	}

	return out
}

func (fs *fuelSeeder) writeFuel(ctx context.Context, tx *keel.DB, imp *fuelImport) error {
	if err := insertRows(ctx, tx, fuelpurchasedb.FuelCards.Handle, fs.cards); err != nil {
		return fmt.Errorf("insert fuel cards: %w", err)
	}
	if err := backdate(ctx, tx, fuelpurchasedb.FuelCards.Handle, fs.cardStamps); err != nil {
		return err
	}

	if err := insertRows(ctx, tx, fuelpurchasedb.ImportBatches.Handle, imp.batches); err != nil {
		return fmt.Errorf("insert fuel feed runs: %w", err)
	}
	if err := backdate(ctx, tx, fuelpurchasedb.ImportBatches.Handle, imp.batchStamps); err != nil {
		return err
	}

	if err := insertRows(ctx, tx, fuelpurchasedb.FuelPurchases.Handle, fs.purchases); err != nil {
		return fmt.Errorf("insert fuel purchases: %w", err)
	}
	if err := backdate(ctx, tx, fuelpurchasedb.FuelPurchases.Handle, fs.purchaseStamps); err != nil {
		return err
	}

	if err := insertRows(ctx, tx, fuelpurchasedb.ImportRows.Handle, imp.rows); err != nil {
		return fmt.Errorf("insert fuel import rows: %w", err)
	}
	if err := backdateTableCreated(ctx, tx, "fuel_purchase_import_rows", imp.rowStamps); err != nil {
		return err
	}

	return fs.writeFeedStates(ctx, tx, imp.batches)
}

func (fs *fuelSeeder) writeFeedStates(
	ctx context.Context,
	tx *keel.DB,
	batches []*fuelpurchase.ImportBatch,
) error {
	g := fs.g
	latest := make(map[fuelpurchase.CardProvider]*fuelpurchase.ImportBatch, len(fuelProviders))
	for _, b := range batches {
		if cur, ok := latest[b.Provider]; !ok || *b.StagedAt > *cur.StagedAt {
			latest[b.Provider] = b
		}
	}
	states := make([]*fuelpurchase.CardFeedState, 0, len(latest))
	for _, provider := range fuelProviders {
		b, ok := latest[provider]
		if !ok {
			continue
		}
		state := &fuelpurchase.CardFeedState{
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			Provider:       provider,
			FeedType:       fuelpurchase.FeedTypeTransactions,
			Cursor:         b.FeedReference,
			LastPolledAt:   *b.StagedAt,
			LastSuccessAt:  *b.StagedAt,
		}
		if provider == fuelpurchase.CardProviderWEX && g.fake.chance(0.5) {
			state.LastPolledAt = fs.now - g.fake.between64(fuelHour, 6*fuelHour)
			state.FailureCount = g.fake.between(1, 3)
			state.LastError = "WEX returned 503 Service Unavailable; the window is retried on the next run"
		}
		states = append(states, state)
	}
	if len(states) == 0 {
		return nil
	}
	cfs := fuelpurchasedb.CardFeedStates
	if _, err := cfs.CreateOrSkip(ctx, tx, states,
		cfs.OrganizationID, cfs.BusinessUnitID, cfs.Provider, cfs.FeedType,
	); err != nil {
		return fmt.Errorf("insert fuel feed states: %w", err)
	}
	return nil
}

func (fs *fuelSeeder) writeMiles(ctx context.Context, tx *keel.DB) error {
	if err := insertRows(ctx, tx, shipmentdb.ShipmentMoveJurisdictionMiles.Handle, fs.routeMiles); err != nil {
		return fmt.Errorf("insert move jurisdiction miles: %w", err)
	}
	if err := backdate(ctx, tx, shipmentdb.ShipmentMoveJurisdictionMiles.Handle, fs.routeStamps); err != nil {
		return err
	}
	if err := insertRows(ctx, tx, iftadb.JurisdictionMileageEntries.Handle, fs.entries); err != nil {
		return fmt.Errorf("insert jurisdiction mileage entries: %w", err)
	}
	return backdate(ctx, tx, iftadb.JurisdictionMileageEntries.Handle, fs.entryStamps)
}

func (fs *fuelSeeder) rateFor(j *ifta.Jurisdiction, fuelType domaintypes.IFTAFuelType) (rate, surcharge float64) {
	rate = 0.30
	if j.CountryCode == ifta.CountryCodeUS {
		if r, ok := fuelDieselRate[j.Code]; ok {
			rate = r
		}
	} else {
		rate = 0.42
	}
	switch fuelType {
	case domaintypes.IFTAFuelTypeGasoline:
		rate = math.Max(rate-0.02, 0.15)
	case domaintypes.IFTAFuelTypeCNG:
		rate = rate * 0.6
	default:
	}
	if j.HasSurcharge {
		surcharge = 0.11
	}
	return rate, surcharge
}

func (fs *fuelSeeder) seedRates(ctx context.Context) (map[ifta.Period]map[ifta.RateKey]*ifta.TaxRate, error) {
	g := fs.g
	fuelTypes := []domaintypes.IFTAFuelType{
		domaintypes.IFTAFuelTypeDiesel, domaintypes.IFTAFuelTypeGasoline, domaintypes.IFTAFuelTypeCNG,
	}
	jurisdictions := make([]*ifta.Jurisdiction, 0, len(fs.jurisByID))
	for _, j := range fs.jurisByID {
		if j.IsIftaMember {
			jurisdictions = append(jurisdictions, j)
		}
	}
	sort.Slice(jurisdictions, func(i, j int) bool { return jurisdictions[i].ID < jurisdictions[j].ID })

	rows := make([]*ifta.TaxRate, 0, len(fs.periods)*len(jurisdictions)*len(fuelTypes))
	years := make([]int64, 0, 4)
	for _, p := range fs.periods {
		if len(years) == 0 || years[len(years)-1] != int64(p.Year) {
			years = append(years, int64(p.Year))
		}
		start, _ := p.Bounds(fs.loc)
		for _, j := range jurisdictions {
			for _, fuelType := range fuelTypes {
				rate, surcharge := fs.rateFor(j, fuelType)
				r := &ifta.TaxRate{
					ID:             idAt("iftr_", start),
					JurisdictionID: j.ID,
					Year:           p.Year,
					Quarter:        p.Quarter,
					FuelType:       fuelType,
					RatePerGallon:  decimal.NewFromFloat(rate),
					SourceNote:     fuelRateNote,
					Version:        1,
				}
				if surcharge > 0 {
					r.SurchargeRatePerGallon = decimal.NewNullDecimal(decimal.NewFromFloat(surcharge))
				}
				r.Normalize()
				rows = append(rows, r)
			}
		}
	}

	tr := iftadb.TaxRates
	inserted, err := tr.CreateOrSkip(ctx, g.db, rows, tr.JurisdictionID, tr.Year, tr.Quarter, tr.FuelType)
	if err != nil {
		return nil, fmt.Errorf("insert ifta tax rates: %w", err)
	}
	g.report.add("IFTA tax rates", int(inserted))

	var loaded []*ifta.TaxRate
	if len(years) > 0 {
		if loaded, err = tr.Where(tr.Year.In(years...)).All(ctx, g.db); err != nil {
			return nil, fmt.Errorf("load ifta tax rates: %w", err)
		}
	}
	out := make(map[ifta.Period]map[ifta.RateKey]*ifta.TaxRate, len(fs.periods))
	for _, r := range loaded {
		p := r.Period()
		if out[p] == nil {
			out[p] = make(map[ifta.RateKey]*ifta.TaxRate)
		}
		out[p][r.Key()] = r
	}
	return out, nil
}

type fuelLineAcc struct {
	j         *ifta.Jurisdiction
	fuelType  domaintypes.IFTAFuelType
	route     decimal.Decimal
	manual    decimal.Decimal
	loaded    decimal.Decimal
	empty     decimal.Decimal
	taxPaid   decimal.Decimal
	purchases int
}

type fuelTotals struct {
	miles   decimal.Decimal
	gallons decimal.Decimal
}

type fuelComputation struct {
	fs             *fuelSeeder
	period         ifta.Period
	rates          map[ifta.RateKey]*ifta.TaxRate
	lines          map[ifta.RateKey]*fuelLineAcc
	byFuel         map[domaintypes.IFTAFuelType]*fuelTotals
	nonQualMiles   decimal.Decimal
	nonQualGallons decimal.Decimal
	noTractorMiles decimal.Decimal
	noTractorCount int
}

func (c *fuelComputation) line(j *ifta.Jurisdiction, fuelType domaintypes.IFTAFuelType) *fuelLineAcc {
	key := ifta.RateKey{JurisdictionID: j.ID, FuelType: fuelType}
	acc, ok := c.lines[key]
	if !ok {
		acc = &fuelLineAcc{j: j, fuelType: fuelType}
		c.lines[key] = acc
	}
	return acc
}

func (c *fuelComputation) totals(fuelType domaintypes.IFTAFuelType) *fuelTotals {
	t, ok := c.byFuel[fuelType]
	if !ok {
		t = &fuelTotals{}
		c.byFuel[fuelType] = t
	}
	return t
}

func (c *fuelComputation) addMiles(row fuelMileRow, route bool) {
	if row.tractorID.IsNil() || !row.miles.IsPositive() {
		return
	}
	t, ok := c.fs.tractors[row.tractorID]
	if !ok || !t.milesEligible() {
		c.nonQualMiles = c.nonQualMiles.Add(row.miles)
		return
	}
	j := c.fs.jurisByID[row.jurisdictionID]
	if j == nil {
		return
	}
	acc := c.line(j, t.fuelType)
	if route {
		acc.route = acc.route.Add(row.miles)
	} else {
		acc.manual = acc.manual.Add(row.miles)
	}
	if row.loaded {
		acc.loaded = acc.loaded.Add(row.miles)
	} else {
		acc.empty = acc.empty.Add(row.miles)
	}
	c.totals(t.fuelType).miles = c.totals(t.fuelType).miles.Add(row.miles)
}

func (c *fuelComputation) addFuel(use fuelUse) {
	if !use.fuelType.CountsForIFTA() || !use.gallons.IsPositive() {
		return
	}
	t, ok := c.fs.tractors[use.tractorID]
	if !ok || !t.qualified {
		c.nonQualGallons = c.nonQualGallons.Add(use.gallons)
		return
	}
	c.totals(use.fuelType).gallons = c.totals(use.fuelType).gallons.Add(use.gallons)
	j := c.fs.jurisByID[use.jurisdictionID]
	if j == nil {
		return
	}
	acc := c.line(j, use.fuelType)
	acc.purchases++
	if use.taxPaid {
		acc.taxPaid = acc.taxPaid.Add(use.gallons)
	}
}

// compute mirrors iftaservice.Compute, which infrastructure cannot import.
func (c *fuelComputation) compute(ret *ifta.Return) []*ifta.ReturnLine {
	fuelTypes := make([]domaintypes.IFTAFuelType, 0, len(c.byFuel))
	for ft := range c.byFuel {
		fuelTypes = append(fuelTypes, ft)
	}
	sort.Slice(fuelTypes, func(i, j int) bool { return fuelTypes[i] < fuelTypes[j] })

	mpgs := make(map[domaintypes.IFTAFuelType]decimal.Decimal, len(fuelTypes))
	fleet := make([]ifta.FleetMPG, 0, len(fuelTypes))
	fuelProblems := make([]ifta.Problem, 0)
	for _, ft := range fuelTypes {
		t := c.byFuel[ft]
		entry := ifta.FleetMPG{
			FuelType:     ft,
			TotalMiles:   t.miles.StringFixed(fuelMilesScale),
			TotalGallons: t.gallons.StringFixed(fuelGallonScale),
		}
		switch {
		case t.gallons.IsPositive() && t.miles.IsPositive():
			mpg := t.miles.Div(t.gallons).Round(fuelMPGScale)
			mpgs[ft] = mpg
			entry.MPG = mpg.StringFixed(fuelMPGScale)
		case t.miles.IsPositive():
			fuelProblems = append(fuelProblems, ifta.Problem{
				Code: ifta.ProblemNoFuelForType,
				Message: "No " + ft.Label() + " was purchased this quarter, so " +
					"fleet MPG cannot be computed and taxable gallons are zero",
				FuelType: ft,
				Amount:   t.miles.StringFixed(fuelMilesScale),
			})
		default:
			fuelProblems = append(fuelProblems, ifta.Problem{
				Code: ifta.ProblemNoFuelForType,
				Message: ft.Label() + " was purchased but no miles were driven on " +
					ft.Label() + " tractors this quarter",
				FuelType: ft,
				Amount:   t.gallons.StringFixed(fuelGallonScale),
			})
		}
		fleet = append(fleet, entry)
	}

	sorted := make([]*fuelLineAcc, 0, len(c.lines))
	for _, acc := range c.lines {
		sorted = append(sorted, acc)
	}
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.fuelType != b.fuelType {
			return a.fuelType < b.fuelType
		}
		if a.j.CountryCode != b.j.CountryCode {
			return a.j.CountryCode < b.j.CountryCode
		}
		return a.j.Code < b.j.Code
	})

	lines := make([]*ifta.ReturnLine, 0, len(sorted))
	missingRate := make([]ifta.Problem, 0)
	nonMember := make([]ifta.Problem, 0)
	for i, acc := range sorted {
		mpg, hasMPG := mpgs[acc.fuelType]
		line, problems := c.buildLine(acc, mpg, hasMPG)
		line.SortOrder = i
		lines = append(lines, line)
		for _, p := range problems {
			if p.Code == ifta.ProblemMissingRate {
				missingRate = append(missingRate, p)
			} else {
				nonMember = append(nonMember, p)
			}
		}
		ret.TotalMiles = ret.TotalMiles.Add(line.TotalMiles)
		if !line.IsIftaMember {
			continue
		}
		ret.TotalTaxableMiles = ret.TotalTaxableMiles.Add(line.TaxableMiles)
		ret.TotalTaxPaidGallons = ret.TotalTaxPaidGallons.Add(line.TaxPaidGallons)
		ret.NetTaxableGallons = ret.NetTaxableGallons.Add(line.NetTaxableGallons)
		ret.TaxDueMinor += line.TaxDueMinor
		ret.SurchargeDueMinor += line.SurchargeDueMinor
	}
	for _, t := range c.byFuel {
		ret.TotalGallons = ret.TotalGallons.Add(t.gallons)
	}
	ret.NetDueMinor = ret.TaxDueMinor + ret.SurchargeDueMinor
	ret.FleetMPGByFuelType = fleet
	ret.NoTractorMiles = c.noTractorMiles.Round(fuelMilesScale)
	ret.NoTractorMoveCount = c.noTractorCount
	ret.UnattributedMiles = decimal.Zero

	problems := make([]ifta.Problem, 0, len(missingRate)+len(fuelProblems)+len(nonMember)+2)
	problems = append(problems, missingRate...)
	problems = append(problems, fuelProblems...)
	if c.noTractorCount > 0 {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNoTractorMiles,
			Message: fmt.Sprintf("%d completed moves totalling ", c.noTractorCount) +
				c.noTractorMiles.StringFixed(fuelMilesScale) +
				" miles have no tractor assigned and cannot be attributed to a fuel type",
			Amount: c.noTractorMiles.StringFixed(fuelMilesScale),
		})
	}
	problems = append(problems, nonMember...)
	if c.nonQualMiles.IsPositive() || c.nonQualGallons.IsPositive() {
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNonQualifiedActivity,
			Message: c.nonQualMiles.StringFixed(fuelMilesScale) + " miles and " +
				c.nonQualGallons.StringFixed(fuelGallonScale) +
				" gallons on tractors that are not IFTA qualified were left off the return",
			Amount: c.nonQualMiles.StringFixed(fuelMilesScale),
		})
	}
	ret.Problems = problems

	return lines
}

func (c *fuelComputation) buildLine(
	acc *fuelLineAcc,
	mpg decimal.Decimal,
	hasMPG bool,
) (*ifta.ReturnLine, []ifta.Problem) {
	j := acc.j
	problems := make([]ifta.Problem, 0, 2)
	totalMiles := acc.route.Add(acc.manual).Round(0)
	line := &ifta.ReturnLine{
		JurisdictionID:    j.ID,
		FuelType:          acc.fuelType,
		IsIftaMember:      j.IsIftaMember,
		TotalMiles:        totalMiles,
		RouteMiles:        acc.route.Round(fuelMilesScale),
		ManualMiles:       acc.manual.Round(fuelMilesScale),
		LoadedMiles:       acc.loaded.Round(fuelMilesScale),
		EmptyMiles:        acc.empty.Round(fuelMilesScale),
		TaxPaidGallonsRaw: acc.taxPaid.Round(fuelGallonScale),
		PurchaseCount:     acc.purchases,
		TaxableMiles:      decimal.Zero,
		TaxPaidGallons:    decimal.Zero,
		TaxableGallons:    decimal.Zero,
		NetTaxableGallons: decimal.Zero,
	}

	if !j.IsIftaMember {
		line.RatePerGallon = decimal.NewNullDecimal(decimal.Zero)
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemNonMemberActivity,
			Message: j.Label() + " is not an IFTA member; its " +
				totalMiles.String() + " miles count toward fleet MPG but carry no tax",
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
			Amount:           totalMiles.StringFixed(fuelMilesScale),
		})
		return line, problems
	}

	line.TaxableMiles = totalMiles
	line.TaxPaidGallons = acc.taxPaid.Round(0)
	if hasMPG && mpg.IsPositive() {
		line.TaxableGallons = line.TaxableMiles.Div(mpg).Round(0)
	}
	line.NetTaxableGallons = line.TaxableGallons.Sub(line.TaxPaidGallons)

	rate := c.rates[ifta.RateKey{JurisdictionID: j.ID, FuelType: acc.fuelType}]
	if rate == nil {
		line.RateMissing = true
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemMissingRate,
			Message: "No " + acc.fuelType.Label() + " tax rate is on file for " + j.Label() +
				" in " + c.period.Label(),
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
		})
		return line, problems
	}

	line.RatePerGallon = decimal.NewNullDecimal(rate.RatePerGallon)
	line.TaxDueMinor = money.MinorUnits(line.NetTaxableGallons.Mul(rate.RatePerGallon).Round(fuelMoneyScale))
	switch {
	case rate.HasSurcharge():
		line.SurchargeRatePerGallon = rate.SurchargeRatePerGallon
		surcharge := line.TaxableGallons.Mul(rate.SurchargeRate()).Round(fuelMoneyScale)
		if surcharge.IsNegative() {
			surcharge = decimal.Zero
		}
		line.SurchargeDueMinor = money.MinorUnits(surcharge)
	case j.HasSurcharge:
		problems = append(problems, ifta.Problem{
			Code: ifta.ProblemMissingRate,
			Message: j.Label() + " charges a surcharge but no " + acc.fuelType.Label() +
				" surcharge rate is on file for " + c.period.Label(),
			JurisdictionCode: j.Key(),
			FuelType:         acc.fuelType,
		})
	}
	line.LineTotalMinor = line.TaxDueMinor + line.SurchargeDueMinor

	return line, problems
}

// seedReturns files one return per quarter the history covers: filed once its
// due date has passed, finalized while it is still due, a draft for the quarter
// in progress, and a draft wherever a missing rate would block finalizing.
func (fs *fuelSeeder) seedReturns(ctx context.Context) error {
	g := fs.g
	rates, err := fs.seedRates(ctx)
	if err != nil {
		return err
	}
	current := ifta.PeriodOf(fs.now, fs.loc)

	returns := make([]*ifta.Return, 0, len(fs.periods))
	lines := make([]*ifta.ReturnLine, 0, 256)
	returnStamps := make([]stamp, 0, len(fs.periods))
	lineStamps := make([]stamp, 0, 256)
	for _, p := range fs.periods {
		if fs.hasReturn[p] {
			continue
		}
		start, end := p.Bounds(fs.loc)
		c := &fuelComputation{
			fs:     fs,
			period: p,
			rates:  rates[p],
			lines:  make(map[ifta.RateKey]*fuelLineAcc),
			byFuel: make(map[domaintypes.IFTAFuelType]*fuelTotals, 2),
		}
		for _, m := range fs.routeMoves {
			if m.overridden || m.completedAt < start || m.completedAt >= end {
				continue
			}
			if m.tractorID.IsNil() {
				c.noTractorCount++
				c.noTractorMiles = c.noTractorMiles.Add(m.miles)
				continue
			}
			for _, row := range m.rows {
				row.tractorID = m.tractorID
				c.addMiles(row, true)
			}
		}
		for _, row := range fs.manual {
			if row.year == p.Year && row.quarter == p.Quarter {
				c.addMiles(row, false)
			}
		}
		for _, use := range fs.uses {
			if use.at >= start && use.at < end {
				c.addFuel(use)
			}
		}
		if len(c.lines) == 0 && c.noTractorCount == 0 {
			continue
		}

		ret := &ifta.Return{
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			Year:           p.Year,
			Quarter:        p.Quarter,
			Status:         ifta.ReturnStatusDraft,
			Timezone:       fs.loc.String(),
			PeriodStart:    start,
			PeriodEnd:      end,
			CurrencyCode:   money.DefaultCurrencyCode,
			Version:        1,
		}
		retLines := c.compute(ret)

		due := p.DueDate(fs.loc)
		var computed int64
		switch {
		case p.Equal(current):
			computed = max(start, fs.now-g.fake.between64(fuelHour, 2*fuelDay))
		case len(ret.BlockingProblems()) > 0:
			computed = min(end+g.fake.between64(3*fuelDay, 10*fuelDay), fs.now-fuelHour)
		case fs.now < due:
			finalized := min(end+g.fake.between64(6*fuelDay, 20*fuelDay), fs.now-fuelHour)
			ret.Status = ifta.ReturnStatusFinalized
			ret.FinalizedAt = ptr(finalized)
			ret.FinalizedByID = g.refs.adminID
			computed = finalized - g.fake.between64(5*fuelMinute, 2*fuelHour)
		default:
			finalized := end + g.fake.between64(8*fuelDay, 20*fuelDay)
			filed := min(finalized+g.fake.between64(fuelDay, 6*fuelDay), due-fuelDay)
			filed = max(filed, finalized)
			ret.Status = ifta.ReturnStatusFiled
			ret.FinalizedAt = ptr(finalized)
			ret.FinalizedByID = g.refs.adminID
			ret.FiledAt = ptr(filed)
			ret.FiledByID = g.refs.adminID
			ret.FilingReference = fmt.Sprintf("IFTA-%s-%d%d-%s", fs.baseState, p.Year, p.Quarter, g.fake.digits(8))
			computed = finalized - g.fake.between64(5*fuelMinute, 2*fuelHour)
		}
		ret.ComputedAt = ptr(computed)
		ret.ID = idAt("ifr_", computed)
		ret.Normalize()
		updated := computed
		if ret.FiledAt != nil {
			updated = *ret.FiledAt
		} else if ret.FinalizedAt != nil {
			updated = *ret.FinalizedAt
		}
		returns = append(returns, ret)
		returnStamps = append(returnStamps, stamp{id: ret.ID, created: computed, updated: updated})
		for _, line := range retLines {
			line.ID = idAt("ifrl_", computed)
			line.OrganizationID = g.refs.orgID
			line.BusinessUnitID = g.refs.buID
			line.ReturnID = ret.ID
			lines = append(lines, line)
			lineStamps = append(lineStamps, stamp{id: line.ID, created: computed, updated: computed})
		}
	}
	if len(returns) == 0 {
		return nil
	}

	if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if err := insertRows(ctx, tx, iftadb.Returns.Handle, returns); err != nil {
			return fmt.Errorf("insert ifta returns: %w", err)
		}
		if err := insertRows(ctx, tx, iftadb.ReturnLines.Handle, lines); err != nil {
			return fmt.Errorf("insert ifta return lines: %w", err)
		}
		if err := backdate(ctx, tx, iftadb.Returns.Handle, returnStamps); err != nil {
			return err
		}
		return backdate(ctx, tx, iftadb.ReturnLines.Handle, lineStamps)
	}); err != nil {
		return err
	}
	g.report.add("IFTA returns", len(returns))
	g.report.add("IFTA return lines", len(lines))

	return nil
}
