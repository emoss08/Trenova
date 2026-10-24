//go:build integration

package agentquerytoolservice_test

import (
	"github.com/emoss08/keel"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmentmanufacturerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/fleetcoderepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/tractorrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/trailerrepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/fleetcodeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/workerptoservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationcategoryrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/modules/places/app/locationcategoryservice"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulalibraryversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplaterepository"
	"github.com/emoss08/trenova/internal/modules/rating/adapters/postgres/formulatemplateversionrepository"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/accessorialchargerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/commodityrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/documenttyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/hazardousmaterialrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/holdreasonrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/servicetyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/shipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/reference/app/accessorialchargeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/commodityservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/documenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/hazardousmaterialservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/holdreasonservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/servicetypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/app/shipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/fuelpurchaserepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/iftarepository"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/iftaservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerptorepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workertrainingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/customfieldvaluerepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/customfieldservice"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	referenceRowsPerTenant = 5
	referencePage          = 2
	walkLimit              = 20
)

// referenceCase is one reference list read against real repositories: how
// to write a row for a tenant (marked rows are the ones its filter selects),
// and that filter as the GraphQL tool and the repository tool each take it.
type referenceCase struct {
	// readsActiveByDefault marks a list whose call with no arguments leaves
	// out the marked rows (training courses read the active ones).
	readsActiveByDefault bool
	tool                 string
	write                func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error)
	marked               map[string]any
	oldMarked            map[string]any
	old                  agentports.AgentQueryTool
	// readsWith attaches what a read of the case needs beside the tenant,
	// such as the loader a card's counts resolve through.
	readsWith func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context
	// withheldAtInternal are the fields a get withholds even at the default
	// agent data access (a detention occurrence's margin, which needs the
	// Margins field group); a case that names none withholds nothing.
	withheldAtInternal []string
}

// customerCacheMiss is a customer cache that holds nothing, so every get
// reads the repository.
type customerCacheMiss struct{}

func (customerCacheMiss) GetByID(
	context.Context,
	customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return nil, customerports.ErrCacheMiss
}

// workerCacheMiss is a worker cache that holds nothing, so every get reads
// the repository.
type workerCacheMiss struct{}

func (workerCacheMiss) GetByID(
	context.Context,
	workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return nil, customerports.ErrCacheMiss
}

func statusOf(marked bool) domaintypes.Status {
	if marked {
		return domaintypes.StatusInactive
	}

	return domaintypes.StatusActive
}

var (
	inactive    = map[string]any{"status": []any{"Inactive"}}
	oldInactive = map[string]any{"filters": []any{map[string]any{
		"field": "status", "operator": "in", "values": []any{"Inactive"},
	}}}
)

func boolFilter(field string, value bool) (typed, old map[string]any) {
	return map[string]any{field: value}, map[string]any{"filters": []any{map[string]any{
		"field": field, "operator": "eq", "value": strconv.FormatBool(value),
	}}}
}

// convertedListCases builds every list converted by a spec over one RLS
// connection, and the resolver services the read executor reaches. A case
// whose rows need records no repository here writes (a pay event's
// shipment) writes them through admin, which row-level security does not
// hold.
//
//nolint:funlen // one entry per converted list
func convertedListCases(
	conn *postgres.Connection,
	admin *keel.DB,
) ([]referenceCase, agentread.ResolverParams) {
	logger := zap.NewNop()
	equipmentTypes := equipmenttyperepository.New(equipmenttyperepository.Params{DB: conn, Logger: logger})
	fleetCodes := fleetcoderepository.New(fleetcoderepository.Params{DB: conn, Logger: logger})
	serviceTypes := servicetyperepository.New(servicetyperepository.Params{DB: conn, Logger: logger})
	shipmentTypes := shipmenttyperepository.New(shipmenttyperepository.Params{DB: conn, Logger: logger})
	commodities := commodityrepository.New(commodityrepository.Params{DB: conn, Logger: logger})
	hazmat := hazardousmaterialrepository.New(hazardousmaterialrepository.Params{DB: conn, Logger: logger})
	accessorials := accessorialchargerepository.New(accessorialchargerepository.Params{DB: conn, Logger: logger})
	documentTypes := documenttyperepository.New(documenttyperepository.Params{DB: conn, Logger: logger})
	categories := locationcategoryrepository.New(locationcategoryrepository.Params{DB: conn, Logger: logger})
	holdReasons := holdreasonrepository.New(holdreasonrepository.Params{DB: conn, Logger: logger})
	customers := customerrepository.New(customerrepository.Params{
		DB: conn, Logger: logger,
	})
	locations := locationrepository.New(locationrepository.Params{DB: conn, Logger: logger})
	carriers := carrierrepository.New(carrierrepository.Params{DB: conn, Logger: logger})
	workers := workerrepository.New(workerrepository.Params{DB: conn, Logger: logger})
	training := workertrainingrepository.New(workertrainingrepository.Params{DB: conn, Logger: logger})
	timeOff := workerptorepository.New(workerptorepository.Params{DB: conn, Logger: logger})
	manufacturers := equipmentmanufacturerrepository.New(
		equipmentmanufacturerrepository.Params{DB: conn, Logger: logger})
	tractors := tractorrepository.New(tractorrepository.Params{DB: conn, Logger: logger})
	trailers := trailerrepository.New(trailerrepository.Params{DB: conn, Logger: logger})
	customFields := customfieldservice.NewValuesService(customfieldservice.ValuesServiceParams{
		Logger: logger,
		ValueRepo: customfieldvaluerepository.New(
			customfieldvaluerepository.Params{DB: conn, Logger: logger}),
		DefinitionRepo: customfieldrepository.New(
			customfieldrepository.Params{DB: conn, Logger: logger}),
	})
	parts := &equipmentParts{
		types: equipmentTypes, manufacturers: manufacturers, workers: workers,
	}

	formulaTemplates := formulatemplaterepository.New(
		formulatemplaterepository.Params{DB: conn, Logger: logger})
	formulaVersions := formulatemplateversionrepository.New(
		formulatemplateversionrepository.Params{DB: conn, Logger: logger})
	libraries := formulalibraryrepository.New(
		formulalibraryrepository.Params{DB: conn, Logger: logger})
	libraryVersions := formulalibraryversionrepository.New(
		formulalibraryversionrepository.Params{DB: conn, Logger: logger})
	fuel := fuelpurchaserepository.New(fuelpurchaserepository.Params{DB: conn, Logger: logger})
	iftaRepo := iftarepository.New(iftarepository.Params{
		DB: conn, Logger: logger, JurisdictionCache: jurisdictionCacheMiss{},
	})
	fuelParts := &fuelParts{parts: parts, tractors: tractors}

	services := agentread.ResolverParams{
		EquipmentTypeService: equipmenttypeservice.New(
			equipmenttypeservice.Params{Logger: logger, Repo: equipmentTypes}),
		FleetCodeService: fleetcodeservice.New(fleetcodeservice.Params{Logger: logger, Repo: fleetCodes}),
		ServiceTypeService: servicetypeservice.New(
			servicetypeservice.Params{Logger: logger, Repo: serviceTypes}),
		ShipmentTypeService: shipmenttypeservice.New(
			shipmenttypeservice.Params{Logger: logger, Repo: shipmentTypes}),
		CommodityService: commodityservice.New(commodityservice.Params{Logger: logger, Repo: commodities}),
		HazardousMaterialService: hazardousmaterialservice.New(
			hazardousmaterialservice.Params{Logger: logger, Repo: hazmat}),
		AccessorialChargeService: accessorialchargeservice.New(
			accessorialchargeservice.Params{Logger: logger, Repo: accessorials}),
		DocumentTypeService: documenttypeservice.New(
			documenttypeservice.Params{Logger: logger, Repo: documentTypes}),
		LocationCategoryService: locationcategoryservice.New(
			locationcategoryservice.Params{Logger: logger, Repo: categories}),
		HoldReasonService: holdreasonservice.New(holdreasonservice.Params{Logger: logger, Repo: holdReasons}),
		CustomerService: customerservice.New(customerservice.Params{
			Logger: logger, Repo: customers, CacheRepo: customerCacheMiss{},
		}),
		LocationService: locationservice.New(locationservice.Params{Logger: logger, Repo: locations}),
		CarrierService:  carrierservice.New(carrierservice.Params{Logger: logger, Repo: carriers}),
		TractorService: tractorservice.New(tractorservice.Params{
			Logger: logger, Repo: tractors, CustomFieldsValuesService: customFields,
		}),
		WorkerTrainingService: workertrainingservice.New(workertrainingservice.Params{
			Logger: logger, Repo: training, WorkerRepo: workers,
		}),
		WorkerService: workerservice.New(workerservice.Params{
			Logger: logger, Repo: workers, CacheRepo: workerCacheMiss{},
			CustomFieldsValuesService: customFields,
		}),
		WorkerPTOService: workerptoservice.New(workerptoservice.Params{
			Logger: logger, Repo: timeOff, WorkerRepo: workers,
		}),
		TrailerService: trailerservice.New(trailerservice.Params{
			Logger: logger, Repo: trailers, CustomFieldsValuesService: customFields,
		}),
		FormulaTemplateService: formulatemplateservice.New(formulatemplateservice.Params{
			Logger: logger, Repo: formulaTemplates, VersionRepo: formulaVersions,
			Resolver: tariff.NewTemplateResolver(tariff.TemplateResolverParams{
				Templates: formulaTemplates, Versions: formulaVersions,
				LibraryVersions: libraryVersions,
			}),
		}),
		FormulaLibraryService: formulalibraryservice.New(formulalibraryservice.Params{
			Logger: logger, Repo: libraries, VersionRepo: libraryVersions,
		}),
		FuelPurchaseService: fuelpurchaseservice.New(fuelpurchaseservice.Params{
			Logger: logger, Repo: fuel, IFTARepo: iftaRepo,
		}),
		IFTAService: iftaservice.New(iftaservice.Params{Logger: logger, Repo: iftaRepo, Fuel: fuel}),
	}

	formulaStats := loaders.NewFormulaTemplateStatsLoaderFactory(
		loaders.FormulaTemplateStatsLoaderFactoryParams{TemplateRepo: formulaTemplates})
	systemTyped, systemOld := boolFilter("isSystem", true)
	parkingTyped, parkingOld := boolFilter("hasSecureParking", true)
	retiredTyped, retiredOld := boolFilter("active", false)
	tenantOf := func(tenant *agentreadtest.Tenant) (pulid.ID, pulid.ID) {
		return tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	}

	cases := []referenceCase{
		{
			tool: "equipment_list_types", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListEquipmentTypesTool(equipmentTypes),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := equipmentTypes.Create(ctx, &equipmenttype.EquipmentType{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
					Class: equipmenttype.ClassTrailer, DeckType: equipmenttype.DeckTypeFlatbed,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_fleet_codes", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListFleetCodesTool(fleetCodes),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := fleetCodes.Create(ctx, &fleetcode.FleetCode{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
					ManagerID: tenant.Actor.UserID,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_service_types", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListServiceTypesTool(serviceTypes),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := serviceTypes.Create(ctx, &servicetype.ServiceType{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_shipment_types", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListShipmentTypesTool(shipmentTypes),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := shipmentTypes.Create(ctx, &shipmenttype.ShipmentType{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_commodities", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListCommoditiesTool(commodities),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := commodities.Create(ctx, &commodity.Commodity{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Name: label,
					FreightClass: commodity.FreightClass50,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_hazardous_materials", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListHazardousMaterialsTool(hazmat),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := hazmat.Create(ctx, &hazardousmaterial.HazardousMaterial{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
					Name: label, Class: hazardousmaterial.HazardousClass1,
					PackingGroup: hazardousmaterial.PackingGroupI,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_accessorial_charges", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListAccessorialChargesTool(accessorials),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := accessorials.Create(ctx, &accessorialcharge.AccessorialCharge{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: "ACC" + label,
					Method: accessorialcharge.MethodFlat, RateUnit: accessorialcharge.RateUnitMile,
					Amount: decimal.NewFromInt(75),
				})
				return created(row, err)
			},
		},
		{
			tool: "document_list_types", marked: systemTyped, oldMarked: systemOld,
			old: agentquerytoolservice.NewListDocumentTypesTool(documentTypes),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := documentTypes.Create(ctx, &documenttype.DocumentType{
					BusinessUnitID: bu, OrganizationID: org, Code: label, Name: label,
					DocumentClassification: documenttype.ClassificationPublic,
					DocumentCategory:       documenttype.CategoryShipment, IsSystem: marked,
				})
				return created(row, err)
			},
		},
		{
			tool: "location_list_categories", marked: parkingTyped, oldMarked: parkingOld,
			old: agentquerytoolservice.NewListLocationCategoriesTool(categories),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := categories.Create(ctx, &locationcategory.LocationCategory{
					BusinessUnitID: bu, OrganizationID: org, Name: label,
					Type:             locationcategory.CategoryTerminal,
					FacilityType:     locationcategory.FacilityTypeCrossDock,
					HasSecureParking: marked,
				})
				return created(row, err)
			},
		},
		{
			tool: "masterdata_list_hold_reasons", marked: retiredTyped, oldMarked: retiredOld,
			old: agentquerytoolservice.NewListHoldReasonsTool(holdReasons),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := holdReasons.Create(ctx, &holdreason.HoldReason{
					BusinessUnitID: bu, OrganizationID: org, Code: label, Label: label,
					Type:            holdreason.HoldTypeOperational,
					DefaultSeverity: holdreason.HoldSeverityInformational,
					Active:          !marked,
				})
				return created(row, err)
			},
		},
		{
			tool: "customer_list", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListCustomersTool(customers),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := customers.Create(ctx, &customer.Customer{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
					Name: label, StateID: tenant.StateID, AddressLine1: "1 Main", City: "Dallas",
					PostalCode: "75001",
				})
				return created(row, err)
			},
		},
		{
			tool: "location_list", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListLocationsTool(locations),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				category, err := categories.Create(ctx, &locationcategory.LocationCategory{
					BusinessUnitID: bu, OrganizationID: org, Name: "Yard " + label,
					Type:         locationcategory.CategoryTerminal,
					FacilityType: locationcategory.FacilityTypeCrossDock,
				})
				if err != nil {
					return pulid.Nil, err
				}
				row, err := locations.Create(ctx, &location.Location{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
					Name: label, LocationCategoryID: category.ID, StateID: tenant.StateID,
					AddressLine1: "1 Main", City: "Dallas", PostalCode: "75001",
					GeofenceType: geofence.TypeAuto,
				})
				return created(row, err)
			},
		},
		{
			tool: "carrier_list", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListCarriersTool(carriers),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := carriers.Create(ctx, &carrier.Carrier{
					BusinessUnitID: bu, OrganizationID: org,
					Status: carrier.Status(statusOf(marked)), Code: label,
					Name: label,
				})
				return created(row, err)
			},
		},
		{
			tool: "worker_list", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListWorkersTool(workers, &mocks.DenyResourcesPermissionEngine{}),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				row, err := workers.Create(ctx, testWorker(tenant, "W"+label, marked, pulid.Nil))
				return created(row, err)
			},
		},
		{
			tool: "equipment_list_tractors", marked: outOfService, oldMarked: oldOutOfService,
			old: agentquerytoolservice.NewListTractorsTool(tractors, &mocks.DenyResourcesPermissionEngine{}),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				typeID, makeID, driverID, err := parts.write(ctx, tenant, "P"+label)
				if err != nil {
					return pulid.Nil, err
				}
				bu, org := tenantOf(tenant)
				row, err := tractors.Create(ctx, &tractor.Tractor{
					BusinessUnitID: bu, OrganizationID: org, Status: equipmentStatusOf(marked),
					Code: label, EquipmentTypeID: typeID, EquipmentManufacturerID: makeID,
					PrimaryWorkerID: driverID, FuelType: domaintypes.IFTAFuelTypeDiesel,
				})
				return created(row, err)
			},
		},
		{
			tool: "equipment_list_trailers", marked: outOfService, oldMarked: oldOutOfService,
			old: agentquerytoolservice.NewListTrailersTool(trailers),
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				typeID, makeID, _, err := parts.write(ctx, tenant, "R"+label)
				if err != nil {
					return pulid.Nil, err
				}
				bu, org := tenantOf(tenant)
				row, err := trailers.Create(ctx, &trailer.Trailer{
					BusinessUnitID: bu, OrganizationID: org, Status: equipmentStatusOf(marked),
					Code: label, EquipmentTypeID: typeID, EquipmentManufacturerID: makeID,
				})
				return created(row, err)
			},
		},
		{
			tool: "worker_list_training_courses", marked: inactive, readsActiveByDefault: true,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := training.CreateCourse(ctx, &worker.TrainingCourse{
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked),
					Code: label, Name: "Course " + label, Category: worker.TrainingCategorySafety,
					Delivery: worker.TrainingDeliveryOnline, DurationMinutes: 30,
				})
				return created(row, err)
			},
		},
		{
			tool: "worker_list_time_off", marked: approved,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				driver, err := workers.Create(ctx, testWorker(tenant, "T"+label, false, pulid.Nil))
				if err != nil {
					return pulid.Nil, err
				}
				return writeLeaveUnderWay(ctx, timeOff, driver, marked)
			},
		},
		{
			tool: "formula_list_templates", marked: inactive, oldMarked: oldInactive,
			old: agentquerytoolservice.NewListFormulaTemplatesTool(formulaTemplates),
			readsWith: func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context {
				return formulaStats.Attach(ctx, pagination.TenantInfo{
					OrgID:  tenant.Actor.OrganizationID,
					BuID:   tenant.Actor.BusinessUnitID,
					UserID: tenant.Actor.UserID,
				})
			},
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := formulaTemplates.Create(ctx, &formulatemplate.FormulaTemplate{
					BusinessUnitID: bu, OrganizationID: org, Name: "Formula " + label,
					Source: "formula\n", Status: formulaStatusOf(marked),
				})
				return created(row, err)
			},
		},
		{
			tool: "formula_list_libraries", marked: inactive,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				row, err := libraries.Create(ctx, &formulalibrary.FormulaLibrary{
					BusinessUnitID: bu, OrganizationID: org, Name: "lib" + strings.ToLower(label),
					Source: "library\n", Status: formulaStatusOf(marked),
				})
				return created(row, err)
			},
		},
		{
			tool: "fuel_list_cards", marked: suspended,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status := fuelpurchase.CardStatusActive
				if marked {
					status = fuelpurchase.CardStatusSuspended
				}
				row, err := fuel.CreateCard(ctx, &fuelpurchase.FuelCard{
					BusinessUnitID: bu, OrganizationID: org, Provider: fuelpurchase.CardProviderComdata,
					LastFour: fmt.Sprintf("%04d", len(label)*100+int(label[len(label)-1])),
					Label:    "Card " + label, Status: status,
				})
				return created(row, err)
			},
		},
		{
			tool: "fuel_list_purchases", marked: taxPaidTyped,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				tractorID, jurisdictionID, err := fuelParts.write(ctx, tenant, "F"+label)
				if err != nil {
					return pulid.Nil, err
				}
				bu, org := tenantOf(tenant)
				purchase := &fuelpurchase.FuelPurchase{
					BusinessUnitID: bu, OrganizationID: org, TractorID: tractorID,
					JurisdictionID: jurisdictionID, PurchasedAt: timeutils.NowUnix() - 86400,
					Vendor: "Pilot", FuelType: domaintypes.IFTAFuelTypeDiesel,
					Quantity: decimal.NewFromInt(100), QuantityUnit: fuelpurchase.QuantityUnitGallon,
					Gallons: decimal.NewFromInt(100), TotalAmountMinor: 38990, CurrencyCode: "USD",
					TransactionReference: "ref-" + label, Source: fuelpurchase.PurchaseSourceManual,
					TaxPaid: marked, CreatedByID: tenant.Actor.UserID,
				}
				purchase.Normalize()
				row, err := fuel.CreatePurchase(ctx, purchase)
				return created(row, err)
			},
		},
		{
			tool: "fuel_list_purchase_imports", marked: failedImports,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status := fuelpurchase.ImportStatusPending
				if marked {
					status = fuelpurchase.ImportStatusFailed
				}
				row, err := fuel.CreateImportBatch(ctx, &fuelpurchase.ImportBatch{
					ID: pulid.MustNew("fpib_"), BusinessUnitID: bu, OrganizationID: org,
					Provider: fuelpurchase.CardProviderComdata, Origin: fuelpurchase.ImportOriginUpload,
					Status: status, FileName: "statement-" + label + ".csv", DefaultCurrency: "USD",
					RowCount: 3,
				})
				return created(row, err)
			},
		},
		{
			tool: "fuel_list_ifta_returns", marked: finalized,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				bu, org := tenantOf(tenant)
				status := ifta.ReturnStatusDraft
				var finalizedAt *int64
				if marked {
					status = ifta.ReturnStatusFinalized
					finalizedAt = new(timeutils.NowUnix())
				}
				index := int(label[len(label)-1] - '0')
				row, err := iftaRepo.CreateReturn(ctx, &ifta.Return{
					BusinessUnitID: bu, OrganizationID: org, Year: 2020 + index, Quarter: 1,
					Status: status, FinalizedAt: finalizedAt, Timezone: "America/Chicago",
					PeriodStart: 1, PeriodEnd: 2,
					CurrencyCode: "USD",
					Problems:     []ifta.Problem{{Code: ifta.ProblemMissingRate, Message: "No rate"}},
				})
				return created(row, err)
			},
		},
		{
			tool: "fuel_list_ifta_mileage_entries", marked: loadedTyped,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				tractorID, jurisdictionID, err := fuelParts.write(ctx, tenant, "M"+label)
				if err != nil {
					return pulid.Nil, err
				}
				bu, org := tenantOf(tenant)
				row, err := iftaRepo.CreateMileageEntry(ctx, &ifta.JurisdictionMileageEntry{
					BusinessUnitID: bu, OrganizationID: org, TractorID: tractorID,
					JurisdictionID: jurisdictionID, TraveledAt: timeutils.NowUnix() - 86400,
					Year: 2026, Quarter: 3, Miles: decimal.NewFromInt(120), Loaded: marked,
					Source: ifta.MileageSourceManual, CreatedByID: tenant.Actor.UserID,
				})
				return created(row, err)
			},
		},
	}
	cases = append(cases, settlementListCases(conn, admin, &services)...)

	return cases, services
}

func formulaStatusOf(marked bool) formulatemplate.Status {
	if marked {
		return formulatemplate.StatusInactive
	}

	return formulatemplate.StatusActive
}

var (
	suspended     = map[string]any{"status": []any{"Suspended"}}
	failedImports = map[string]any{"status": []any{"Failed"}}
	finalized     = map[string]any{"status": []any{"Finalized"}}
	taxPaidTyped  = map[string]any{"taxPaid": true}
	loadedTyped   = map[string]any{"loaded": true}
)

// jurisdictionCacheMiss is a jurisdiction cache that holds nothing, so every
// read loads the global list.
type jurisdictionCacheMiss struct{}

func (jurisdictionCacheMiss) GetAll(context.Context) ([]*ifta.Jurisdiction, error) {
	return nil, nil
}

func (jurisdictionCacheMiss) Set(context.Context, []*ifta.Jurisdiction) error { return nil }

func (jurisdictionCacheMiss) Invalidate(context.Context) error { return nil }

// fuelParts writes what a fuel purchase or a mileage entry names: a tractor,
// with its type, make and driver, beside the tenant's own jurisdiction.
type fuelParts struct {
	parts    *equipmentParts
	tractors tractorports.TractorRepository
}

func (p *fuelParts) write(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
) (tractorID, jurisdictionID pulid.ID, err error) {
	typeID, makeID, driverID, err := p.parts.write(ctx, tenant, label)
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	saved, err := p.tractors.Create(ctx, &tractor.Tractor{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		Status: domaintypes.EquipmentStatusAvailable, Code: label, EquipmentTypeID: typeID,
		EquipmentManufacturerID: makeID, PrimaryWorkerID: driverID,
		FuelType: domaintypes.IFTAFuelTypeDiesel,
	})
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}

	return saved.ID, tenant.JurisdictionID, nil
}

// writeLeaveUnderWay writes leave that began three days ago and is still
// running, which the default window reads only because it overlaps.
func writeLeaveUnderWay(
	ctx context.Context,
	repo workerports.WorkerPTORepository,
	driver *worker.Worker,
	marked bool,
) (pulid.ID, error) {
	now := timeutils.NowUnix()
	status := worker.PTOStatusRequested
	if marked {
		status = worker.PTOStatusApproved
	}
	row, err := repo.Create(ctx, &worker.WorkerPTO{
		BusinessUnitID: driver.BusinessUnitID, OrganizationID: driver.OrganizationID,
		WorkerID: driver.ID, Status: status, Type: worker.PTOTypeVacation,
		StartDate: now - 3*timeutils.SecondsPerDay, EndDate: now + 2*timeutils.SecondsPerDay,
		Days: decimal.NewFromInt(5), Reason: "Family trip",
	})

	return created(row, err)
}

var (
	approved        = map[string]any{"status": []any{"Approved"}}
	outOfService    = map[string]any{"status": []any{"OutOfService"}}
	oldOutOfService = map[string]any{"filters": []any{map[string]any{
		"field": "status", "operator": "in", "values": []any{"OutOfService"},
	}}}
)

func equipmentStatusOf(marked bool) domaintypes.EquipmentStatus {
	if marked {
		return domaintypes.EquipmentStatusOOS
	}

	return domaintypes.EquipmentStatusAvailable
}

// equipmentParts writes what a tractor or trailer row needs before it: an
// equipment type, a manufacturer and a driver.
type equipmentParts struct {
	types         equipmenttypeports.EquipmentTypeRepository
	manufacturers equipmentmanufacturerports.EquipmentManufacturerRepository
	workers       workerports.WorkerRepository
}

func (p *equipmentParts) write(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
) (typeID, makeID, driverID pulid.ID, err error) {
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	kind, err := p.types.Create(ctx, &equipmenttype.EquipmentType{
		BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
		Code: label, Class: equipmenttype.ClassTractor,
		DeckType: equipmenttype.DeckTypeFlatbed,
	})
	if err != nil {
		return pulid.Nil, pulid.Nil, pulid.Nil, err
	}
	maker, err := p.manufacturers.Create(ctx, &equipmentmanufacturer.EquipmentManufacturer{
		BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
		Name: "Maker " + label,
	})
	if err != nil {
		return pulid.Nil, pulid.Nil, pulid.Nil, err
	}
	driver, err := p.workers.Create(ctx, testWorker(tenant, label, false, pulid.Nil))
	if err != nil {
		return pulid.Nil, pulid.Nil, pulid.Nil, err
	}

	return kind.ID, maker.ID, driver.ID, nil
}

func created[T interface{ GetID() pulid.ID }](row T, err error) (pulid.ID, error) {
	if err != nil {
		return pulid.Nil, err
	}

	return row.GetID(), nil
}

type referenceTenant struct {
	tenant *agentreadtest.Tenant
	ctx    context.Context
	rows   []string
	marked []string
}

func writeReferenceRows(
	t *testing.T,
	c *referenceCase,
	tenant *agentreadtest.Tenant,
	label string,
) *referenceTenant {
	t.Helper()

	out := &referenceTenant{
		tenant: tenant,
		ctx:    readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil),
	}
	if c.readsWith != nil {
		out.ctx = c.readsWith(out.ctx, tenant)
	}
	for idx := range referenceRowsPerTenant {
		marked := idx == referenceRowsPerTenant-1
		id, err := c.write(out.ctx, tenant, fmt.Sprintf("%s%d", label, idx), marked)
		require.NoError(t, err, c.tool)
		out.rows = append(out.rows, id.String())
		if marked {
			out.marked = append(out.marked, id.String())
		}
	}

	return out
}

func callTool(
	t *testing.T,
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *permissionports.RequestActor,
	args map[string]any,
) map[string]any {
	t.Helper()

	data, err := tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Timezone:       "America/Chicago",
		Params:         args,
		RunID:          pulid.MustNew("ar_"),
	})
	require.NoError(t, err, tool.Name())

	return asMap(t, data)
}

// walk reads every page of a list, by nextCursor or by nextOffset, whichever
// the tool returns.
func walk(
	t *testing.T,
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *permissionports.RequestActor,
	args map[string]any,
) (ids []string, pages int) {
	t.Helper()

	call := make(map[string]any, len(args)+2)
	for key, value := range args {
		call[key] = value
	}
	call["limit"] = referencePage
	for pages = 1; pages <= walkLimit; pages++ {
		page := callTool(t, ctx, tool, actor, call)
		ids = append(ids, itemIDs(page)...)
		if page["hasMore"] != true {
			return ids, pages
		}
		if cursor, ok := page["nextCursor"].(string); ok {
			call["after"] = cursor
			continue
		}
		call["offset"] = page["nextOffset"]
	}
	require.FailNow(t, tool.Name()+" never reached its last page")

	return ids, pages
}

func TestConvertedLists_ReadWhatTheRepositoryToolsReadUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	cases, services := convertedListCases(conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: services})

	env := &convertedListEnv{
		engine: engine,
		reader: reader,
		public: public,
		mine:   agentreadtest.SeedTenant(t, admin, "A"),
		theirs: agentreadtest.SeedTenant(t, admin, "B"),
	}
	for idx := range cases {
		c := &cases[idx]
		t.Run(c.tool, func(t *testing.T) { checkConvertedList(t, c, env) })
	}
}

// convertedListEnv is what a converted list is read against: the engine
// and readers at the default data access and at Public, and two tenants.
type convertedListEnv struct {
	engine permissionports.PermissionEngine
	reader agentports.GraphQLReader
	public agentports.GraphQLReader
	mine   *agentreadtest.Tenant
	theirs *agentreadtest.Tenant
	// attach adds to a tenant's context the loaders an operation's resolvers
	// read through, as the reader's caller does for a request.
	attach func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context
}

// within is the context a tenant's rows are read in.
func (e *convertedListEnv) within(ctx context.Context, tenant *agentreadtest.Tenant) context.Context {
	if e.attach == nil {
		return ctx
	}

	return e.attach(ctx, tenant)
}

// checkConvertedList holds one converted list to its tenant's rows under
// row-level security, to the repository tool's rows where it has one, to its
// typed filter, and to what a Public role may not see.
func checkConvertedList(t *testing.T, c *referenceCase, env *convertedListEnv) {
	t.Helper()

	converted, err := agentquerytoolservice.ConvertedListTool(c.tool, env.reader, env.engine)
	require.NoError(t, err)
	ours := writeReferenceRows(t, c, env.mine, "A")
	other := writeReferenceRows(t, c, env.theirs, "B")
	actor := env.mine.Actor

	byDefault := ours.rows
	if c.readsActiveByDefault {
		byDefault = slices.DeleteFunc(slices.Clone(ours.rows), func(id string) bool {
			return slices.Contains(ours.marked, id)
		})
	}

	t.Run("every page once, and nothing of another tenant", func(t *testing.T) {
		ids, pages := walk(t, ours.ctx, converted, actor, nil)
		assert.Greater(t, pages, 1, "the walk must cross a page")
		assert.ElementsMatch(t, byDefault, ids)
		for _, id := range other.rows {
			assert.NotContains(t, ids, id)
		}
	})

	t.Run("a typed filter narrows to what it names", func(t *testing.T) {
		converted, _ := walk(t, ours.ctx, converted, actor, c.marked)
		assert.ElementsMatch(t, ours.marked, converted)
	})

	if c.old != nil {
		t.Run("the same rows the repository tool paged through", func(t *testing.T) {
			converted, _ := walk(t, ours.ctx, converted, actor, nil)
			before, _ := walk(t, ours.ctx, c.old, actor, nil)
			assert.ElementsMatch(t, before, converted)
		})

		t.Run("a typed filter narrows as the repository tool's filter did", func(t *testing.T) {
			converted, _ := walk(t, ours.ctx, converted, actor, c.marked)
			before, _ := walk(t, ours.ctx, c.old, actor, c.oldMarked)
			assert.ElementsMatch(t, before, converted)
		})
	}

	t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
		hidden, err := agentquerytoolservice.ConvertedListTool(c.tool, env.public, &publicRole{})
		require.NoError(t, err)
		page := callTool(t, ours.ctx, hidden, actor, map[string]any{"limit": 50})
		withheld, _ := page["withheldByAccess"].([]any)
		require.NotEmpty(t, withheld)
		items, _ := page["items"].([]any)
		require.Len(t, items, len(byDefault))
		for _, item := range items {
			row, _ := item.(map[string]any)
			assert.Contains(t, row, "id")
			for _, field := range withheld {
				assert.NotContains(t, row, field)
			}
		}
	})
}

// A tractor's drivers are workers. A reader who may read tractors but not
// workers is told each driver's name, which the worker reference view shows
// to every reader of a record that links to a worker; the repository tool
// left assignedTo out, and the GraphQL tool now names the driver as the
// screen does.
func TestConvertedLists_ATractorsDriversShowTheirReferenceView(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	cases, services := convertedListCases(conn, admin)
	idx := slices.IndexFunc(cases, func(c referenceCase) bool {
		return c.tool == "equipment_list_tractors"
	})
	require.GreaterOrEqual(t, idx, 0)
	tenant := agentreadtest.SeedTenant(t, admin, "A")
	ours := writeReferenceRows(t, &cases[idx], tenant, "A")

	engine := &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceWorker.String()},
	}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	converted, err := agentquerytoolservice.ConvertedListTool("equipment_list_tractors", reader, engine)
	require.NoError(t, err)

	page := callTool(t, ours.ctx, converted, tenant.Actor, map[string]any{"limit": 50})
	items, _ := page["items"].([]any)
	require.Len(t, items, referenceRowsPerTenant)
	for _, item := range items {
		driver, ok := item.(map[string]any)["primaryWorker"].(map[string]any)
		require.True(t, ok, "the driver is named")
		assert.Contains(t, driver, "id")
		assert.Contains(t, driver, "wholeName", "a driver's name is in the worker reference view")
	}
	withheld, _ := page["withheldByAccess"].([]any)
	for _, path := range withheld {
		assert.NotContains(t, path, "primaryWorker.wholeName")
	}

	get, err := agentquerytoolservice.ConvertedGetTool("equipment_get_tractor", reader, engine)
	require.NoError(t, err)
	record := callTool(t, ours.ctx, get, tenant.Actor, map[string]any{"tractorId": ours.rows[0]})
	driver, ok := record["primaryWorker"].(map[string]any)
	require.True(t, ok, "the card names the driver")
	assert.Contains(t, driver, "firstName", "a driver's name is in the worker reference view")
	assert.Contains(t, driver, "lastName")
}

// testWorker is a driver of the tenant's, inactive when marked, managed by
// manager when one is named.
func testWorker(
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
	manager pulid.ID,
) *worker.Worker {
	now := timeutils.NowUnix()

	return &worker.Worker{
		BusinessUnitID: tenant.Actor.BusinessUnitID,
		OrganizationID: tenant.Actor.OrganizationID,
		StateID:        tenant.StateID,
		ManagerID:      manager,
		Status:         statusOf(marked),
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "Driver",
		LastName:       label,
		AddressLine1:   "1 Main",
		City:           "Dallas",
		PostalCode:     "75001",
		Gender:         worker.GenderMale,
		Profile: &worker.WorkerProfile{
			DOB:           now - 86400*365*30,
			LicenseNumber: "DL" + label + tenant.Actor.OrganizationID.String()[4:10],
			LicenseExpiry: now + 86400*365,
			HireDate:      now - 86400*365,
		},
	}
}

// A person whose worker grant reaches only their team reads, through the
// GraphQL list, exactly the workers the repository list showed them: the
// read_scope policy narrows the screen's statement as it narrowed the tool's.
func TestConvertedLists_WorkerListIsNarrowedToTheTeamOnBothPaths(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	_, services := convertedListCases(conn, admin)
	workers := workerrepository.New(workerrepository.Params{DB: conn, Logger: zap.NewNop()})
	tenant := agentreadtest.SeedTenant(t, admin, "A")
	tenantCtx := dbscope.WithTenant(t.Context(), tenant.Scope())

	team := make([]string, 0, 2)
	for idx := range 5 {
		manager := pulid.Nil
		if idx < 2 {
			manager = tenant.Actor.UserID
		}
		saved, err := workers.Create(readscope.With(tenantCtx, nil),
			testWorker(tenant, fmt.Sprintf("T%d", idx), false, manager))
		require.NoError(t, err)
		if idx < 2 {
			team = append(team, saved.ID.String())
		}
	}

	scoped := readscope.With(tenantCtx, func(context.Context) (readscope.Grants, error) {
		return readscope.Grants{"worker": {
			Scope: readscope.ScopeTeam, Team: []pulid.ID{tenant.Actor.UserID},
		}}, nil
	})
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	converted, err := agentquerytoolservice.ConvertedListTool("worker_list", reader, engine)
	require.NoError(t, err)
	old := agentquerytoolservice.NewListWorkersTool(workers, engine)

	before, _ := walk(t, scoped, old, tenant.Actor, nil)
	after, _ := walk(t, scoped, converted, tenant.Actor, nil)
	assert.ElementsMatch(t, team, before, "the repository list reads only the team")
	assert.ElementsMatch(t, before, after, "the GraphQL list reads what the repository list read")

	everyone, _ := walk(t, readscope.With(tenantCtx, nil), converted, tenant.Actor, nil)
	assert.Len(t, everyone, 5, "without a narrow grant every worker is read")
}
