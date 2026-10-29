package shipmentrecordquerytools

import (
	"context"
	"maps"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/formulaquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/masterdataquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/partyquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmenttypeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationcategoryports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/commodityports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/shipmenttypeports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

const (
	maxRecordsAsked    = 10
	recordMatchLimit   = 3
	choiceLimit        = 25
	choiceTextRunes    = 80
	paramCustomer      = "customer"
	paramLocations     = "locations"
	paramCommodities   = "commodities"
	recordsAllFound    = "Every record is in Trenova: call shipment_create with these ids."
	recordsSomeMissing = "Propose what is missing now, in this turn, with customer_create, " +
		"location_create (category from locationCategories) and masterdata_create_commodity; do not " +
		"ask first. Then say the shipment follows once they are approved, and call " +
		"shipment_create with their new ids then."
	recordsUnreadable = "you may not read these records"
	recordFieldCity   = "city"
	recordFieldHazmat = "hazardous"
	recordFieldCode   = "code"
	recordFieldZIP    = "postalCode"
)

var compactColumns = []string{
	"id", recordFieldCode, "name", "description", "status", "address", recordFieldCity, "state",
	recordFieldZIP, "freightClass", recordFieldHazmat, "temperatureControlled", "type",
}

type ShipmentRecordsToolParams struct {
	fx.In

	Customers          customerports.CustomerRepository
	Locations          locationports.LocationRepository
	Commodities        commodityports.CommodityRepository
	ServiceTypes       servicetypeports.ServiceTypeRepository
	ShipmentTypes      shipmenttypeports.ShipmentTypeRepository
	EquipmentTypes     equipmenttypeports.EquipmentTypeRepository
	LocationCategories locationcategoryports.LocationCategoryRepository
	FormulaTemplates   formulatemplateports.FormulaTemplateRepository
	Permissions        permissionports.PermissionEngine
}

type findShipmentRecordsTool struct {
	customers   agenttoolports.AgentQueryTool
	locations   agenttoolports.AgentQueryTool
	commodities agenttoolports.AgentQueryTool
	choices     []recordChoice
	access      fieldgate.Access
}

type recordChoice struct {
	key    string
	tool   agenttoolports.AgentQueryTool
	params map[string]any
}

func NewFindShipmentRecordsTool(p *ShipmentRecordsToolParams) agenttoolports.AgentQueryTool {
	return &findShipmentRecordsTool{
		customers:   partyquerytools.NewListCustomersTool(p.Customers),
		locations:   partyquerytools.NewListLocationsTool(p.Locations),
		commodities: masterdataquerytools.NewListCommoditiesTool(p.Commodities),
		choices: []recordChoice{
			{
				key:  "serviceTypes",
				tool: masterdataquerytools.NewListServiceTypesTool(p.ServiceTypes),
			},
			{
				key:  "shipmentTypes",
				tool: masterdataquerytools.NewListShipmentTypesTool(p.ShipmentTypes),
			},
			{
				key:  "equipmentTypes",
				tool: masterdataquerytools.NewListEquipmentTypesTool(p.EquipmentTypes),
			},
			{
				key:  "locationCategories",
				tool: masterdataquerytools.NewListLocationCategoriesTool(p.LocationCategories),
			},
			{
				key:  "formulaTemplates",
				tool: formulaquerytools.NewListFormulaTemplatesTool(p.FormulaTemplates),
				params: map[string]any{"filters": []any{
					map[string]any{
						querytool.ParamField: "status",
						"operator":           "eq",
						"value":              querytool.ActiveStatus,
					},
					map[string]any{
						querytool.ParamField: "type",
						"operator":           "eq",
						"value":              "FreightCharge",
					},
				}},
			},
		},
		access: fieldgate.NewAccess(p.Permissions),
	}
}

func (t *findShipmentRecordsTool) Name() string { return agentpreset.ToolFindShipmentRecords }

func (t *findShipmentRecordsTool) Description() string {
	return "Match the customer, stop locations and commodities a new shipment names, in " +
		"one call, and say which are missing. Each comes with its closest matches, and the " +
		"answer lists the service types, shipment types, equipment types, location " +
		"categories and rating methods to choose from. Use it before shipment_create " +
		"instead of one list call per record."
}

func (t *findShipmentRecordsTool) ParamSchema() map[string]any {
	text := func(description string) map[string]any {
		return map[string]any{
			toolschema.KeyType:        toolschema.TypeString,
			toolschema.KeyDescription: description,
		}
	}

	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramCustomer: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyDescription: "The customer the load is billed to, as the person " +
					"or the document names it.",
				toolschema.KeyProperties: map[string]any{
					"name":          text("The customer's name."),
					recordFieldCode: text("Optional: its code, when given."),
				},
				toolschema.KeyAdditionalProperties: false,
			},
			paramLocations: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Every stop's place, in travel order.",
				toolschema.KeyMaxItems:    maxRecordsAsked,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeObject,
					toolschema.KeyProperties: map[string]any{
						"name":          text("The facility or company name."),
						"addressLine1":  text("Its street address."),
						recordFieldCity: text("Its city."),
						"state":         text("Its two-letter state."),
						recordFieldZIP:  text("Its ZIP code."),
					},
					toolschema.KeyAdditionalProperties: false,
				},
			},
			paramCommodities: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "What the load carries, by name.",
				toolschema.KeyMaxItems:    maxRecordsAsked,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
			},
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *findShipmentRecordsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipment,
		Effect:   agenttooltrust.ToolEffectLookup,
		Rationale: "Matches the records a new shipment names against the organization's own; " +
			"nothing changes and nothing is sent.",
	})
}

type shipmentRecords struct {
	Customer    *recordMatch   `json:"customer,omitempty"`
	Locations   []recordMatch  `json:"locations,omitempty"`
	Commodities []recordMatch  `json:"commodities,omitempty"`
	Missing     []string       `json:"missing,omitempty"`
	Choices     map[string]any `json:"choices"`
	Next        string         `json:"next"`
}

type recordMatch struct {
	Asked   string           `json:"asked"`
	Found   bool             `json:"found"`
	Matches []map[string]any `json:"matches,omitempty"`
	Note    string           `json:"note,omitempty"`
}

type recordLookup struct {
	tool    agenttoolports.AgentQueryTool
	kind    string
	asked   string
	queries []string
	slot    *recordMatch
}

func (t *findShipmentRecordsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	result := &shipmentRecords{Choices: make(map[string]any, len(t.choices))}
	lookups := t.lookups(params.Params, result)

	var mu sync.Mutex
	var wg sync.WaitGroup
	for idx := range lookups {
		lookup := &lookups[idx]
		wg.Go(func() { t.match(ctx, params, lookup) })
	}
	for _, choice := range t.choices {
		wg.Go(func() {
			rows := t.rows(ctx, params, choice.tool, choice.params)
			mu.Lock()
			result.Choices[choice.key] = rows
			mu.Unlock()
		})
	}
	wg.Wait()

	for idx := range lookups {
		if !lookups[idx].slot.Found {
			result.Missing = append(result.Missing, lookups[idx].kind+" "+lookups[idx].asked)
		}
	}
	result.Next = recordsAllFound
	if len(result.Missing) > 0 {
		result.Next = recordsSomeMissing
	}

	return result, nil
}

func (t *findShipmentRecordsTool) lookups(
	params map[string]any,
	result *shipmentRecords,
) []recordLookup {
	lookups := make([]recordLookup, 0, 1+2*maxRecordsAsked)

	if customer, ok := params[paramCustomer].(map[string]any); ok {
		name := querytool.OptionalString(customer, "name")
		code := querytool.OptionalString(customer, recordFieldCode)
		if name != "" || code != "" {
			result.Customer = &recordMatch{Asked: stringutils.FirstNonEmpty(name, code)}
			lookups = append(lookups, recordLookup{
				tool: t.customers, kind: paramCustomer, asked: result.Customer.Asked,
				queries: stringutils.FilterEmpty([]string{code, name}), slot: result.Customer,
			})
		}
	}

	places := asObjects(params[paramLocations])
	result.Locations = make([]recordMatch, len(places))
	for idx, place := range places {
		name := querytool.OptionalString(place, "name")
		street := querytool.OptionalString(place, "addressLine1")
		city := querytool.OptionalString(place, recordFieldCity)
		result.Locations[idx] = recordMatch{Asked: describePlace(name, street, city,
			querytool.OptionalString(place, "state"))}
		lookups = append(lookups, recordLookup{
			tool: t.locations, kind: "location", asked: result.Locations[idx].Asked,
			queries: stringutils.FilterEmpty([]string{name, street}), slot: &result.Locations[idx],
		})
	}

	names := querytool.OptionalStrings(params, paramCommodities)
	names = names[:min(len(names), maxRecordsAsked)]
	result.Commodities = make([]recordMatch, len(names))
	for idx, name := range names {
		result.Commodities[idx] = recordMatch{Asked: name}
		lookups = append(lookups, recordLookup{
			tool: t.commodities, kind: "commodity", asked: name,
			queries: []string{name}, slot: &result.Commodities[idx],
		})
	}

	return lookups
}

func (t *findShipmentRecordsTool) match(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	lookup *recordLookup,
) {
	if !t.access.MayRead(ctx, params, lookup.tool.Policy().Resource) {
		lookup.slot.Note = recordsUnreadable
		return
	}
	for _, query := range lookup.queries {
		rows := t.rows(ctx, params, lookup.tool, map[string]any{
			querytool.ParamQuery: query, querytool.ParamLimit: recordMatchLimit,
		})
		if len(rows) > 0 {
			lookup.slot.Found = true
			lookup.slot.Matches = rows
			return
		}
	}
}

func (t *findShipmentRecordsTool) rows(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	tool agenttoolports.AgentQueryTool,
	args map[string]any,
) []map[string]any {
	if !t.access.MayRead(ctx, params, tool.Policy().Resource) {
		return nil
	}

	call := *params
	call.Params = map[string]any{querytool.ParamLimit: choiceLimit}
	maps.Copy(call.Params, args)
	answer, err := tool.Query(ctx, &call)
	if err != nil {
		return nil
	}

	return compactRows(answer)
}

func compactRows(answer any) []map[string]any {
	var outcome struct {
		Items []map[string]any `json:"items"`
	}
	if err := jsonutils.Convert(answer, &outcome); err != nil {
		return nil
	}

	rows := make([]map[string]any, 0, len(outcome.Items))
	for _, item := range outcome.Items {
		row := make(map[string]any, len(compactColumns))
		for _, column := range compactColumns {
			value, ok := item[column]
			if !ok || value == nil || value == "" {
				continue
			}
			if text, isText := value.(string); isText {
				value = stringutils.Ellipsize(text, choiceTextRunes)
			}
			row[column] = value
		}
		rows = append(rows, row)
	}

	return rows
}

func asObjects(raw any) []map[string]any {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	objects := make([]map[string]any, 0, min(len(list), maxRecordsAsked))
	for _, entry := range list {
		if len(objects) == maxRecordsAsked {
			break
		}
		if object, isObject := entry.(map[string]any); isObject {
			objects = append(objects, object)
		}
	}

	return objects
}

func describePlace(name, street, city, state string) string {
	return strings.Join(stringutils.FilterEmpty([]string{name, street, city + " " + state}), ", ")
}
