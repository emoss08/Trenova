package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
The expand step.

A list answers "which ones", and then someone asks about one of them. These
return the whole record rather than a projection, because that is the question:
a list already decided what was worth summarising, and a model that has narrowed
to a single row needs the fields the summary left out.

Each names the list tool that yields its id. Without that, a model holding a
customer's name and no id has nowhere to go but guessing one.
*/
type getSpec struct {
	name      string
	entity    string
	summary   string
	resource  permission.Resource
	paramName string
	fetch     func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error)
}

type getTool struct {
	spec getSpec
}

func newGetTool(spec getSpec) serviceports.AgentQueryTool {
	return &getTool{spec: spec}
}

func (t *getTool) Name() string { return t.spec.name }

func (t *getTool) Description() string { return t.spec.summary }

func (t *getTool) PermissionResource() permission.Resource { return t.spec.resource }

func (t *getTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			t.spec.paramName: map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("The %s's id.", t.spec.entity),
			},
		},
		"required":             []string{t.spec.paramName},
		"additionalProperties": false,
	}
}

func (t *getTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
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

	return t.spec.fetch(ctx, id, pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	})
}

func newGetCustomerTool(repo repositories.CustomerRepository) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_customer",
		entity:   "customer",
		resource: permission.ResourceCustomer,
		summary: "Retrieve one customer by id, with their billing and email profiles. " +
			"Use list_customers first when you have a name or a code rather than an id.",
		paramName: "customerId",
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
	})
}

func newGetCarrierTool(repo repositories.CarrierRepository) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_carrier",
		entity:   "carrier",
		resource: permission.ResourceCarrier,
		summary: "Retrieve one carrier by id, including its authority, insurance and " +
			"compliance state. Use list_carriers first when you have a name or a code.",
		paramName: "carrierId",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetCarrierByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	})
}

func newGetTractorTool(repo repositories.TractorRepository) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_tractor",
		entity:   "tractor",
		resource: permission.ResourceTractor,
		summary: "Retrieve one tractor by id, with its equipment details, fleet and " +
			"assigned workers. Use list_tractors first when you have a unit code.",
		paramName: "tractorId",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetTractorByIDRequest{
				ID:                      id,
				TenantInfo:              tenant,
				TractorRelationIncludes: repositories.FullTractorRelationIncludes(),
			})
		},
	})
}

func newGetTrailerTool(repo repositories.TrailerRepository) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_trailer",
		entity:   "trailer",
		resource: permission.ResourceTrailer,
		summary: "Retrieve one trailer by id, with its equipment details and fleet. " +
			"Use list_trailers first when you have a unit code.",
		paramName: "trailerId",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetTrailerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	})
}

func newGetInvoiceTool(repo repositories.InvoiceRepository) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_invoice",
		entity:   "invoice",
		resource: permission.ResourceInvoice,
		summary: "Retrieve one invoice by id, with its line items and totals. Use " +
			"list_invoices first when you have a number or are looking for what is unpaid.",
		paramName: "invoiceId",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	})
}

func getCatalogSpecs() []getSpec {
	return []getSpec{
		specOfGet(newGetCustomerTool(nil)),
		specOfGet(newGetCarrierTool(nil)),
		specOfGet(newGetTractorTool(nil)),
		specOfGet(newGetTrailerTool(nil)),
		specOfGet(newGetInvoiceTool(nil)),
	}
}

func specOfGet(tool serviceports.AgentQueryTool) getSpec {
	return tool.(*getTool).spec //nolint:errcheck,forcetypeassert // constructed above
}
