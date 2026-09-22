package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/core/services/tablequeryservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

/*
Keeping a view somebody described.

compose_table_view answers a question once. A view is the answer kept: it
appears in the table's view picker, it can be made the organization's default,
and the person who asked for it never has to describe it again.

The description is compiled here rather than accepted as filters, for the same
reason it is everywhere else — a saved view built from a field that does not
exist is a view that silently shows everything, and it would keep doing that
every time somebody opened it.
*/

type tableConfigWriter interface {
	Create(
		ctx context.Context,
		entity *tableconfiguration.TableConfiguration,
	) (*tableconfiguration.TableConfiguration, error)
}

type saveTableViewTool struct {
	configs  tableConfigWriter
	composer *tablequeryservice.Service
	catalog  *filtercatalog.Catalog
}

func newSaveTableViewTool(
	configs *tableconfigurationservice.Service,
	composer *tablequeryservice.Service,
	catalog *filtercatalog.Catalog,
) serviceports.AgentTool {
	return &saveTableViewTool{configs: configs, composer: composer, catalog: catalog}
}

func (t *saveTableViewTool) Name() string { return "save_table_view" }

func (t *saveTableViewTool) Description() string {
	return "Save a described view of a table so it appears in that table's view picker and " +
		"can be opened again without describing it. Use it when somebody says they want to " +
		"see something regularly — \"keep me a view of loads sitting unbilled over a week\" " +
		"— rather than for a one-off question, which compose_table_view already answers."
}

func (t *saveTableViewTool) ParamSchema() map[string]any {
	resources := t.catalog.Resources()
	entities := make([]string, 0, len(resources))
	for _, resource := range resources {
		entities = append(entities, resource.Entity)
	}

	return map[string]any{
		"type":     "object",
		"required": []string{"entity", "name", "description"},
		"properties": map[string]any{
			"entity": map[string]any{
				"type":        "string",
				"enum":        entities,
				"description": "Which table the view is of.",
			},
			"name": map[string]any{
				"type":        "string",
				"description": "What to call the view in the picker.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "What the view should show, in plain words.",
			},
			"shared": map[string]any{
				"type": "boolean",
				"description": "Whether everyone in the organization sees the view. " +
					"Defaults to false, which keeps it to the person who asked.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *saveTableViewTool) Reversible() bool { return true }

func (t *saveTableViewTool) PermissionResource() permission.Resource {
	return permission.ResourceTableConfiguration
}

func (t *saveTableViewTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *saveTableViewTool) RequiresIdempotencyKey() bool { return false }

func (t *saveTableViewTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *saveTableViewTool) Validate(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if optionalString(params, "name") == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Give the view a name")
	}
	if optionalString(params, "description") == "" {
		multiErr.Add("description", errortypes.ErrRequired, "Say what the view should show")
	}
	entity := optionalString(params, "entity")
	if _, ok := t.catalog.ByEntity(entity); entity != "" && !ok {
		multiErr.Add("entity", errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a table a view can be saved for", entity))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *saveTableViewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	entity := optionalString(params.Params, "entity")
	resource, ok := t.catalog.ByEntity(entity)
	if !ok {
		return fmt.Errorf("%q is not a table a view can be saved for", entity)
	}

	tenant := tenantFrom(params)

	composed, err := t.composer.Compose(ctx, &tablequeryservice.ComposeRequest{
		TenantInfo: tenant,
		Actor:      params.Actor,
		Resource:   resource.Resource,
		Prompt:     optionalString(params.Params, "description"),
	})
	if err != nil {
		return err
	}

	// A view that compiled to nothing is a view of everything, saved under a
	// name that promises otherwise. Refusing is the only honest outcome.
	if len(composed.FieldFilters) == 0 && composed.Query == "" {
		return fmt.Errorf(
			"none of that could be turned into filters on %s, so the view would show "+
				"everything under a name that says otherwise: %s",
			resource.Entity, unresolvedSummary(composed),
		)
	}

	visibility := tableconfiguration.VisibilityPrivate
	if optionalBool(params.Params, "shared") {
		visibility = tableconfiguration.VisibilityPublic
	}

	_, err = t.configs.Create(ctx, &tableconfiguration.TableConfiguration{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		UserID:         params.Actor.UserID,
		Name:           optionalString(params.Params, "name"),
		Description:    composed.Explanation,
		Resource:       resource.Resource.String(),
		Visibility:     visibility,
		TableConfig: &tableconfiguration.TableConfig{
			FieldFilters: composed.FieldFilters,
			Sort:         composed.Sort,
			JoinOperator: "and",
		},
	})

	return err
}

// unresolvedSummary says what could not be expressed, so a refusal names the
// thing to reword rather than only reporting that it failed.
func unresolvedSummary(composed *tablequeryservice.ComposeResult) string {
	if len(composed.Unresolved) == 0 {
		return "nothing in the description named a field on this table"
	}

	summary := ""
	for i, unresolved := range composed.Unresolved {
		if i > 0 {
			summary += "; "
		}
		summary += unresolved.Phrase + " (" + unresolved.Reason + ")"
	}

	return summary
}
