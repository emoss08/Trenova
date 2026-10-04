package onboardingservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/internal/core/services/locationcategoryservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/servicetypeservice"
	"github.com/emoss08/trenova/internal/core/services/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const (
	sampleStateCode      = "TX"
	flatRateTemplateName = "Flat Rate"
	sampleYearsOfAge     = 38
	sampleLicenseYears   = 3
	sampleHourStart      = 8
	samplePickupHours    = 24
	sampleDeliveryHours  = 54
	sampleWindowHours    = 2
	sampleWeightPounds   = 38_000
	samplePieces         = 22
)

type SampleDataRequest struct {
	TenantInfo   pagination.TenantInfo
	Actor        *services.RequestActor
	Organization *tenant.Organization
}

type entityCreator[T any] interface {
	Create(ctx context.Context, entity T, actor *services.RequestActor) (T, error)
}

type formulaTemplates interface {
	InstallStandards(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*formulatemplateservice.InstallStandardsResponse, error)
}

type stateFinder interface {
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
}

type formulaTemplateFinder interface {
	FindByNames(
		ctx context.Context,
		req repositories.GetFormulaTemplatesByNamesRequest,
	) ([]*formulatemplate.FormulaTemplate, error)
}

type SampleDataParams struct {
	fx.In

	Customers        *customerservice.Service
	Locations        *locationservice.Service
	LocationTypes    *locationcategoryservice.Service
	Workers          *workerservice.Service
	Tractors         *tractorservice.Service
	Trailers         *trailerservice.Service
	EquipmentTypes   *equipmenttypeservice.Service
	Manufacturers    *equipmentmanufacturerservice.Service
	ServiceTypes     *servicetypeservice.Service
	ShipmentTypes    *shipmenttypeservice.Service
	FormulaTemplates *formulatemplateservice.Service
	TemplateFinder   repositories.FormulaTemplateRepository
	States           repositories.UsStateRepository
	Shipments        services.ShipmentService
}

type SampleData struct {
	customers      entityCreator[*customer.Customer]
	locations      entityCreator[*location.Location]
	locationTypes  entityCreator[*locationcategory.LocationCategory]
	workers        entityCreator[*worker.Worker]
	tractors       entityCreator[*tractor.Tractor]
	trailers       entityCreator[*trailer.Trailer]
	equipmentTypes entityCreator[*equipmenttype.EquipmentType]
	manufacturers  entityCreator[*equipmentmanufacturer.EquipmentManufacturer]
	serviceTypes   entityCreator[*servicetype.ServiceType]
	shipmentTypes  entityCreator[*shipmenttype.ShipmentType]
	shipments      entityCreator[*shipment.Shipment]
	templates      formulaTemplates
	templateFinder formulaTemplateFinder
	states         stateFinder
	now            func() time.Time
}

func NewSampleData(p SampleDataParams) *SampleData {
	return &SampleData{
		customers:      p.Customers,
		locations:      p.Locations,
		locationTypes:  p.LocationTypes,
		workers:        p.Workers,
		tractors:       p.Tractors,
		trailers:       p.Trailers,
		equipmentTypes: p.EquipmentTypes,
		manufacturers:  p.Manufacturers,
		serviceTypes:   p.ServiceTypes,
		shipmentTypes:  p.ShipmentTypes,
		shipments:      p.Shipments,
		templates:      p.FormulaTemplates,
		templateFinder: p.TemplateFinder,
		states:         p.States,
		now:            time.Now,
	}
}

type sampleReferences struct {
	locationCategoryID pulid.ID
	tractorTypeID      pulid.ID
	trailerTypeID      pulid.ID
	manufacturerID     pulid.ID
	serviceTypeID      pulid.ID
	shipmentTypeID     pulid.ID
	formulaTemplateID  pulid.ID
}

type sampleAddress struct {
	line1      string
	city       string
	stateID    pulid.ID
	postalCode string
	latitude   float64
	longitude  float64
	timezone   string
}

type sampleSite struct {
	code    string
	name    string
	address sampleAddress
}

var sampleSites = []sampleSite{
	{
		code: "SMP-DAL-DC",
		name: "Sample Distribution Center",
		address: sampleAddress{
			line1: "1500 Commerce St", city: "Dallas", postalCode: "75201",
			latitude: 32.7801, longitude: -96.8005, timezone: "America/Chicago",
		},
	},
	{
		code: "SMP-FTW-WH",
		name: "Sample Warehouse",
		address: sampleAddress{
			line1: "300 Throckmorton St", city: "Fort Worth", postalCode: "76102",
			latitude: 32.7555, longitude: -97.3308, timezone: "America/Chicago",
		},
	},
	{
		code: "SMP-HOU-ST",
		name: "Sample Retail Store",
		address: sampleAddress{
			line1: "900 Main St", city: "Houston", postalCode: "77002",
			latitude: 29.7589, longitude: -95.3677, timezone: "America/Chicago",
		},
	},
	{
		code: "SMP-SAT-PL",
		name: "Sample Manufacturing Plant",
		address: sampleAddress{
			line1: "100 Alamo Plaza", city: "San Antonio", postalCode: "78205",
			latitude: 29.4246, longitude: -98.4951, timezone: "America/Chicago",
		},
	},
}

func (d *SampleData) Load(ctx context.Context, req *SampleDataRequest) error {
	if req.Organization == nil {
		return errortypes.NewBusinessError("The organization profile is required for sample data")
	}

	state, err := d.states.GetByAbbreviation(ctx, sampleStateCode)
	if err != nil {
		return fmt.Errorf("resolve the sample data state: %w", err)
	}

	refs, err := d.createReferences(ctx, req)
	if err != nil {
		return fmt.Errorf("create sample reference data: %w", err)
	}

	customers, err := d.createCustomers(ctx, req, state.ID)
	if err != nil {
		return err
	}

	locations, err := d.createLocations(ctx, req, refs, state.ID)
	if err != nil {
		return err
	}

	driver, err := d.createWorker(ctx, req, sampleAddress{
		line1:      req.Organization.AddressLine1,
		city:       req.Organization.City,
		stateID:    req.Organization.StateID,
		postalCode: req.Organization.PostalCode,
	})
	if err != nil {
		return err
	}

	if err = d.createEquipment(ctx, req, refs, driver.ID); err != nil {
		return err
	}

	return d.createShipments(ctx, req, refs, customers, locations)
}

func (d *SampleData) createReferences(
	ctx context.Context,
	req *SampleDataRequest,
) (*sampleReferences, error) {
	tenantInfo := req.TenantInfo
	refs := new(sampleReferences)

	category, err := d.locationTypes.Create(ctx, &locationcategory.LocationCategory{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Customer Facility",
		Description:    "Sample category created with the onboarding sample data",
		Type:           locationcategory.CategoryCustomerLocation,
		Color:          "#0ea5e9",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.locationCategoryID = category.ID

	tractorType, err := d.equipmentTypes.Create(ctx, &equipmenttype.EquipmentType{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           "TRACTOR",
		Description:    "Sample tractor type",
		Class:          equipmenttype.ClassTractor,
		Color:          "#171717",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.tractorTypeID = tractorType.ID

	trailerType, err := d.equipmentTypes.Create(ctx, &equipmenttype.EquipmentType{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           "DRYVAN",
		Description:    "Sample 53 ft dry van",
		Class:          equipmenttype.ClassTrailer,
		Color:          "#525252",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.trailerTypeID = trailerType.ID

	manufacturer, err := d.manufacturers.Create(ctx, &equipmentmanufacturer.EquipmentManufacturer{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Name:           "Sample Manufacturer",
		Description:    "Created with the onboarding sample data",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.manufacturerID = manufacturer.ID

	serviceType, err := d.serviceTypes.Create(ctx, &servicetype.ServiceType{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           "STD",
		Description:    "Standard service",
		Color:          "#16a34a",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.serviceTypeID = serviceType.ID

	shipmentType, err := d.shipmentTypes.Create(ctx, &shipmenttype.ShipmentType{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Code:           "FTL",
		Description:    "Full truckload",
		Color:          "#2563eb",
	}, req.Actor)
	if err != nil {
		return nil, err
	}
	refs.shipmentTypeID = shipmentType.ID

	refs.formulaTemplateID, err = d.flatRateTemplate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return refs, nil
}

func (d *SampleData) flatRateTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (pulid.ID, error) {
	installed, err := d.templates.InstallStandards(ctx, tenantInfo)
	if err != nil {
		return pulid.Nil, err
	}

	for _, template := range installed.Installed {
		if template != nil && template.Name == flatRateTemplateName {
			return template.ID, nil
		}
	}

	existing, err := d.templateFinder.FindByNames(ctx, repositories.GetFormulaTemplatesByNamesRequest{
		TenantInfo: tenantInfo,
		Names:      []string{flatRateTemplateName},
	})
	if err != nil {
		return pulid.Nil, err
	}
	for _, template := range existing {
		if template != nil && template.Name == flatRateTemplateName {
			return template.ID, nil
		}
	}

	return pulid.Nil, errortypes.NewBusinessError("The flat rate formula template is not available")
}

func (d *SampleData) createCustomers(
	ctx context.Context,
	req *SampleDataRequest,
	stateID pulid.ID,
) ([]*customer.Customer, error) {
	specs := []struct {
		code string
		name string
		site sampleSite
	}{
		{code: "SAMPLE1", name: "Sample Shipper Co", site: sampleSites[3]},
		{code: "SAMPLE2", name: "Sample Retail Group", site: sampleSites[2]},
	}

	created := make([]*customer.Customer, 0, len(specs))
	for _, spec := range specs {
		entity, err := d.customers.Create(ctx, &customer.Customer{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Status:         domaintypes.StatusActive,
			Code:           spec.code,
			Name:           spec.name,
			AddressLine1:   spec.site.address.line1,
			City:           spec.site.address.city,
			StateID:        stateID,
			PostalCode:     spec.site.address.postalCode,
		}, req.Actor)
		if err != nil {
			return nil, err
		}
		created = append(created, entity)
	}

	return created, nil
}

func (d *SampleData) createLocations(
	ctx context.Context,
	req *SampleDataRequest,
	refs *sampleReferences,
	stateID pulid.ID,
) ([]*location.Location, error) {
	created := make([]*location.Location, 0, len(sampleSites))
	for _, site := range sampleSites {
		latitude := site.address.latitude
		longitude := site.address.longitude
		radius := geofence.DefaultRadiusMeters

		entity, err := d.locations.Create(ctx, &location.Location{
			OrganizationID:       req.TenantInfo.OrgID,
			BusinessUnitID:       req.TenantInfo.BuID,
			LocationCategoryID:   refs.locationCategoryID,
			Status:               domaintypes.StatusActive,
			Code:                 site.code,
			Name:                 site.name,
			AddressLine1:         site.address.line1,
			City:                 site.address.city,
			StateID:              stateID,
			PostalCode:           site.address.postalCode,
			Timezone:             site.address.timezone,
			Latitude:             &latitude,
			Longitude:            &longitude,
			GeofenceType:         geofence.TypeAuto,
			GeofenceRadiusMeters: &radius,
		}, req.Actor)
		if err != nil {
			return nil, err
		}
		created = append(created, entity)
	}

	return created, nil
}

func (d *SampleData) createWorker(
	ctx context.Context,
	req *SampleDataRequest,
	address sampleAddress,
) (*worker.Worker, error) {
	now := d.now()

	return d.workers.Create(ctx, &worker.Worker{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		FirstName:      "Sam",
		LastName:       "Sample",
		Gender:         worker.GenderMale,
		AddressLine1:   address.line1,
		City:           address.city,
		StateID:        address.stateID,
		PostalCode:     address.postalCode,
		Profile: &worker.WorkerProfile{
			OrganizationID:   req.TenantInfo.OrgID,
			BusinessUnitID:   req.TenantInfo.BuID,
			LicenseStateID:   address.stateID,
			DOB:              now.AddDate(-sampleYearsOfAge, 0, 0).Unix(),
			LicenseNumber:    "SAMPLE0001",
			CDLClass:         worker.CDLClassA,
			Endorsement:      worker.EndorsementTypeNone,
			LicenseExpiry:    now.AddDate(sampleLicenseYears, 0, 0).Unix(),
			HireDate:         now.AddDate(-1, 0, 0).Unix(),
			ComplianceStatus: worker.ComplianceStatusPending,
		},
	}, req.Actor)
}

func (d *SampleData) createEquipment(
	ctx context.Context,
	req *SampleDataRequest,
	refs *sampleReferences,
	driverID pulid.ID,
) error {
	if _, err := d.tractors.Create(ctx, &tractor.Tractor{
		OrganizationID:          req.TenantInfo.OrgID,
		BusinessUnitID:          req.TenantInfo.BuID,
		EquipmentTypeID:         refs.tractorTypeID,
		EquipmentManufacturerID: refs.manufacturerID,
		PrimaryWorkerID:         driverID,
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "SAMPLE-T1",
		FuelType:                domaintypes.IFTAFuelTypeDiesel,
		OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
	}, req.Actor); err != nil {
		return err
	}

	_, err := d.trailers.Create(ctx, &trailer.Trailer{
		OrganizationID:          req.TenantInfo.OrgID,
		BusinessUnitID:          req.TenantInfo.BuID,
		EquipmentTypeID:         refs.trailerTypeID,
		EquipmentManufacturerID: refs.manufacturerID,
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "SAMPLE-V1",
		OwnershipType:           domaintypes.OwnershipTypeCompanyOwned,
	}, req.Actor)

	return err
}

func (d *SampleData) createShipments(
	ctx context.Context,
	req *SampleDataRequest,
	refs *sampleReferences,
	customers []*customer.Customer,
	locations []*location.Location,
) error {
	loc := timeutils.LoadLocation(sampleSites[0].address.timezone)
	day := time.Unix(timeutils.DayStart(d.now().Unix(), loc), 0).In(loc)
	plans := []struct {
		customer *customer.Customer
		pickup   *location.Location
		delivery *location.Location
		bol      string
		rate     int64
		dayShift int
	}{
		{customer: customers[0], pickup: locations[3], delivery: locations[0], bol: "SAMPLE-BOL-1001", rate: 1850, dayShift: 0},
		{customer: customers[1], pickup: locations[1], delivery: locations[2], bol: "SAMPLE-BOL-1002", rate: 1275, dayShift: 1},
	}

	for _, plan := range plans {
		base := day.AddDate(0, 0, plan.dayShift)
		pickupAt := base.Add(time.Duration(samplePickupHours+sampleHourStart) * time.Hour)
		deliveryAt := base.Add(time.Duration(sampleDeliveryHours+sampleHourStart) * time.Hour)
		weight := int64(sampleWeightPounds)
		pieces := int64(samplePieces)

		if _, err := d.shipments.Create(ctx, &shipment.Shipment{
			OrganizationID:    req.TenantInfo.OrgID,
			BusinessUnitID:    req.TenantInfo.BuID,
			EnteredByID:       req.TenantInfo.UserID,
			Status:            shipment.StatusNew,
			EntryMethod:       shipment.EntryMethodManual,
			FreightTerms:      shipment.FreightTermsPrepaid,
			CustomerID:        plan.customer.ID,
			ServiceTypeID:     refs.serviceTypeID,
			ShipmentTypeID:    refs.shipmentTypeID,
			FormulaTemplateID: refs.formulaTemplateID,
			TractorTypeID:     refs.tractorTypeID,
			TrailerTypeID:     refs.trailerTypeID,
			BOL:               plan.bol,
			BaseRate:          decimal.NewNullDecimal(decimal.NewFromInt(plan.rate)),
			RatingUnit:        1,
			Pieces:            &pieces,
			Weight:            &weight,
			Moves: []*shipment.ShipmentMove{{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				Status:         shipment.MoveStatusNew,
				Loaded:         true,
				Sequence:       0,
				Stops: []*shipment.Stop{
					sampleStop(req.TenantInfo, plan.pickup.ID, shipment.StopTypePickup, 0, pickupAt),
					sampleStop(req.TenantInfo, plan.delivery.ID, shipment.StopTypeDelivery, 1, deliveryAt),
				},
			}},
		}, req.Actor); err != nil {
			return err
		}
	}

	return nil
}

func sampleStop(
	tenantInfo pagination.TenantInfo,
	locationID pulid.ID,
	stopType shipment.StopType,
	sequence int64,
	at time.Time,
) *shipment.Stop {
	end := at.Add(sampleWindowHours * time.Hour).Unix()

	return &shipment.Stop{
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		LocationID:           locationID,
		Status:               shipment.StopStatusNew,
		Type:                 stopType,
		ScheduleType:         shipment.StopScheduleTypeAppointment,
		Sequence:             sequence,
		ScheduledWindowStart: at.Unix(),
		ScheduledWindowEnd:   &end,
	}
}
