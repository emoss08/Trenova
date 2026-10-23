package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
The expand step.

A list answers "which ones", and then someone asks about one of them. These
return the whole record rather than a projection, because that is the question:
a list already decided what was worth summarising, and a model that has narrowed
to a single row needs the fields the summary left out.

Each names the list tool that yields its id, in the summary and again in
idSource on the parameter itself. Without that, a model holding a customer's
name and no id has nowhere to go but guessing one.
*/
type getSpec struct {
	name        string
	entity      string
	summary     string
	searchTerms []string
	resource    permission.Resource
	kinds       []permission.RecordKind
	paramName   string
	idSource    string
	reads       agent.ExternalRead
	source      agent.TaintSource
	rationale   string
	fetch       func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error)
	// batch, on a get that reads through GraphQL, makes its parameter a list
	// of at most this many ids, each read on its own.
	batch int
	// shape applies to a record read through GraphQL what the card's
	// resolver does not, and whether it is one of several read together.
	shape func(record map[string]any, several bool)
}

type getTool struct {
	spec   getSpec
	access fieldAccess
}

func newGetTool(
	spec *getSpec,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getTool{spec: *spec, access: newFieldAccess(permissions)}
}

func (t *getTool) Name() string { return t.spec.name }

func (t *getTool) Description() string { return t.spec.summary }

func (t *getTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *getTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource:  t.spec.resource,
		reads:     t.spec.reads,
		source:    t.spec.source,
		rationale: t.spec.rationale,
	})
}

// ParamSchema marks the id with the tool's resource when that resource is one
// kind of record with one prefix: the record it reads is the resource it is
// permitted under. A tool reading one kind of a resource that covers several,
// a detention occurrence under detention, names that kind instead, so its id
// is still held to a prefix rather than checked for shape alone.
func (t *getTool) ParamSchema() map[string]any {
	property := agenttoolschema.IDText(t.spec.idDescription())
	if len(t.spec.kinds) > 0 {
		property = agenttoolschema.KindID(t.spec.idDescription(), t.spec.kinds...)
	} else if _, typed := t.spec.resource.IDPrefix(); typed {
		property = agenttoolschema.RecordIDText(t.spec.resource, t.spec.idDescription())
	}

	return idSchema(t.spec.paramName, property)
}

func (s getSpec) idDescription() string {
	if s.idSource == "" {
		return fmt.Sprintf("The %s's id.", s.entity)
	}

	return fmt.Sprintf("The %s's id, %s.", s.entity, s.idSource)
}

func (t *getTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	// Parsing rather than passing the string through is what turns a
	// paraphrased id — or a pro number pasted where an id belongs — into a
	// plain error, instead of an empty result the model reports as a missing
	// record.
	id, err := requirePulid(params.Params, t.spec.paramName)
	if err != nil {
		return nil, err
	}

	entity, err := t.spec.fetch(ctx, id, pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	})
	if err != nil {
		return nil, err
	}

	return t.access.redactor(ctx, params).withhold(entity)
}

// tractorGetSpec reads one tractor through the Desk's tractor card.
func tractorGetSpec() getSpec {
	return getSpec{
		name:     "equipment_get_tractor",
		entity:   "tractor",
		resource: permission.ResourceTractor,
		summary: "Retrieve one tractor by id, with its equipment details, fleet, lease and " +
			"assigned drivers. Use equipment_list_tractors first when you have a unit code.",
		paramName: "tractorId",
		idSource:  "from equipment_list_tractors",
	}
}

// trailerGetSpec reads one trailer through the Desk's trailer card.
func trailerGetSpec() getSpec {
	return getSpec{
		name:     "equipment_get_trailer",
		entity:   "trailer",
		resource: permission.ResourceTrailer,
		summary: "Retrieve one trailer by id, with its equipment details, fleet and lease. " +
			"Use equipment_list_trailers first when you have a unit code.",
		paramName: "trailerId",
		idSource:  "from equipment_list_trailers",
	}
}

const (
	locationEntity  = "location"
	locationIDParam = "locationId"
)

// customerGetSpec reads one customer through the Desk's customer card.
func customerGetSpec() getSpec {
	return getSpec{
		name:        "customer_get",
		entity:      labelCustomer,
		searchTerms: []string{"customer details", "customer profile"},
		resource:    permission.ResourceCustomer,
		summary: "Retrieve one customer by id, with their address, authority numbers and " +
			"billing profile: terms, cycle, delivery, credit and the documents each invoice " +
			"needs. Use customer_list first when you have a name or a code rather than an id.",
		paramName: paramCustomerID,
		idSource:  "from customer_list",
	}
}

// carrierGetSpec reads one carrier through the Desk's carrier card.
func carrierGetSpec() getSpec {
	return getSpec{
		name:     "carrier_get",
		entity:   labelCarrier,
		resource: permission.ResourceCarrier,
		summary: "Retrieve one carrier by id, including its authority, insurance and " +
			"compliance state. Use carrier_list first when you have a name or a code.",
		paramName: paramCarrierID,
		idSource:  "from carrier_list",
	}
}

// locationGetSpec reads one location through the Desk's location card.
func locationGetSpec() getSpec {
	return getSpec{
		name:     "location_get",
		entity:   locationEntity,
		resource: permission.ResourceLocation,
		summary: "Retrieve one location by id, with its address, coordinates, geofence, " +
			"timezone and category (appointments, parking, overnight). Use location_list " +
			"first when you have a name, a code or a city.",
		paramName: locationIDParam,
		idSource:  "from location_list",
	}
}

// graphQLGetSpecs are every get tool that reads through GraphQL by a spec
// alone.
func graphQLGetSpecs() []getSpec {
	return []getSpec{
		customerGetSpec(),
		carrierGetSpec(),
		locationGetSpec(),
		carrierIntelEventGetSpec(),
		tractorGetSpec(),
		trailerGetSpec(),
		invoiceGetSpec(),
		invoiceAdjustmentGetSpec(),
	}
}

// graphQLGetProviders registers each spec as its own get tool.
func graphQLGetProviders() []any {
	specs := graphQLGetSpecs()
	providers := make([]any, 0, len(specs))
	for idx := range specs {
		spec := &specs[idx]
		providers = append(providers, func(
			reader agentports.GraphQLReader,
			permissions permissionports.PermissionEngine,
		) (agentports.AgentQueryTool, error) {
			return newGraphQLGetTool(spec, reader, permissions)
		})
	}

	return providers
}

func getCatalogSpecs() []getSpec {
	return []getSpec{
		customerGetSpec(),
		carrierGetSpec(),
		tractorGetSpec(),
		trailerGetSpec(),
		invoiceGetSpec(),
		invoiceAdjustmentGetSpec(),
		locationGetSpec(),
		specOfGet(newGetDetentionOccurrenceTool(nil, nil, nil)),
		specOfGet(newGetServiceFailureTool(nil, nil)),
		specOfGet(newGetFormulaTemplateTool(nil, nil)),
	}
}

func specOfGet(tool agentports.AgentQueryTool) getSpec {
	return tool.(*getTool).spec //nolint:errcheck,forcetypeassert // constructed above
}

func idSchema(param string, property map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{param: property},
		"required":             []string{param},
		"additionalProperties": false,
	}
}
