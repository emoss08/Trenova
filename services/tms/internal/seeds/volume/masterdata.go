package volume

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer/equipmentmanufacturerdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory/locationcategorydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const mdDay = int64(86_400)

type mdDriver struct {
	worker  *worker.Worker
	profile *worker.WorkerProfile
	home    *cityRecord
}

type mdEquipment struct {
	tractorTypes  []*equipmenttype.EquipmentType
	trailerTypes  []*equipmenttype.EquipmentType
	kinds         map[pulid.ID]trailerKind
	manufacturers map[string]pulid.ID
	fleetCodes    []pulid.ID
}

func (g *generator) seedMasterData(ctx context.Context) error {
	return g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		steps := []func(context.Context, *keel.DB) error{
			g.mdCustomers,
			g.facMasterData,
			g.mdLocations,
			g.mdCommodities,
		}
		for _, step := range steps {
			if err := step(ctx, tx); err != nil {
				return err
			}
		}

		equipment, err := g.mdEquipment(ctx, tx)
		if err != nil {
			return err
		}
		drivers, err := g.mdWorkers(ctx, tx, equipment.fleetCodes)
		if err != nil {
			return err
		}
		if err = g.mdFleet(ctx, tx, equipment, drivers); err != nil {
			return err
		}

		return g.mdCarriers(ctx, tx)
	})
}

func (g *generator) mdCreated() int64 {
	return g.fake.between64(unix(g.start.AddDate(-2, 0, 0)), unix(g.start))
}

func (g *generator) mdUpdated(created int64) int64 {
	return g.fake.between64(created, unix(g.now))
}

func (g *generator) mdStamp(id pulid.ID, created int64) stamp {
	return stamp{id: id, created: created, updated: g.mdUpdated(created)}
}

func mdTaken[T any](
	ctx context.Context,
	db *keel.DB,
	h keel.Handle[*T],
	column keel.Expr,
) (map[string]struct{}, error) {
	values, err := keel.Into[string](ctx, db, h.Pick(keel.StringOf(column).Lower().Coalesce("")))
	if err != nil {
		return nil, fmt.Errorf("load existing %s values: %w", h.Ref.Table.Name, err)
	}
	taken := make(map[string]struct{}, len(values))
	for _, v := range values {
		if v != "" {
			taken[v] = struct{}{}
		}
	}

	return taken, nil
}

func mdSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
		if b.Len() == 20 {
			break
		}
	}
	return b.String()
}

func (g *generator) mdTimes(rows int, prefix string) ([]int64, []pulid.ID) {
	created := make([]int64, rows)
	ids := make([]pulid.ID, rows)
	for i := range rows {
		created[i] = g.mdCreated()
		ids[i] = idAt(prefix, created[i])
	}
	return created, ids
}

var mdPaymentTerms = []customer.PaymentTerm{
	customer.PaymentTermNet30,
	customer.PaymentTermNet45,
	customer.PaymentTermNet15,
	customer.PaymentTermNet60,
	customer.PaymentTermDueOnReceipt,
	customer.PaymentTermNet10,
	customer.PaymentTermNet90,
}

var mdPaymentTermWeights = []int{50, 15, 10, 10, 5, 5, 5}

func (g *generator) mdCustomers(ctx context.Context, tx *keel.DB) error {
	taken, err := mdTaken(ctx, tx, customerdb.Customers.Handle, customerdb.Customers.Code)
	if err != nil {
		return err
	}

	n := g.scale.customers
	created, ids := g.mdTimes(n, "cus_")
	customers := make([]*customer.Customer, 0, n)
	profiles := make([]*customer.CustomerBillingProfile, 0, n)
	emails := make([]*customer.CustomerEmailProfile, 0, n)
	stamps := make([]stamp, 0, n)
	profileStamps := make([]stamp, 0, n)
	emailStamps := make([]stamp, 0, n)
	g.customers = make([]*customerRecord, 0, n)

	for i := range n {
		name := g.fake.companyName()
		home := g.fake.city()
		lat, lon := g.fake.jitter(home)
		status := domaintypes.StatusActive
		if g.fake.chance(0.06) {
			status = domaintypes.StatusInactive
		}

		entity := &customer.Customer{
			ID:                     ids[i],
			BusinessUnitID:         g.refs.buID,
			OrganizationID:         g.refs.orgID,
			StateID:                g.refs.stateIDs[home.State],
			Status:                 status,
			Code:                   uniqueCode(codeFrom(name, g.fake.between(4, 6)), taken),
			Name:                   name,
			AddressLine1:           g.fake.street(),
			City:                   home.Name,
			PostalCode:             g.fake.postalCode(home),
			IsGeocoded:             true,
			Latitude:               &lat,
			Longitude:              &lon,
			StatusUpdatePreference: pick(g.fake, mdStatusUpdatePreferences),
			AllowConsolidation:     g.fake.chance(0.7),
			ConsolidationPriority:  g.fake.between(1, 5),
		}
		if g.fake.chance(0.15) {
			entity.AddressLine2 = fmt.Sprintf("Suite %d", g.fake.between(100, 950))
		}
		if g.fake.chance(0.4) {
			entity.ExternalID = "ERP-" + g.fake.digits(6)
		}
		if entity.StatusUpdatePreference != customer.StatusUpdateNone {
			entity.StatusUpdateRecipients = "tracking@" + mdSlug(name) + ".example.com"
		}
		if g.fake.chance(0.3) {
			entity.CheckCallIntervalHours = ptr(int16(pick(g.fake, []int{2, 4, 6, 8, 12})))
		}
		if g.fake.chance(0.25) {
			entity.EtaRiskThresholdMinutes = ptr(int16(pick(g.fake, []int{30, 60, 90, 120})))
		}

		term := mdPaymentTerms[g.fake.weighted(mdPaymentTermWeights)]
		profile := customer.NewDefaultBillingProfile(g.refs.orgID, g.refs.buID, entity.ID)
		profile.ID = idAt("cbp_", created[i])
		profile.PaymentTerm = term
		g.mdVaryBillingProfile(profile, name)

		email := &customer.CustomerEmailProfile{
			ID:                    idAt("cem_", created[i]),
			BusinessUnitID:        g.refs.buID,
			OrganizationID:        g.refs.orgID,
			CustomerID:            entity.ID,
			Subject:               "Invoice from " + g.refs.orgName,
			ToRecipients:          "ap@" + mdSlug(name) + ".example.com",
			AttachmentName:        "Invoice",
			IncludeShipmentDetail: g.fake.chance(0.5),
		}
		if g.fake.chance(0.3) {
			email.CCRecipients = "billing@" + mdSlug(name) + ".example.com"
		}

		weight := max(1, int(math.Round(900*math.Pow(float64(i+1), -1.05))))
		if status == domaintypes.StatusInactive {
			weight = 1
		}

		customers = append(customers, entity)
		profiles = append(profiles, profile)
		stamps = append(stamps, g.mdStamp(entity.ID, created[i]))
		profileStamps = append(profileStamps, g.mdStamp(profile.ID, created[i]))
		billingEmail := ""
		if g.fake.chance(0.75) {
			billingEmail = email.ToRecipients
			emails = append(emails, email)
			emailStamps = append(emailStamps, g.mdStamp(email.ID, created[i]))
		} else {
			profile.EmailInvoiceEnabled = false
			profile.AutoSendInvoiceOnGeneration = false
		}
		g.customers = append(g.customers, &customerRecord{
			ID:           entity.ID,
			Code:         entity.Code,
			Name:         entity.Name,
			AddressLine1: entity.AddressLine1,
			City:         home.Name,
			StateAbbr:    home.State,
			PostalCode:   entity.PostalCode,
			BillingEmail: billingEmail,
			PaymentTerm:  invoice.PaymentTerm(term),
			Weight:       weight,
			Home:         home,
		})
	}

	if err = insertRows(ctx, tx, customerdb.Customers.Handle, customers); err != nil {
		return fmt.Errorf("insert customers: %w", err)
	}
	if err = insertRows(ctx, tx, customerdb.CustomerBillingProfiles.Handle, profiles); err != nil {
		return fmt.Errorf("insert customer billing profiles: %w", err)
	}
	if err = insertRows(ctx, tx, customerdb.CustomerEmailProfiles.Handle, emails); err != nil {
		return fmt.Errorf("insert customer email profiles: %w", err)
	}
	for _, b := range []struct {
		table  string
		stamps []stamp
	}{
		{table: "customers", stamps: stamps},
		{table: "customer_billing_profiles", stamps: profileStamps},
		{table: "customer_email_profiles", stamps: emailStamps},
	} {
		if err = backdateTable(ctx, tx, b.table, b.stamps); err != nil {
			return err
		}
	}

	g.report.add("Customers", len(customers))
	g.report.add("Customer billing profiles", len(profiles))
	g.report.add("Customer email profiles", len(emails))

	return nil
}

var mdStatusUpdatePreferences = []customer.StatusUpdatePreference{
	customer.StatusUpdateNone,
	customer.StatusUpdateNone,
	customer.StatusUpdateArrivals,
	customer.StatusUpdateDepartures,
	customer.StatusUpdateArrivalsAndDepartures,
}

func (g *generator) mdVaryBillingProfile(profile *customer.CustomerBillingProfile, name string) {
	if g.fake.chance(0.6) {
		limit := decimal.NewFromInt(int64(g.fake.between(5, 100)) * 5_000)
		profile.CreditLimit = decimal.NullDecimal{Decimal: limit, Valid: true}
	}
	switch roll := g.fake.intn(100); {
	case roll < 3:
		profile.CreditStatus = customer.CreditStatusHold
		profile.CreditHoldReason = "Past due balance over 60 days"
	case roll < 9:
		profile.CreditStatus = customer.CreditStatusWarning
	case roll < 12:
		profile.CreditStatus = customer.CreditStatusReview
	default:
		profile.CreditStatus = customer.CreditStatusActive
	}
	if g.fake.chance(0.2) {
		profile.InvoiceDetail = customer.InvoiceDetailSummary
	}
	profile.RequirePONumber = g.fake.chance(0.15)
	profile.RequireBOLNumber = g.fake.chance(0.25)
	profile.SendStatements = g.fake.chance(0.7)
	profile.SendPaymentReminders = g.fake.chance(0.8)
	if g.fake.chance(0.1) {
		profile.ApplyLateCharges = true
		profile.LateChargeRate = decimal.NullDecimal{
			Decimal: decimal.NewFromFloat(1.5),
			Valid:   true,
		}
		profile.GracePeriodDays = int16(g.fake.between(5, 15))
	}
	if g.fake.chance(0.05) {
		profile.TaxExempt = true
		profile.TaxExemptNumber = "TX-" + g.fake.digits(9)
	}
	if g.fake.chance(0.2) {
		profile.BillingNotes = pick(g.fake, []string{
			"Send invoices with signed POD attached.",
			"AP portal upload required in addition to email.",
			"Lumper receipts must accompany every invoice.",
			"Bill " + name + " corporate, not the ship-from facility.",
			"Detention billed only with in/out times on the BOL.",
		})
	}
}

type mdLocationKind struct {
	category     string
	catType      locationcategory.Category
	facilityType locationcategory.FacilityType
	color        string
	shipper      bool
	consignee    bool
	weight       int
	appointment  bool
	owned        bool
	name         func(g *generator, c *cityRecord, owner string) string
}

var mdLocationKinds = []mdLocationKind{
	{
		category: "Distribution Center", catType: locationcategory.CategoryDistributionCenter,
		facilityType: locationcategory.FacilityTypeCrossDock, color: "#2563EB",
		shipper: true, consignee: true, weight: 20, appointment: true, owned: true,
		name: func(g *generator, _ *cityRecord, owner string) string {
			return owner + " Distribution Center"
		},
	},
	{
		category: "Manufacturing Plant", catType: locationcategory.CategoryCustomerLocation,
		color: "#7C3AED", shipper: true, weight: 18, owned: true,
		name: func(g *generator, c *cityRecord, owner string) string {
			return owner + " " + c.Name + " Plant"
		},
	},
	{
		category: "Storage Warehouse", catType: locationcategory.CategoryWarehouse,
		facilityType: locationcategory.FacilityTypeStorageWarehouse, color: "#0891B2",
		shipper: true, consignee: true, weight: 18, owned: true,
		name: func(g *generator, _ *cityRecord, owner string) string {
			return owner + " Warehouse"
		},
	},
	{
		category: "Cold Storage", catType: locationcategory.CategoryWarehouse,
		facilityType: locationcategory.FacilityTypeColdStorage, color: "#0EA5E9",
		shipper: true, consignee: true, weight: 8, appointment: true,
		name: func(g *generator, c *cityRecord, owner string) string {
			return pick(g.fake, companyPrefixes) + " " + c.Name + " Cold Storage"
		},
	},
	{
		category: "Cross-Dock", catType: locationcategory.CategoryWarehouse,
		facilityType: locationcategory.FacilityTypeCrossDock, color: "#14B8A6",
		shipper: true, consignee: true, weight: 6,
		name: func(g *generator, c *cityRecord, owner string) string {
			return pick(g.fake, companyPrefixes) + " " + c.Name + " Cross-Dock"
		},
	},
	{
		category: "Port Terminal", catType: locationcategory.CategoryPort,
		facilityType: locationcategory.FacilityTypeIntermodalFacility, color: "#1D4ED8",
		shipper: true, consignee: true, weight: 3, appointment: true,
		name: func(g *generator, c *cityRecord, owner string) string {
			return fmt.Sprintf("Port of %s Terminal %d", c.Name, g.fake.between(1, 9))
		},
	},
	{
		category: "Rail Ramp", catType: locationcategory.CategoryRailYard,
		facilityType: locationcategory.FacilityTypeIntermodalFacility, color: "#B45309",
		shipper: true, consignee: true, weight: 3,
		name: func(g *generator, c *cityRecord, owner string) string {
			return c.Name + " " + pick(g.fake, []string{"BNSF", "UP", "CSX", "NS"}) + " Rail Ramp"
		},
	},
	{
		category: "Retail Store", catType: locationcategory.CategoryCustomerLocation,
		color: "#DB2777", consignee: true, weight: 14,
		name: func(g *generator, _ *cityRecord, owner string) string {
			return fmt.Sprintf("%s Store #%d", owner, g.fake.between(100, 9999))
		},
	},
	{
		category: "Grocery Distribution", catType: locationcategory.CategoryDistributionCenter,
		facilityType: locationcategory.FacilityTypeColdStorage, color: "#16A34A",
		consignee: true, weight: 6, appointment: true,
		name: func(g *generator, c *cityRecord, owner string) string {
			return pick(g.fake, companyPrefixes) + " Grocers " + c.Name + " DC"
		},
	},
	{
		category: "Truck Stop", catType: locationcategory.CategoryTruckStop, color: "#64748B",
		weight: 2,
		name: func(g *generator, c *cityRecord, owner string) string {
			return pick(g.fake, []string{"Pilot", "Love's", "TA", "Petro"}) + " " + c.Name
		},
	},
}

func (g *generator) mdLocationOwner(
	kind *mdLocationKind,
	weights []int,
) (*customerRecord, *cityRecord) {
	if !kind.owned || len(g.customers) == 0 || !g.fake.chance(0.6) {
		return nil, g.fake.city()
	}
	owner := g.customers[g.fake.weighted(weights)]
	if owner.Home == nil {
		return owner, g.fake.city()
	}
	if g.fake.chance(0.4) {
		return owner, owner.Home
	}
	return owner, g.fake.cityNear(owner.Home, 400)
}

func (g *generator) mdLocationCategories(
	ctx context.Context,
	tx *keel.DB,
) (map[string]pulid.ID, error) {
	existing, err := locationcategorydb.LocationCategories.Query().All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load location categories: %w", err)
	}
	byName := make(map[string]pulid.ID, len(existing)+len(mdLocationKinds))
	for _, c := range existing {
		byName[strings.ToLower(c.Name)] = c.ID
	}

	created := make([]*locationcategory.LocationCategory, 0, len(mdLocationKinds))
	stamps := make([]stamp, 0, len(mdLocationKinds))
	for _, kind := range mdLocationKinds {
		key := strings.ToLower(kind.category)
		if _, ok := byName[key]; ok {
			continue
		}
		at := g.mdCreated()
		category := &locationcategory.LocationCategory{
			ID:                  idAt("lc_", at),
			BusinessUnitID:      g.refs.buID,
			OrganizationID:      g.refs.orgID,
			Name:                kind.category,
			Description:         kind.category + " facilities",
			Type:                kind.catType,
			FacilityType:        kind.facilityType,
			Color:               kind.color,
			RequiresAppointment: kind.appointment,
			HasSecureParking:    kind.catType != locationcategory.CategoryCustomerLocation,
			AllowsOvernight:     kind.catType == locationcategory.CategoryTruckStop,
			HasRestroom:         true,
		}
		byName[key] = category.ID
		created = append(created, category)
		stamps = append(stamps, g.mdStamp(category.ID, at))
	}
	if len(created) == 0 {
		return byName, nil
	}
	if err := insertRows(ctx, tx, locationcategorydb.LocationCategories.Handle, created); err != nil {
		return nil, fmt.Errorf("insert location categories: %w", err)
	}
	if err := backdate(ctx, tx, locationcategorydb.LocationCategories.Handle, stamps); err != nil {
		return nil, err
	}
	g.report.add("Location categories", len(created))

	return byName, nil
}

func (g *generator) mdLocations(ctx context.Context, tx *keel.DB) error {
	categories, err := g.mdLocationCategories(ctx, tx)
	if err != nil {
		return err
	}
	taken, err := mdTaken(ctx, tx, locationdb.Locations.Handle, locationdb.Locations.Code)
	if err != nil {
		return err
	}

	weights := make([]int, len(mdLocationKinds))
	for i, kind := range mdLocationKinds {
		weights[i] = kind.weight
	}

	ownerWeights := make([]int, len(g.customers))
	for i, c := range g.customers {
		ownerWeights[i] = c.Weight
	}

	n := g.scale.locations
	created, ids := g.mdTimes(n, "loc_")
	locations := make([]*location.Location, 0, n)
	stamps := make([]stamp, 0, n)
	idStrings := make([]string, 0, n)
	g.locations = make([]*locationRecord, 0, n)
	radius := geofence.DefaultRadiusMeters

	for i := range n {
		kind := &mdLocationKinds[g.fake.weighted(weights)]
		owner, city := g.mdLocationOwner(kind, ownerWeights)
		ownerName := g.fake.companyName()
		if owner != nil {
			ownerName = owner.Name
		}
		lat, lon := g.fake.jitter(city)
		name := kind.name(g, city, ownerName)
		code := uniqueCode(codeFrom(name, 4)+"-"+codeFrom(city.Name, 3), taken)

		entity := &location.Location{
			ID:                   ids[i],
			BusinessUnitID:       g.refs.buID,
			OrganizationID:       g.refs.orgID,
			LocationCategoryID:   categories[strings.ToLower(kind.category)],
			StateID:              g.refs.stateIDs[city.State],
			Status:               domaintypes.StatusActive,
			Code:                 code,
			Name:                 name,
			AddressLine1:         g.fake.street(),
			City:                 city.Name,
			PostalCode:           g.fake.postalCode(city),
			Timezone:             city.Timezone,
			IsGeocoded:           true,
			Latitude:             &lat,
			Longitude:            &lon,
			GeofenceType:         geofence.TypeAuto,
			GeofenceRadiusMeters: &radius,
		}
		if g.fake.chance(0.03) {
			entity.Status = domaintypes.StatusInactive
		}
		if g.fake.chance(0.2) {
			entity.Description = pick(g.fake, []string{
				"Check in at the guard shack; dock doors on the north side.",
				"Appointments required. Arrive 30 minutes early with seal intact.",
				"Live unload only. No overnight parking on site.",
				"Drop trailers in the south lot; pick up empties from row C.",
				"Hazmat placards must be visible at gate check.",
			})
		}

		locations = append(locations, entity)
		stamps = append(stamps, g.mdStamp(entity.ID, created[i]))
		idStrings = append(idStrings, entity.ID.String())
		if entity.Status == domaintypes.StatusActive {
			rec := &locationRecord{
				ID:        entity.ID,
				Code:      entity.Code,
				Name:      entity.Name,
				City:      city,
				Address:   entity.AddressLine1,
				Shipper:   kind.shipper,
				Consignee: kind.consignee,
			}
			g.locations = append(g.locations, rec)
			if owner != nil {
				owner.Facilities = append(owner.Facilities, rec)
			}
		}
	}

	if err = insertRows(ctx, tx, locationdb.Locations.Handle, locations); err != nil {
		return fmt.Errorf("insert locations: %w", err)
	}
	l := locationdb.Locations
	locationIDs := make([]pulid.ID, 0, len(locations))
	for _, loc := range locations {
		locationIDs = append(locationIDs, loc.ID)
	}
	if _, err = l.Where(l.ID.In(locationIDs...)).Update(ctx, tx, l.GeofenceGeometry.ToExpr(
		postgis.Buffer(postgis.Pt(l.Longitude, l.Latitude), l.GeofenceRadiusMeters),
	)); err != nil {
		return fmt.Errorf("build location geofences: %w", err)
	}
	if err = backdate(ctx, tx, locationdb.Locations.Handle, stamps); err != nil {
		return err
	}
	g.report.add("Locations", len(locations))

	return nil
}

type mdCommodityDef struct {
	name        string
	description string
	class       commodity.FreightClass
	weight      float64
	tempMin     *int
	tempMax     *int
	hazmat      bool
	stackable   bool
	fragile     bool
	instruction string
}

func mdTemp(low, high int) (lowPtr, highPtr *int) {
	return &low, &high
}

func mdReefer(name, description string, low, high int, weight float64) mdCommodityDef {
	minTemp, maxTemp := mdTemp(low, high)
	return mdCommodityDef{
		name: name, description: description, class: commodity.FreightClass65,
		weight: weight, tempMin: minTemp, tempMax: maxTemp, stackable: true,
		instruction: "Pre-cool trailer and run continuous; download reefer report at delivery.",
	}
}

func mdDry(
	name, description string,
	class commodity.FreightClass,
	weight float64,
	stackable, fragile bool,
) mdCommodityDef {
	return mdCommodityDef{
		name: name, description: description, class: class, weight: weight,
		stackable: stackable, fragile: fragile,
	}
}

var mdCommodityCatalog = []mdCommodityDef{
	mdDry(
		"Canned Vegetables",
		"Palletized canned vegetables, shrink-wrapped",
		commodity.FreightClass55,
		2100,
		true,
		false,
	),
	mdDry(
		"Bottled Water",
		"Cased bottled water on 48x40 pallets",
		commodity.FreightClass55,
		2300,
		false,
		false,
	),
	mdDry(
		"Soft Drinks",
		"Cased carbonated beverages",
		commodity.FreightClass60,
		2200,
		false,
		false,
	),
	mdDry(
		"Breakfast Cereal",
		"Cased cereal, light and cube-heavy",
		commodity.FreightClass150,
		650,
		true,
		false,
	),
	mdDry(
		"Snack Foods",
		"Chips and snacks in corrugated cases",
		commodity.FreightClass175,
		450,
		true,
		true,
	),
	mdDry(
		"Paper Towels",
		"Household paper products, cube-out freight",
		commodity.FreightClass150,
		500,
		true,
		false,
	),
	mdDry(
		"Corrugated Boxes",
		"Flattened corrugated cartons on pallets",
		commodity.FreightClass70,
		1400,
		true,
		false,
	),
	mdDry(
		"Plastic Resin",
		"Polyethylene resin pellets in gaylord boxes",
		commodity.FreightClass55,
		1650,
		false,
		false,
	),
	mdDry(
		"Household Appliances",
		"Boxed major appliances",
		commodity.FreightClass125,
		320,
		false,
		true,
	),
	mdDry(
		"Flat-Pack Furniture",
		"Ready-to-assemble furniture cartons",
		commodity.FreightClass125,
		700,
		true,
		false,
	),
	mdDry("Mattresses", "Compressed boxed mattresses", commodity.FreightClass200, 380, true, false),
	mdDry(
		"Consumer Electronics",
		"Boxed televisions and small electronics",
		commodity.FreightClass92_5,
		600,
		false,
		true,
	),
	mdDry(
		"Auto Parts",
		"Mixed automotive aftermarket parts",
		commodity.FreightClass85,
		1200,
		true,
		false,
	),
	mdDry("Tires", "Floor-loaded passenger tires", commodity.FreightClass110, 25, true, false),
	mdDry(
		"Pet Food",
		"Bagged dry pet food on pallets",
		commodity.FreightClass60,
		2000,
		true,
		false,
	),
	mdDry(
		"Bagged Cement",
		"Portland cement in 94 lb bags",
		commodity.FreightClass50,
		3000,
		false,
		false,
	),
	mdDry(
		"Building Insulation",
		"Compressed fiberglass insulation rolls",
		commodity.FreightClass250,
		300,
		true,
		false,
	),
	mdDry(
		"Ceramic Tile",
		"Boxed floor tile, heavy and dense",
		commodity.FreightClass60,
		2500,
		false,
		true,
	),
	mdDry(
		"Glass Containers",
		"Empty glass bottles and jars on tiers",
		commodity.FreightClass85,
		1800,
		false,
		true,
	),
	mdDry(
		"Aluminum Cans",
		"Empty beverage cans on tier sheets",
		commodity.FreightClass250,
		400,
		true,
		false,
	),
	mdDry(
		"Cotton Apparel",
		"Boxed apparel and textiles",
		commodity.FreightClass100,
		500,
		true,
		false,
	),
	mdDry(
		"Medical Supplies",
		"Boxed disposable medical supplies",
		commodity.FreightClass100,
		450,
		true,
		true,
	),
	mdDry(
		"Pharmaceuticals",
		"Packaged over-the-counter pharmaceuticals",
		commodity.FreightClass92_5,
		550,
		true,
		true,
	),
	mdDry(
		"Sporting Goods",
		"Mixed boxed sporting goods",
		commodity.FreightClass125,
		400,
		true,
		false,
	),
	mdDry(
		"Hardware",
		"Fasteners, tools and hardware on pallets",
		commodity.FreightClass70,
		1500,
		true,
		false,
	),
	mdDry(
		"Lawn and Garden",
		"Bagged soil, mulch and fertilizer",
		commodity.FreightClass55,
		2400,
		false,
		false,
	),
	mdDry(
		"Solar Panels",
		"Crated photovoltaic modules",
		commodity.FreightClass125,
		1100,
		false,
		true,
	),
	mdDry(
		"Lithium-Free Batteries",
		"Alkaline consumer batteries",
		commodity.FreightClass65,
		1500,
		true,
		false,
	),
	mdDry(
		"Office Paper",
		"Cased copy paper on pallets",
		commodity.FreightClass55,
		2000,
		true,
		false,
	),
	mdDry("Beer", "Cased and kegged beer", commodity.FreightClass60, 2200, false, false),
	mdDry(
		"Wine and Spirits",
		"Cased wine and spirits",
		commodity.FreightClass65,
		1600,
		false,
		true,
	),
	mdDry("Grain Sacks", "Bagged grain and flour", commodity.FreightClass50, 2600, true, false),
	mdReefer("Fresh Produce", "Mixed fresh fruits and vegetables", 34, 38, 1500),
	mdReefer("Frozen Foods", "Frozen prepared meals and vegetables", -10, 0, 1800),
	mdReefer("Ice Cream", "Frozen dairy desserts, deep frozen", -20, -10, 1400),
	mdReefer("Fresh Meat", "Boxed fresh beef and pork", 28, 34, 2000),
	mdReefer("Poultry", "Fresh boxed poultry", 28, 32, 2000),
	mdReefer("Dairy Products", "Milk, cheese and yogurt", 33, 38, 1900),
	mdReefer("Fresh Seafood", "Iced fresh seafood", 30, 34, 1500),
	mdReefer("Fresh Flowers", "Cut flowers in buckets", 34, 36, 400),
	mdReefer("Chocolate and Candy", "Temperature-sensitive confectionery", 55, 65, 900),
	mdReefer("Vaccines and Biologics", "Cold-chain pharmaceuticals", 36, 46, 300),
	mdDry(
		"Steel Coils",
		"Hot rolled steel coils, chained and tarped",
		commodity.FreightClass50,
		40000,
		false,
		false,
	),
	mdDry(
		"Structural Steel",
		"Steel beams and I-beams",
		commodity.FreightClass50,
		4000,
		false,
		false,
	),
	mdDry(
		"Dimensional Lumber",
		"Banded softwood lumber units",
		commodity.FreightClass55,
		2500,
		true,
		false,
	),
	mdDry("Plywood", "Banded plywood and OSB sheets", commodity.FreightClass55, 2800, true, false),
	mdDry(
		"Concrete Pipe",
		"Precast concrete pipe sections",
		commodity.FreightClass50,
		6000,
		false,
		false,
	),
	mdDry("Machinery", "Crated industrial machinery", commodity.FreightClass85, 5000, false, true),
	{
		name: "Paint and Coatings", description: "Flammable paint in pails and drums",
		class: commodity.FreightClass60, weight: 1800, hazmat: true,
		instruction: "Placard as required; keep away from oxidizers.",
	},
	{
		name: "Industrial Cleaners", description: "Corrosive cleaning chemicals in totes",
		class: commodity.FreightClass60, weight: 2400, hazmat: true,
		instruction: "Corrosive; segregate from food products.",
	},
	{
		name: "Lithium Ion Batteries", description: "Packaged lithium ion cells and packs",
		class: commodity.FreightClass85, weight: 900, hazmat: true,
		instruction: "Class 9; protect from damage and short circuit.",
	},
	{
		name: "Propane Cylinders", description: "Filled propane exchange cylinders in cages",
		class: commodity.FreightClass77_5, weight: 1200, hazmat: true,
		instruction: "Flammable gas; secure cages upright.",
	},
}

func (g *generator) mdCommodities(ctx context.Context, tx *keel.DB) error {
	taken, err := mdTaken(ctx, tx, commoditydb.Commodities.Handle, commoditydb.Commodities.Name)
	if err != nil {
		return err
	}
	hm := hazardousmaterialdb.HazardousMaterials
	hazmats, err := hm.Pick(hm.ID).
		Where(hm.Status.Eq(domaintypes.StatusActive)).
		Order(hm.ID.Asc()).
		All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load hazardous materials: %w", err)
	}
	hazmatIDs := make([]pulid.ID, 0, len(hazmats))
	for _, h := range hazmats {
		hazmatIDs = append(hazmatIDs, h.ID)
	}

	rows := make([]*commodity.Commodity, 0, len(mdCommodityCatalog))
	stamps := make([]stamp, 0, len(mdCommodityCatalog))
	for _, def := range mdCommodityCatalog {
		if def.hazmat && len(hazmatIDs) == 0 {
			continue
		}
		if _, ok := taken[strings.ToLower(def.name)]; ok {
			continue
		}
		at := g.mdCreated()
		weight := def.weight
		row := &commodity.Commodity{
			ID:                  idAt("com_", at),
			BusinessUnitID:      g.refs.buID,
			OrganizationID:      g.refs.orgID,
			Status:              domaintypes.StatusActive,
			Name:                def.name,
			Description:         def.description,
			MinTemperature:      def.tempMin,
			MaxTemperature:      def.tempMax,
			WeightPerUnit:       &weight,
			FreightClass:        def.class,
			LoadingInstructions: def.instruction,
			Stackable:           def.stackable,
			Fragile:             def.fragile,
		}
		if def.hazmat {
			row.HazardousMaterialID = pick(g.fake, hazmatIDs)
		}
		rows = append(rows, row)
		stamps = append(stamps, g.mdStamp(row.ID, at))
	}
	if len(rows) > 0 {
		if err = insertRows(ctx, tx, commoditydb.Commodities.Handle, rows); err != nil {
			return fmt.Errorf("insert commodities: %w", err)
		}
		if err = backdate(ctx, tx, commoditydb.Commodities.Handle, stamps); err != nil {
			return err
		}
	}
	g.report.add("Commodities", len(rows))

	cm := commoditydb.Commodities
	all, err := cm.Where(cm.Status.Eq(domaintypes.StatusActive)).Order(cm.ID.Asc()).All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load commodities: %w", err)
	}
	g.commodities = make([]*commodityRecord, 0, len(all))
	for _, c := range all {
		record := &commodityRecord{
			ID:             c.ID,
			Name:           c.Name,
			WeightPerPiece: 40,
			Hazmat:         c.HazardousMaterialID.IsNotNil(),
		}
		if c.WeightPerUnit != nil && *c.WeightPerUnit > 0 {
			record.WeightPerPiece = int64(math.Round(*c.WeightPerUnit))
		}
		if c.MinTemperature != nil {
			record.TempMin = ptr(int16(*c.MinTemperature))
		}
		if c.MaxTemperature != nil {
			record.TempMax = ptr(int16(*c.MaxTemperature))
		}
		g.commodities = append(g.commodities, record)
	}

	return nil
}

var mdTractorModels = map[string][]string{
	"Freightliner":  {"Cascadia", "Cascadia 126", "Columbia", "Coronado"},
	"Peterbilt":     {"579", "389", "567"},
	"Kenworth":      {"T680", "T880", "W990"},
	"Volvo":         {"VNL 760", "VNL 860", "VNR 640"},
	"International": {"LT625", "RH613", "LoneStar"},
	"Mack":          {"Anthem", "Pinnacle", "Granite"},
	"Western Star":  {"57X", "49X"},
}

var mdTrailerModels = map[string][]string{
	"Great Dane":        {"Champion CL", "Everest SS", "Freedom LT"},
	"Utility":           {"4000D-X", "3000R", "4000AE"},
	"Wabash":            {"DuraPlate", "ArcticLite", "Transcission"},
	"Hyundai Translead": {"HT Composite", "ThermoTech"},
	"Stoughton":         {"Composite Plate", "Z-Plate"},
	"Vanguard":          {"VXP", "CoolGlobe"},
	"Fontaine":          {"Revolution", "Infinity"},
}

func mdMakes(models map[string][]string) []string {
	makes := make([]string, 0, len(models))
	for name := range models {
		makes = append(makes, name)
	}
	slices.Sort(makes)
	return makes
}

var mdFleetCatalog = []struct {
	code        string
	description string
}{
	{"OTR-E", "Over-the-road East"},
	{"OTR-W", "Over-the-road West"},
	{"REG-SE", "Southeast Regional"},
	{"REG-MW", "Midwest Regional"},
	{"REG-SW", "Southwest Regional"},
	{"REG-NE", "Northeast Regional"},
	{"REG-PNW", "Pacific Northwest Regional"},
	{"LOC-DFW", "Dallas-Fort Worth Local"},
	{"LOC-CHI", "Chicagoland Local"},
	{"LOC-ATL", "Atlanta Local"},
	{"LOC-LA", "Southern California Local"},
	{"DED-RTL", "Dedicated Retail"},
	{"DED-GRC", "Dedicated Grocery"},
	{"DED-AUTO", "Dedicated Automotive"},
	{"REEFER", "Temperature-Controlled"},
	{"FLATBED", "Flatbed and Open Deck"},
	{"HAZMAT", "Hazmat Certified"},
	{"TEAM", "Expedited Teams"},
	{"INTERMOD", "Intermodal Drayage"},
	{"OWNOP", "Owner-Operators"},
}

var mdFleetColors = []string{
	"#2563EB", "#16A34A", "#DC2626", "#9333EA", "#EA580C", "#0891B2", "#CA8A04", "#DB2777",
	"#4F46E5", "#059669", "#B91C1C", "#7C3AED", "#C2410C", "#0E7490", "#A16207", "#BE185D",
}

func (g *generator) mdEquipment(ctx context.Context, tx *keel.DB) (*mdEquipment, error) {
	equipment := &mdEquipment{
		manufacturers: make(map[string]pulid.ID),
		kinds:         make(map[pulid.ID]trailerKind),
	}

	et := equipmenttypedb.EquipmentTypes
	types, err := et.Where(et.Status.Eq(domaintypes.StatusActive)).Order(et.ID.Asc()).All(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("load equipment types: %w", err)
	}
	for _, t := range types {
		switch t.Class {
		case equipmenttype.ClassTractor:
			equipment.tractorTypes = append(equipment.tractorTypes, t)
		case equipmenttype.ClassTrailer:
			equipment.trailerTypes = append(equipment.trailerTypes, t)
			equipment.kinds[t.ID] = trailerKindOf(t)
		}
	}
	if len(equipment.tractorTypes) == 0 || len(equipment.trailerTypes) == 0 {
		return nil, fmt.Errorf(
			"%w: no active tractor or trailer equipment types",
			ErrMissingSeedData,
		)
	}

	if err := g.mdManufacturers(ctx, tx, equipment); err != nil {
		return nil, err
	}

	fleetCodes, err := g.mdFleetCodes(ctx, tx)
	if err != nil {
		return nil, err
	}
	equipment.fleetCodes = fleetCodes

	return equipment, nil
}

func (g *generator) mdManufacturers(
	ctx context.Context,
	tx *keel.DB,
	equipment *mdEquipment,
) error {
	existing, err := equipmentmanufacturerdb.EquipmentManufacturers.Query().All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load equipment manufacturers: %w", err)
	}
	for _, m := range existing {
		equipment.manufacturers[strings.ToLower(m.Name)] = m.ID
	}

	names := append(mdMakes(mdTractorModels), mdMakes(mdTrailerModels)...)
	rows := make([]*equipmentmanufacturer.EquipmentManufacturer, 0, len(names))
	stamps := make([]stamp, 0, len(names))
	for _, name := range names {
		key := strings.ToLower(name)
		if _, ok := equipment.manufacturers[key]; ok {
			continue
		}
		at := g.mdCreated()
		row := &equipmentmanufacturer.EquipmentManufacturer{
			ID:             idAt("em_", at),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			Status:         domaintypes.StatusActive,
			Name:           name,
			Description:    name + " equipment",
		}
		equipment.manufacturers[key] = row.ID
		rows = append(rows, row)
		stamps = append(stamps, g.mdStamp(row.ID, at))
	}
	if len(rows) == 0 {
		return nil
	}
	if err := insertRows(ctx, tx, equipmentmanufacturerdb.EquipmentManufacturers.Handle, rows); err != nil {
		return fmt.Errorf("insert equipment manufacturers: %w", err)
	}
	if err := backdate(ctx, tx, equipmentmanufacturerdb.EquipmentManufacturers.Handle, stamps); err != nil {
		return err
	}
	g.report.add("Equipment manufacturers", len(rows))

	return nil
}

func (g *generator) mdFleetCodes(ctx context.Context, tx *keel.DB) ([]pulid.ID, error) {
	taken, err := mdTaken(ctx, tx, fleetcodedb.FleetCodes.Handle, fleetcodedb.FleetCodes.Code)
	if err != nil {
		return nil, err
	}

	n := g.scale.fleetCodes
	rows := make([]*fleetcode.FleetCode, 0, n)
	stamps := make([]stamp, 0, n)
	ids := make([]pulid.ID, 0, n)
	for i := range n {
		def := mdFleetCatalog[i%len(mdFleetCatalog)]
		at := g.mdCreated()
		revenue := float64(g.fake.between(180, 420)) * 1_000
		deadhead := float64(g.fake.between(6, 15))
		mileage := float64(g.fake.between(80, 140)) * 1_000
		row := &fleetcode.FleetCode{
			ID:             idAt("fc_", at),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			ManagerID:      pick(g.fake, g.refs.userIDs),
			Status:         domaintypes.StatusActive,
			Code:           uniqueCode(def.code, taken),
			Description:    def.description,
			RevenueGoal:    &revenue,
			DeadheadGoal:   &deadhead,
			MileageGoal:    &mileage,
			Color:          mdFleetColors[i%len(mdFleetColors)],
		}
		rows = append(rows, row)
		ids = append(ids, row.ID)
		stamps = append(stamps, g.mdStamp(row.ID, at))
	}
	if err = insertRows(ctx, tx, fleetcodedb.FleetCodes.Handle, rows); err != nil {
		return nil, fmt.Errorf("insert fleet codes: %w", err)
	}
	if err = backdate(ctx, tx, fleetcodedb.FleetCodes.Handle, stamps); err != nil {
		return nil, err
	}
	g.report.add("Fleet codes", len(rows))

	return ids, nil
}

var (
	mdDriverTypes = []worker.DriverType{
		worker.DriverTypeOTR,
		worker.DriverTypeRegional,
		worker.DriverTypeLocal,
		worker.DriverTypeTeam,
	}
	mdDriverTypeWeights = []int{45, 30, 20, 5}
	mdEndorsements      = []worker.EndorsementType{
		worker.EndorsementTypeNone,
		worker.EndorsementTypeHazmat,
		worker.EndorsementTypeTanker,
		worker.EndorsementTypeTankerHazmat,
		worker.EndorsementTypeDoubleTriple,
		worker.EndorsementTypePassenger,
	}
	mdEndorsementWeights = []int{50, 12, 10, 10, 15, 3}
	mdTrainingHealth     = []worker.TrainingHealth{
		worker.TrainingHealthCurrent,
		worker.TrainingHealthDueSoon,
		worker.TrainingHealthScheduled,
		worker.TrainingHealthOverdue,
		worker.TrainingHealthExpiringSoon,
	}
	mdTrainingWeights = []int{75, 10, 5, 5, 5}
	mdPTOTypes        = []worker.PTOType{
		worker.PTOTypeVacation,
		worker.PTOTypeSick,
		worker.PTOTypePersonal,
		worker.PTOTypeHoliday,
		worker.PTOTypeBereavement,
	}
	mdPTOReasons = []string{
		"Family vacation", "Not feeling well", "Personal appointment", "Home time",
		"Family emergency", "Doctor visit", "Wedding attendance", "Moving day", "Jury duty",
	}
	mdExaminers = []string{
		"Concentra Medical Centers", "US HealthWorks", "Dr. Alan Pierce, DO",
		"Dr. Maria Kessler, MD", "WorkCare Occupational Health", "Dr. Henry Osei, PA-C",
	}
)

func (g *generator) mdWorkers(
	ctx context.Context,
	tx *keel.DB,
	fleetCodes []pulid.ID,
) ([]*mdDriver, error) {
	n := g.scale.drivers
	now := unix(g.now)
	created, ids := g.mdTimes(n, "wrk_")
	drivers := make([]*mdDriver, 0, n)
	workers := make([]*worker.Worker, 0, n)
	profiles := make([]*worker.WorkerProfile, 0, n)
	ptos := make([]*worker.WorkerPTO, 0, n*2)
	workerStamps := make([]stamp, 0, n)
	profileStamps := make([]stamp, 0, n)
	ptoStamps := make([]stamp, 0, n*2)
	usedEmails := make(map[string]struct{}, n)

	for i := range n {
		home := g.fake.city()
		gender := worker.GenderMale
		if g.fake.chance(0.14) {
			gender = worker.GenderFemale
		}
		first, last := g.fake.firstName(), g.fake.lastName()
		active := g.fake.chance(0.9)
		workerType := worker.WorkerTypeEmployee
		if g.fake.chance(0.2) {
			workerType = worker.WorkerTypeContractor
		}
		driverType := mdDriverTypes[g.fake.weighted(mdDriverTypeWeights)]

		email := uniqueCode(strings.ToLower(first+"."+last), usedEmails) + "@drivers.example.com"
		entity := &worker.Worker{
			ID:                    ids[i],
			BusinessUnitID:        g.refs.buID,
			OrganizationID:        g.refs.orgID,
			StateID:               g.refs.stateIDs[home.State],
			FleetCodeID:           pick(g.fake, fleetCodes),
			Status:                domaintypes.StatusActive,
			Type:                  workerType,
			DriverType:            driverType,
			FirstName:             first,
			LastName:              last,
			AddressLine1:          g.fake.street(),
			City:                  home.Name,
			PostalCode:            g.fake.postalCode(home),
			Email:                 email,
			PhoneNumber:           g.fake.phone(),
			EmergencyContactName:  g.fake.firstName() + " " + last,
			EmergencyContactPhone: g.fake.phone(),
			Gender:                gender,
			CanBeAssigned:         active,
			AvailableForDispatch:  active,
		}
		if g.fake.chance(0.5) {
			entity.ManagerID = pick(g.fake, g.refs.userIDs)
		}
		if g.fake.chance(0.3) {
			entity.ExternalID = "PAY-" + g.fake.digits(6)
		}
		if !active {
			entity.Status = domaintypes.StatusInactive
		}

		profile := g.mdProfile(entity, home, created[i], now, active)
		if !active {
			entity.AssignmentBlocked = "Separated"
			if profile.TerminationDate == nil {
				entity.AssignmentBlocked = "On leave"
			}
		} else if !profile.IsQualified {
			entity.AssignmentBlocked = profile.DisqualificationReason
			entity.CanBeAssigned = false
		}

		workers = append(workers, entity)
		profiles = append(profiles, profile)
		workerStamps = append(workerStamps, g.mdStamp(entity.ID, created[i]))
		profileStamps = append(profileStamps, g.mdStamp(profile.ID, created[i]))
		drivers = append(drivers, &mdDriver{worker: entity, profile: profile, home: home})

		for range g.fake.between(0, 3) {
			pto := g.mdPTO(entity.ID, now)
			ptos = append(ptos, pto)
			ptoStamps = append(ptoStamps, g.mdStamp(pto.ID, min(pto.StartDate-mdDay*14, now)))
		}
	}

	if err := insertRows(ctx, tx, workerdb.Workers.Handle, workers); err != nil {
		return nil, fmt.Errorf("insert workers: %w", err)
	}
	if err := insertRows(ctx, tx, workerdb.WorkerProfiles.Handle, profiles); err != nil {
		return nil, fmt.Errorf("insert worker profiles: %w", err)
	}
	if err := insertRows(ctx, tx, workerdb.WorkerPTOs.Handle, ptos); err != nil {
		return nil, fmt.Errorf("insert worker pto: %w", err)
	}
	for _, b := range []struct {
		table  string
		stamps []stamp
	}{
		{table: "workers", stamps: workerStamps},
		{table: "worker_profiles", stamps: profileStamps},
		{table: "worker_pto", stamps: ptoStamps},
	} {
		if err := backdateTable(ctx, tx, b.table, b.stamps); err != nil {
			return nil, err
		}
	}
	g.report.add("Workers", len(workers))
	g.report.add("Worker profiles", len(profiles))
	g.report.add("Worker PTO", len(ptos))

	return drivers, nil
}

func (g *generator) mdProfile(
	entity *worker.Worker,
	home *cityRecord,
	created, now int64,
	active bool,
) *worker.WorkerProfile {
	year := 365 * mdDay
	endorsement := mdEndorsements[g.fake.weighted(mdEndorsementWeights)]
	cdl := worker.CDLClassA
	switch roll := g.fake.intn(100); {
	case roll < 2:
		cdl = worker.CDLClassC
	case roll < 10:
		cdl = worker.CDLClassB
	}

	licenseExpiry := now + g.fake.between64(30*mdDay, 5*year)
	medicalExpiry := now + g.fake.between64(30*mdDay, 2*year)
	switch roll := g.fake.intn(100); {
	case roll < 3:
		licenseExpiry = now - g.fake.between64(mdDay, 60*mdDay)
	case roll < 6:
		medicalExpiry = now - g.fake.between64(mdDay, 45*mdDay)
	case roll < 12:
		medicalExpiry = now + g.fake.between64(mdDay, 29*mdDay)
	}

	profile := &worker.WorkerProfile{
		ID:                  idAt("wrkp_", created),
		WorkerID:            entity.ID,
		BusinessUnitID:      g.refs.buID,
		OrganizationID:      g.refs.orgID,
		LicenseStateID:      g.refs.stateIDs[home.State],
		DOB:                 now - g.fake.between64(23*year, 64*year),
		LicenseNumber:       home.State + g.fake.digits(8),
		CDLClass:            cdl,
		Endorsement:         endorsement,
		LicenseExpiry:       licenseExpiry,
		MedicalCardExpiry:   &medicalExpiry,
		MedicalExaminerName: pick(g.fake, mdExaminers),
		MedicalExaminerNPI:  "1" + g.fake.digits(9),
		HireDate:            created - g.fake.between64(0, 12*year),
		PhysicalDueDate:     ptr(now + g.fake.between64(mdDay, year)),
		MVRDueDate:          ptr(now + g.fake.between64(mdDay, year)),
		TrainingHealth:      mdTrainingHealth[g.fake.weighted(mdTrainingWeights)],
		NextTrainingDue:     ptr(now + g.fake.between64(mdDay, 180*mdDay)),
		LastComplianceCheck: now - g.fake.between64(0, 30*mdDay),
		LastMVRCheck:        now - g.fake.between64(30*mdDay, year),
		LastDrugTest:        now - g.fake.between64(14*mdDay, year),
		DrugAlcoholStatus:   worker.DrugAlcoholClear,
		ReturnToDutyStatus:  worker.ReturnToDutyNotRequired,
		ShortHaulExempt:     entity.DriverType == worker.DriverTypeLocal && g.fake.chance(0.5),
		ComplianceStatus:    worker.ComplianceStatusPending,
		SafetyRating:        worker.SafetyRatingExcellent,
		SafetyScore:         100,
	}
	queried := now - g.fake.between64(mdDay, 300*mdDay)
	profile.LastClearinghouseQueryAt = &queried
	profile.NextClearinghouseQueryDue = ptr(queried + year)

	if endorsement.RequiresHazmatExpiry() {
		profile.HazmatExpiry = ptr(now + g.fake.between64(60*mdDay, 5*year))
	}
	if g.fake.chance(0.15) {
		profile.TWICCardNumber = g.fake.digits(12)
		profile.TWICExpiry = ptr(now + g.fake.between64(90*mdDay, 5*year))
	}
	if g.fake.chance(0.05) {
		profile.CDLRestrictions = pick(
			g.fake,
			[]string{"L - No air brakes", "E - No manual transmission", "K - Intrastate only"},
		)
	}

	switch roll := g.fake.intn(100); {
	case roll < 3:
		profile.DrugAlcoholStatus = worker.DrugAlcoholPending
	case roll < 6:
		profile.DrugAlcoholStatus = worker.DrugAlcoholUnknown
	}

	switch roll := g.fake.intn(100); {
	case roll < 55:
		profile.SafetyRating, profile.SafetyScore = worker.SafetyRatingExcellent, int16(
			g.fake.between(90, 100),
		)
	case roll < 85:
		profile.SafetyRating, profile.SafetyScore = worker.SafetyRatingGood, int16(
			g.fake.between(80, 89),
		)
	case roll < 95:
		profile.SafetyRating, profile.SafetyScore = worker.SafetyRatingWatch, int16(
			g.fake.between(65, 79),
		)
	default:
		profile.SafetyRating, profile.SafetyScore = worker.SafetyRatingAtRisk, int16(
			g.fake.between(40, 64),
		)
	}

	next := min(licenseExpiry, medicalExpiry)
	if profile.HazmatExpiry != nil {
		next = min(next, *profile.HazmatExpiry)
	}
	profile.NextCredentialExpiry = &next

	switch {
	case licenseExpiry < now:
		profile.ComplianceStatus = worker.ComplianceStatusNonCompliant
		profile.DisqualificationReason = "CDL expired"
	case medicalExpiry < now:
		profile.ComplianceStatus = worker.ComplianceStatusNonCompliant
		profile.DisqualificationReason = "Medical certificate expired"
	case profile.DrugAlcoholStatus != worker.DrugAlcoholClear:
		profile.ComplianceStatus = worker.ComplianceStatusPending
	default:
		profile.ComplianceStatus = worker.ComplianceStatusCompliant
	}
	profile.IsQualified = profile.ComplianceStatus == worker.ComplianceStatusCompliant

	if !active && g.fake.chance(0.7) {
		profile.TerminationDate = ptr(now - g.fake.between64(mdDay, 300*mdDay))
	}

	return profile
}

func (g *generator) mdPTO(workerID pulid.ID, now int64) *worker.WorkerPTO {
	start := now + g.fake.between64(-150, 90)*mdDay
	end := start + g.fake.between64(1, 6)*mdDay
	pto := &worker.WorkerPTO{
		ID:             idAt("wrkpto_", min(start, now)),
		WorkerID:       workerID,
		BusinessUnitID: g.refs.buID,
		OrganizationID: g.refs.orgID,
		Type:           pick(g.fake, mdPTOTypes),
		StartDate:      start,
		EndDate:        end,
		Days:           pto.ComputePTODays(start, end, nil, true, nil),
		Reason:         pick(g.fake, mdPTOReasons),
		Status:         worker.PTOStatusRequested,
	}

	roll := g.fake.intn(100)
	switch {
	case start < now && roll < 85, start >= now && roll < 45:
		pto.Status = worker.PTOStatusApproved
		pto.ApproverID = g.refs.adminID
	case start >= now && roll < 85:
		pto.Status = worker.PTOStatusRequested
	case roll < 93:
		pto.Status = worker.PTOStatusRejected
		pto.RejectorID = g.refs.adminID
		pto.RejectionReason = "Insufficient coverage for the requested dates"
	default:
		pto.Status = worker.PTOStatusCancelled
		pto.CancelledByID = workerID
		pto.CancellationReason = "Plans changed"
	}

	return pto
}

var (
	mdLessors = []string{
		"Penske Truck Leasing",
		"Ryder System",
		"Element Fleet",
		"Wheels Inc",
	}
	mdTrailerLessors = []string{
		"XTRA Lease",
		"Premier Trailer Leasing",
		"Star Leasing",
		"Wells Fargo Equipment Finance",
	}
)

func (g *generator) mdTrailerTypeWeights(types []*equipmenttype.EquipmentType) []int {
	weights := make([]int, len(types))
	for i, t := range types {
		switch trailerKindOf(t) {
		case trailerKindDry:
			weights[i] = 16
		case trailerKindReefer:
			weights[i] = 6
		case trailerKindFlatbed:
			weights[i] = 3
		case trailerKindOther:
			weights[i] = 1
		}
	}
	return weights
}

func (g *generator) mdFleet(
	ctx context.Context,
	tx *keel.DB,
	equipment *mdEquipment,
	drivers []*mdDriver,
) error {
	tractorTaken, err := mdTaken(ctx, tx, tractordb.Tractors.Handle, tractordb.Tractors.Code)
	if err != nil {
		return err
	}
	trailerTaken, err := mdTaken(ctx, tx, trailerdb.Trailers.Handle, trailerdb.Trailers.Code)
	if err != nil {
		return err
	}

	ordered := make([]*mdDriver, 0, len(drivers))
	var inactive []*mdDriver
	for _, d := range drivers {
		if d.worker.Status == domaintypes.StatusActive && d.worker.CanBeAssigned {
			ordered = append(ordered, d)
		} else {
			inactive = append(inactive, d)
		}
	}
	reserve := len(ordered) / 10
	teammates := ordered[len(ordered)-reserve:]
	ordered = append(ordered[:len(ordered)-reserve:len(ordered)-reserve], inactive...)

	trailers, trailerStamps := g.mdTrailers(equipment, trailerTaken)
	available := make([]*trailer.Trailer, 0, len(trailers))
	for _, t := range trailers {
		if t.Status == domaintypes.EquipmentStatusAvailable {
			available = append(available, t)
		}
	}

	now := unix(g.now)
	n := min(g.scale.tractors, len(ordered))
	tractorMakes := mdMakes(mdTractorModels)
	tractors := make([]*tractor.Tractor, 0, n)
	tractorStamps := make([]stamp, 0, n)
	g.crews = make([]*crew, 0, n)
	nextTeammate := 0
	nextTrailer := 0

	for i := range n {
		d := ordered[i]
		at := max(g.mdCreated(), d.profile.HireDate)
		at = min(at, unix(g.start))
		makeName := pick(g.fake, tractorMakes)
		year := g.fake.between(2015, g.now.Year())
		entity := &tractor.Tractor{
			ID:                      idAt("trac_", at),
			BusinessUnitID:          g.refs.buID,
			OrganizationID:          g.refs.orgID,
			PrimaryWorkerID:         d.worker.ID,
			EquipmentTypeID:         pick(g.fake, equipment.tractorTypes).ID,
			EquipmentManufacturerID: equipment.manufacturers[strings.ToLower(makeName)],
			StateID:                 g.refs.stateIDs[d.home.State],
			FleetCodeID:             d.worker.FleetCodeID,
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    uniqueCode(fmt.Sprintf("%d", 1000+i), tractorTaken),
			Make:                    makeName,
			Model:                   pick(g.fake, mdTractorModels[makeName]),
			Year:                    &year,
			LicensePlateNumber:      g.fake.plate(),
			RegistrationNumber:      "REG" + g.fake.digits(8),
			RegistrationExpiry:      ptr(now + g.fake.between64(-20, 365)*mdDay),
			FuelType:                domaintypes.IFTAFuelTypeDiesel,
			IFTAQualified:           true,
			Vin:                     g.fake.vin(),
			OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
		}
		if g.fake.chance(0.02) {
			entity.FuelType = domaintypes.IFTAFuelTypeCNG
		}
		switch {
		case d.worker.Type == worker.WorkerTypeContractor:
			entity.OwnershipType = domaintypes.OwnershipTypeOwnerOperator
			entity.OwnerWorkerID = &d.worker.ID
		case g.fake.chance(0.18):
			entity.OwnershipType = domaintypes.OwnershipTypeLeased
			entity.LessorName = pick(g.fake, mdLessors)
			entity.LeaseReference = "LSE-" + g.fake.digits(7)
			entity.LeaseEndDate = ptr(now + g.fake.between64(60, 1_400)*mdDay)
		}

		usable := d.worker.Status == domaintypes.StatusActive
		switch {
		case !usable && g.fake.chance(0.5):
			entity.Status = domaintypes.EquipmentStatusSold
		case !usable:
			entity.Status = domaintypes.EquipmentStatusOOS
		default:
			switch roll := g.fake.intn(100); {
			case roll < 7:
				entity.Status = domaintypes.EquipmentStatusAtMaintenance
				usable = false
			case roll < 10:
				entity.Status = domaintypes.EquipmentStatusOOS
				usable = false
			}
		}

		if usable && len(available) > 0 {
			paired := available[nextTrailer%len(available)]
			paired.FleetCodeID = d.worker.FleetCodeID
			c := &crew{
				WorkerID:      d.worker.ID,
				TractorID:     entity.ID,
				TractorTypeID: entity.EquipmentTypeID,
				TrailerID:     paired.ID,
				TrailerTypeID: paired.EquipmentTypeID,
				TrailerKind:   equipment.kinds[paired.EquipmentTypeID],
				Hazmat:        d.profile.Endorsement.RequiresHazmatExpiry(),
				HomeState:     d.home.State,
			}
			nextTrailer++
			if nextTeammate < len(teammates) &&
				(d.worker.DriverType == worker.DriverTypeTeam || g.fake.chance(0.1)) {
				mate := teammates[nextTeammate].worker.ID
				nextTeammate++
				c.SecondaryWorkerID = &mate
				entity.SecondaryWorkerID = mate
			}
			g.crews = append(g.crews, c)
		}

		tractors = append(tractors, entity)
		tractorStamps = append(tractorStamps, g.mdStamp(entity.ID, at))
	}

	if err = insertRows(ctx, tx, tractordb.Tractors.Handle, tractors); err != nil {
		return fmt.Errorf("insert tractors: %w", err)
	}
	if err = insertRows(ctx, tx, trailerdb.Trailers.Handle, trailers); err != nil {
		return fmt.Errorf("insert trailers: %w", err)
	}
	if err = backdate(ctx, tx, tractordb.Tractors.Handle, tractorStamps); err != nil {
		return err
	}
	if err = backdate(ctx, tx, trailerdb.Trailers.Handle, trailerStamps); err != nil {
		return err
	}
	g.report.add("Tractors", len(tractors))
	g.report.add("Trailers", len(trailers))

	tractorOf := make(map[pulid.ID]pulid.ID, len(tractors))
	for _, t := range tractors {
		tractorOf[t.PrimaryWorkerID] = t.ID
	}
	g.drivers = make([]*driverRecord, 0, len(drivers))
	for _, d := range drivers {
		g.drivers = append(g.drivers, &driverRecord{
			WorkerID:    d.worker.ID,
			FleetCodeID: d.worker.FleetCodeID,
			Type:        d.worker.Type,
			DriverType:  d.worker.DriverType,
			Active:      d.worker.Status == domaintypes.StatusActive,
			HireDate:    d.profile.HireDate,
			Home:        d.home,
			TractorID:   tractorOf[d.worker.ID],
		})
	}

	return nil
}

func (g *generator) mdTrailers(
	equipment *mdEquipment,
	taken map[string]struct{},
) ([]*trailer.Trailer, []stamp) {
	n := g.scale.trailers
	now := unix(g.now)
	weights := g.mdTrailerTypeWeights(equipment.trailerTypes)
	trailerMakes := mdMakes(mdTrailerModels)
	rows := make([]*trailer.Trailer, 0, n)
	stamps := make([]stamp, 0, n)

	for i := range n {
		at := g.mdCreated()
		makeName := pick(g.fake, trailerMakes)
		year := g.fake.between(2010, g.now.Year())
		maxLoad := g.fake.between(43, 48) * 1_000
		state := pick(g.fake, cities).State
		row := &trailer.Trailer{
			ID:                      idAt("tr_", at),
			BusinessUnitID:          g.refs.buID,
			OrganizationID:          g.refs.orgID,
			EquipmentTypeID:         equipment.trailerTypes[g.fake.weighted(weights)].ID,
			EquipmentManufacturerID: equipment.manufacturers[strings.ToLower(makeName)],
			RegistrationStateID:     g.refs.stateIDs[state],
			FleetCodeID:             pick(g.fake, equipment.fleetCodes),
			Status:                  domaintypes.EquipmentStatusAvailable,
			Code:                    uniqueCode(fmt.Sprintf("53%04d", 1000+i), taken),
			Make:                    makeName,
			Model:                   pick(g.fake, mdTrailerModels[makeName]),
			Year:                    &year,
			LicensePlateNumber:      g.fake.plate(),
			Vin:                     g.fake.vin(),
			RegistrationNumber:      "TRG" + g.fake.digits(8),
			MaxLoadWeight:           &maxLoad,
			OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
			LastInspectionDate:      ptr(now - g.fake.between64(5, 360)*mdDay),
			RegistrationExpiry:      ptr(now + g.fake.between64(-15, 365)*mdDay),
		}
		if g.fake.chance(0.2) {
			row.OwnershipType = domaintypes.OwnershipTypeLeased
			row.LessorName = pick(g.fake, mdTrailerLessors)
			row.LeaseReference = "TL-" + g.fake.digits(7)
			row.LeaseEndDate = ptr(now + g.fake.between64(90, 1_800)*mdDay)
		}
		switch roll := g.fake.intn(100); {
		case roll < 6:
			row.Status = domaintypes.EquipmentStatusAtMaintenance
		case roll < 10:
			row.Status = domaintypes.EquipmentStatusOOS
		}
		rows = append(rows, row)
		stamps = append(stamps, g.mdStamp(row.ID, at))
	}

	return rows, stamps
}

var mdCarrierSuffixes = []string{
	"Transport", "Logistics", "Freight Lines", "Trucking", "Express", "Carriers", "Transportation",
	"Hauling", "Freightways", "Motor Lines",
}

func (g *generator) mdCarrierName() string {
	switch g.fake.intn(3) {
	case 0:
		return fmt.Sprintf("%s %s", pick(g.fake, companyPrefixes), pick(g.fake, mdCarrierSuffixes))
	case 1:
		return fmt.Sprintf("%s %s", g.fake.lastName(), pick(g.fake, mdCarrierSuffixes))
	default:
		return fmt.Sprintf("%s & Sons %s", g.fake.lastName(), pick(g.fake, mdCarrierSuffixes))
	}
}

func (g *generator) mdUniqueDigits(size int, taken map[string]struct{}) string {
	for {
		value := fmt.Sprintf("%d", g.fake.between(1, 9)) + g.fake.digits(size-1)
		if _, ok := taken[value]; !ok {
			taken[value] = struct{}{}
			return value
		}
	}
}

func (g *generator) mdUniqueSCAC(name string, taken map[string]struct{}) string {
	candidate := codeFrom(name, 4)
	for {
		key := strings.ToLower(candidate)
		if _, ok := taken[key]; !ok {
			taken[key] = struct{}{}
			return candidate
		}
		candidate = g.fake.letters(4)
	}
}

func (g *generator) mdCarriers(ctx context.Context, tx *keel.DB) error {
	codes, err := mdTaken(ctx, tx, carrierdb.Carriers.Handle, carrierdb.Carriers.Code)
	if err != nil {
		return err
	}
	scacs, err := mdTaken(ctx, tx, carrierdb.Carriers.Handle, carrierdb.Carriers.SCAC)
	if err != nil {
		return err
	}
	dots, err := mdTaken(ctx, tx, carrierdb.Carriers.Handle, carrierdb.Carriers.DOTNumber)
	if err != nil {
		return err
	}
	mcs, err := mdTaken(ctx, tx, carrierdb.Carriers.Handle, carrierdb.Carriers.MCNumber)
	if err != nil {
		return err
	}

	n := g.scale.carriers
	created, ids := g.mdTimes(n, "car_")
	rows := make([]*carrier.Carrier, 0, n)
	contacts := make([]*carrier.CarrierContact, 0, n*2)
	stamps := make([]stamp, 0, n)
	contactStamps := make([]stamp, 0, n*2)
	g.carriers = make([]pulid.ID, 0, n)
	ein := carrier.TaxIDTypeEIN

	for i := range n {
		name := g.mdCarrierName()
		slug := mdSlug(name)
		home := g.fake.city()
		remit := home
		if g.fake.chance(0.2) {
			remit = g.fake.city()
		}
		stateID := g.refs.stateIDs[home.State]
		remitStateID := g.refs.stateIDs[remit.State]

		row := &carrier.Carrier{
			ID:                ids[i],
			BusinessUnitID:    g.refs.buID,
			OrganizationID:    g.refs.orgID,
			StateID:           &stateID,
			RemitStateID:      &remitStateID,
			Status:            carrier.StatusActive,
			Code:              uniqueCode(codeFrom(name, g.fake.between(3, 5)), codes),
			Name:              name,
			CarrierType:       carrier.TypeCommon,
			DOTNumber:         g.mdUniqueDigits(7, dots),
			MCNumber:          g.mdUniqueDigits(6, mcs),
			SCAC:              g.mdUniqueSCAC(name, scacs),
			ComplianceStatus:  carrier.ComplianceStatusQualified,
			SafetyRating:      carrier.SafetyRatingSatisfactory,
			TaxID:             g.fake.digits(2) + "-" + g.fake.digits(7),
			TaxIDType:         &ein,
			W9OnFile:          g.fake.chance(0.9),
			PaymentMethod:     carrier.PaymentMethodACHManual,
			PaymentTermDays:   pick(g.fake, []int{15, 21, 30, 30, 45}),
			RemitToName:       name + " LLC",
			RemitAddressLine1: fmt.Sprintf("PO Box %d", g.fake.between(1000, 99999)),
			RemitCity:         remit.Name,
			RemitPostalCode:   g.fake.postalCode(remit),
			AddressLine1:      g.fake.street(),
			City:              home.Name,
			PostalCode:        g.fake.postalCode(home),
			Phone:             g.fake.phone(),
			Email:             "dispatch@" + slug + ".example.com",
		}
		if g.fake.chance(0.3) {
			row.DBAName = strings.Fields(name)[0]
		}
		if g.fake.chance(0.4) {
			row.PaymentMethod = carrier.PaymentMethodCheck
		}
		switch roll := g.fake.intn(100); {
		case roll < 30:
			row.CarrierType = carrier.TypeContract
			row.Is1099Eligible = true
		case roll < 35:
			row.CarrierType = carrier.TypeBroker
		case roll < 40:
			row.CarrierType = carrier.TypeExempt
		}
		switch roll := g.fake.intn(100); {
		case roll < 30:
			row.SafetyRating = carrier.SafetyRatingNotRated
		case roll < 38:
			row.SafetyRating = carrier.SafetyRatingConditional
		case roll < 40:
			row.SafetyRating = carrier.SafetyRatingUnsatisfactory
		}
		switch roll := g.fake.intn(100); {
		case roll < 12:
			row.ComplianceStatus = carrier.ComplianceStatusPending
		case roll < 20:
			row.ComplianceStatus = carrier.ComplianceStatusExpired
		case roll < 25:
			row.ComplianceStatus = carrier.ComplianceStatusDisqualified
			row.DisqualifiedReason = pick(g.fake, []string{
				"Insurance lapsed", "Operating authority revoked", "Unsatisfactory safety rating",
			})
		default:
			row.QualifiedAt = ptr(created[i] + g.fake.between64(1, 30)*mdDay)
		}
		switch roll := g.fake.intn(100); {
		case roll < 8:
			row.Status = carrier.StatusInactive
		case roll < 12:
			row.Status = carrier.StatusDoNotUse
			row.Notes = "Do not use: repeated late deliveries and claims."
		}
		if g.fake.chance(0.2) && row.Notes == "" {
			row.Notes = pick(g.fake, []string{
				"Strong on Texas triangle lanes.",
				"Reefer capacity out of the Southeast.",
				"Flatbed and step-deck specialists.",
				"Will cover weekend pickups with notice.",
				"Prefers drop-and-hook freight.",
			})
		}

		rows = append(rows, row)
		stamps = append(stamps, g.mdStamp(row.ID, created[i]))
		if row.Status == carrier.StatusActive &&
			row.ComplianceStatus == carrier.ComplianceStatusQualified {
			g.carriers = append(g.carriers, row.ID)
		}

		for c, role := range []struct {
			title   string
			mailbox string
		}{
			{title: "Dispatch Manager", mailbox: "dispatch"},
			{title: "Accounts Receivable", mailbox: "ar"},
		} {
			contact := &carrier.CarrierContact{
				ID:                        idAt("carc_", created[i]),
				BusinessUnitID:            g.refs.buID,
				OrganizationID:            g.refs.orgID,
				CarrierID:                 row.ID,
				Name:                      g.fake.firstName() + " " + g.fake.lastName(),
				Title:                     role.title,
				Email:                     role.mailbox + "@" + slug + ".example.com",
				Phone:                     g.fake.phone(),
				IsPrimary:                 c == 0,
				ReceivesRateConfirmations: c == 0,
			}
			contacts = append(contacts, contact)
			contactStamps = append(contactStamps, g.mdStamp(contact.ID, created[i]))
		}
	}

	if err = insertRows(ctx, tx, carrierdb.Carriers.Handle, rows); err != nil {
		return fmt.Errorf("insert carriers: %w", err)
	}
	if err = insertRows(ctx, tx, carrierdb.CarrierContacts.Handle, contacts); err != nil {
		return fmt.Errorf("insert carrier contacts: %w", err)
	}
	if err = backdate(ctx, tx, carrierdb.Carriers.Handle, stamps); err != nil {
		return err
	}
	if err = backdate(ctx, tx, carrierdb.CarrierContacts.Handle, contactStamps); err != nil {
		return err
	}
	g.report.add("Carriers", len(rows))
	g.report.add("Carrier contacts", len(contacts))

	return nil
}
