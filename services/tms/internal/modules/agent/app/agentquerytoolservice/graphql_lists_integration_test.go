//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/equipmenttyperepository"
	"github.com/emoss08/trenova/internal/modules/equipment/adapters/postgres/fleetcoderepository"
	"github.com/emoss08/trenova/internal/modules/equipment/app/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/fleetcodeservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/m2msync"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierrepository"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationcategoryrepository"
	"github.com/emoss08/trenova/internal/modules/places/app/locationcategoryservice"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
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
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
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
	tool      string
	write     func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error)
	marked    map[string]any
	oldMarked map[string]any
	old       agentports.AgentQueryTool
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
// connection, and the resolver services the read executor reaches.
//
//nolint:funlen // one entry per converted list
func convertedListCases(conn *postgres.Connection) ([]referenceCase, agentread.ResolverParams) {
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
		DB: conn, Logger: logger, M2MSync: m2msync.NewSyncer(m2msync.SyncerParams{Logger: logger}),
	})
	locations := locationrepository.New(locationrepository.Params{DB: conn, Logger: logger})
	carriers := carrierrepository.New(carrierrepository.Params{DB: conn, Logger: logger})

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
		CustomerService:   customerservice.New(customerservice.Params{Logger: logger, Repo: customers}),
		LocationService:   locationservice.New(locationservice.Params{Logger: logger, Repo: locations}),
		CarrierService:    carrierservice.New(carrierservice.Params{Logger: logger, Repo: carriers}),
	}

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
					BusinessUnitID: bu, OrganizationID: org, Status: statusOf(marked), Code: label,
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
	}

	return cases, services
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
	cases, services := convertedListCases(conn)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: services})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")

	for idx := range cases {
		c := &cases[idx]
		t.Run(c.tool, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)
			ours := writeReferenceRows(t, c, mine, "A")
			other := writeReferenceRows(t, c, theirs, "B")
			actor := mine.Actor

			t.Run("every page once, and nothing of another tenant", func(t *testing.T) {
				ids, pages := walk(t, ours.ctx, converted, actor, nil)
				assert.Greater(t, pages, 1, "the walk must cross a page")
				assert.ElementsMatch(t, ours.rows, ids)
				for _, id := range other.rows {
					assert.NotContains(t, ids, id)
				}
			})

			t.Run("the same rows the repository tool paged through", func(t *testing.T) {
				converted, _ := walk(t, ours.ctx, converted, actor, nil)
				before, _ := walk(t, ours.ctx, c.old, actor, nil)
				assert.ElementsMatch(t, before, converted)
			})

			t.Run("a typed filter narrows as the repository tool's filter did", func(t *testing.T) {
				converted, _ := walk(t, ours.ctx, converted, actor, c.marked)
				before, _ := walk(t, ours.ctx, c.old, actor, c.oldMarked)
				assert.ElementsMatch(t, ours.marked, converted)
				assert.ElementsMatch(t, before, converted)
			})

			t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
				hidden, err := agentquerytoolservice.ConvertedListTool(c.tool, public, &publicRole{})
				require.NoError(t, err)
				page := callTool(t, ours.ctx, hidden, actor, map[string]any{"limit": 50})
				withheld, _ := page["withheldByAccess"].([]any)
				require.NotEmpty(t, withheld)
				items, _ := page["items"].([]any)
				require.Len(t, items, referenceRowsPerTenant)
				for _, item := range items {
					row, _ := item.(map[string]any)
					assert.Contains(t, row, "id")
					for _, field := range withheld {
						assert.NotContains(t, row, field)
					}
				}
			})
		})
	}
}
