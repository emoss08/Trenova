package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
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
}

type getTool struct {
	spec   getSpec
	access fieldAccess
}

func newGetTool(
	spec *getSpec,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getTool{spec: *spec, access: newFieldAccess(permissions)}
}

func (t *getTool) Name() string { return t.spec.name }

func (t *getTool) Description() string { return t.spec.summary }

func (t *getTool) SearchTerms() []string { return t.spec.searchTerms }

func (t *getTool) Policy() serviceports.ToolPolicy {
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
	params *serviceports.QueryToolParams,
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

func newGetCustomerTool(
	repo repositories.CustomerRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:        "get_customer",
		entity:      labelCustomer,
		searchTerms: []string{"customer details", "customer profile"},
		resource:    permission.ResourceCustomer,
		summary: "Retrieve one customer by id, with their billing and email profiles. " +
			"Use list_customers first when you have a name or a code rather than an id.",
		paramName: "customerId",
		idSource:  "from list_customers",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
				CustomerFilterOptions: repositories.CustomerFilterOptions{
					IncludeState:          true,
					IncludeBillingProfile: true,
				},
			})
		},
	}, permissions)
}

func newGetCarrierTool(
	repo repositories.CarrierRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "get_carrier",
		entity:   "carrier",
		resource: permission.ResourceCarrier,
		summary: "Retrieve one carrier by id, including its authority, insurance and " +
			"compliance state. Use list_carriers first when you have a name or a code.",
		paramName: "carrierId",
		idSource:  "from list_carriers",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetCarrierByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	}, permissions)
}

func newGetTractorTool(
	repo repositories.TractorRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "get_tractor",
		entity:   "tractor",
		resource: permission.ResourceTractor,
		summary: "Retrieve one tractor by id, with its equipment details, fleet and " +
			"assigned workers. Use list_tractors first when you have a unit code.",
		paramName: "tractorId",
		idSource:  "from list_tractors",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetTractorByIDRequest{
				ID:                      id,
				TenantInfo:              tenant,
				TractorRelationIncludes: repositories.FullTractorRelationIncludes(),
			})
		},
	}, permissions)
}

func newGetTrailerTool(
	repo repositories.TrailerRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "get_trailer",
		entity:   "trailer",
		resource: permission.ResourceTrailer,
		summary: "Retrieve one trailer by id, with its equipment details and fleet. " +
			"Use list_trailers first when you have a unit code.",
		paramName: "trailerId",
		idSource:  "from list_trailers",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetTrailerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	}, permissions)
}

func getCatalogSpecs() []getSpec {
	return []getSpec{
		specOfGet(newGetCustomerTool(nil, nil)),
		specOfGet(newGetCarrierTool(nil, nil)),
		specOfGet(newGetTractorTool(nil, nil)),
		specOfGet(newGetTrailerTool(nil, nil)),
		specOfGet(newGetDetentionOccurrenceTool(nil, nil, nil)),
		specOfGet(newGetServiceFailureTool(nil, nil)),
	}
}

func specOfGet(tool serviceports.AgentQueryTool) getSpec {
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
