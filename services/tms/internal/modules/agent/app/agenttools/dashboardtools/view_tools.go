package dashboardtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/tablequeryservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tableconfigurationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

type tableConfigWriter interface {
	Create(
		ctx context.Context,
		entity *tableconfiguration.TableConfiguration,
		actor *permissionports.RequestActor,
	) (*tableconfiguration.TableConfiguration, error)
}

type viewComposer interface {
	Compose(
		ctx context.Context,
		req *tablequeryservice.ComposeRequest,
	) (*tablequeryservice.ComposeResult, error)
}

type SaveTableViewTool struct {
	configs  tableConfigWriter
	composer viewComposer
	catalog  *filtercatalog.Catalog
}

func NewSaveTableViewTool(
	configs *tableconfigurationservice.Service,
	composer *tablequeryservice.Service,
	catalog *filtercatalog.Catalog,
) agenttoolports.AgentTool {
	return &SaveTableViewTool{configs: configs, composer: composer, catalog: catalog}
}

func (t *SaveTableViewTool) Name() string { return "report_save_table_view" }

func (t *SaveTableViewTool) SearchTerms() []string {
	return []string{"save view", "saved view", "pick again later"}
}

func (t *SaveTableViewTool) Description() string {
	return "Save a described view of a table so it appears in that table's view picker and " +
		"can be opened again without describing it. Use it when somebody says they want to " +
		"see something regularly — \"keep me a view of loads sitting unbilled over a week\" " +
		"— rather than for a one-off question, which report_compose_table_view already answers."
}

func (t *SaveTableViewTool) ParamSchema() map[string]any {
	resources := t.catalog.Resources()
	entities := make([]string, 0, len(resources))
	for _, resource := range resources {
		entities = append(entities, resource.Entity)
	}

	return map[string]any{
		"type":     "object",
		"required": []string{"entity", "name", "description"},
		"properties": map[string]any{
			"entity": agenttoolschema.Enum(
				"Which table the view is of.",
				agenttoolschema.Derived("tableView.entity", entities),
			),
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

func (t *SaveTableViewTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:                t.Name(),
		Kind:                agenttooltrust.ToolKindAction,
		Resource:            permission.ResourceTableConfiguration,
		Operation:           permission.OpCreate,
		Scope:               agenttooltrust.ToolScopeTenant,
		DefaultTier:         agent.TierPropose,
		MaxTier:             agent.TierAutoExecute,
		Egress:              []agent.EgressClass{agent.EgressPersonal, agent.EgressInternal},
		Classify:            classifyTableView,
		PersonalRunsUnasked: true,
		Effect:              agenttooltrust.ToolEffectChange,
		Reversible:          true,
		ReadsExternal:       agenttooltrust.ExternalReadNever,
		Rationale: "A private view is the caller's own picker entry; a shared one " +
			"appears for every colleague, and nothing leaves the organization.",
	}
}

func (t *SaveTableViewTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return t.validateArgs(params.Params)
}

func (t *SaveTableViewTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if toolparams.OptionalString(params, "name") == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Give the view a name")
	}
	if toolparams.OptionalString(params, "description") == "" {
		multiErr.Add("description", errortypes.ErrRequired, "Say what the view should show")
	}
	entity := toolparams.OptionalString(params, "entity")
	if _, ok := t.catalog.ByEntity(entity); entity != "" && !ok {
		multiErr.Add("entity", errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a table a view can be saved for", entity))
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *SaveTableViewTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	draft, err := t.Draft(params)
	if err != nil {
		return err
	}

	composed, err := t.composer.Compose(ctx, &tablequeryservice.ComposeRequest{
		TenantInfo: toolparams.TenantFrom(params),
		Actor:      params.Actor,
		Resource:   draft.resource.Resource,
		Prompt:     draft.prompt,
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
			draft.resource.Entity, unresolvedSummary(composed),
		)
	}

	_, err = t.configs.Create(ctx, draft.configuration(params, composed), params.Actor)

	return err
}

type tableViewDraft struct {
	resource   filtercatalog.Resource
	name       string
	prompt     string
	visibility tableconfiguration.Visibility
}

func (t *SaveTableViewTool) Draft(
	params *agenttoolports.ToolExecuteParams,
) (*tableViewDraft, error) {
	entity := toolparams.OptionalString(params.Params, "entity")
	resource, ok := t.catalog.ByEntity(entity)
	if !ok {
		return nil, fmt.Errorf("%q is not a table a view can be saved for", entity)
	}

	visibility := tableconfiguration.VisibilityPrivate
	if toolparams.OptionalBool(params.Params, "shared") {
		visibility = tableconfiguration.VisibilityPublic
	}

	return &tableViewDraft{
		resource:   resource,
		name:       toolparams.OptionalString(params.Params, "name"),
		prompt:     toolparams.OptionalString(params.Params, "description"),
		visibility: visibility,
	}, nil
}

func (d *tableViewDraft) configuration(
	params *agenttoolports.ToolExecuteParams,
	composed *tablequeryservice.ComposeResult,
) *tableconfiguration.TableConfiguration {
	configuration := &tableconfiguration.TableConfiguration{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		UserID:         params.Actor.UserID,
		Name:           d.name,
		Resource:       d.resource.Resource.String(),
		Visibility:     d.visibility,
		TableConfig:    &tableconfiguration.TableConfig{JoinOperator: "and"},
	}
	if composed != nil {
		configuration.Description = composed.Explanation
		configuration.TableConfig.FieldFilters = composed.FieldFilters
		configuration.TableConfig.Sort = composed.Sort
	}

	return configuration
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
