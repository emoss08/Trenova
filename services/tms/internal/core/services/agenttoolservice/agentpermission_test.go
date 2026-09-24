package agenttoolservice

import (
	"reflect"
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/testutil/providertest"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/require"
)

/*
A tool a scheduled desk is told to use, whose permission no agent may hold,
does nothing — and says so only when the desk is woken for real work.

agentAllowedPermissions is the whole permission model for an agent principal:
there is no role behind it, so a missing entry is a flat refusal at the moment
the run tries to act. It is also the ceiling on what any agent can ever do, so
it must not be wider than the desks need either. Several entries were missing
against the templates already shipped — the load monitor is told to evaluate
service failures and resolve them, and could do neither.

The rule is the desks, not the catalog: a tool a background template lists
must be one an agent may call, and a tool only a person's assistant offers
need not be. That is what keeps update_shipment closed to an agent while
create_shipment, which the intake desk runs on, is open.
*/

type toolPermission struct {
	resource  permission.Resource
	operation permission.Operation
	// selfScoped tools act for the person in the conversation, and the
	// runtime withholds them from any run nobody is watching.
	selfScoped bool
}

// toolPermissions maps each tool's wire name to what it needs, read from the
// policy every registered tool declares.
func toolPermissions(t *testing.T) map[string]toolPermission {
	t.Helper()

	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(
			agentquerytoolservice.FilterCatalog(),
		),
	}

	out := map[string]toolPermission{}
	for _, provider := range append(agentquerytoolservice.ToolProviders(), ToolProviders()...) {
		declarer, ok := providertest.Build(t, provider, supplied).(serviceports.ToolPolicyDeclarer)
		require.Truef(t, ok, "%T builds a tool with no policy", provider)

		policy := declarer.Policy()
		out[policy.Name] = toolPermission{
			resource:   policy.Resource,
			operation:  policy.Operation,
			selfScoped: policy.Scope == agent.ToolScopeSelf,
		}
	}
	require.NotEmpty(t, out)

	return out
}

// backgroundTools are the tools the templates that run without a person
// hold — their starters and the core tools every agent carries — which is the
// set an agent principal has to be able to call.
func backgroundTools(t *testing.T) map[string][]agentdefinition.Template {
	t.Helper()

	out := map[string][]agentdefinition.Template{}
	for _, template := range agentdefinition.AllTemplates() {
		if template.StarterTrigger() == agentdefinition.TriggerChat {
			continue
		}
		for _, tool := range template.StarterTools() {
			out[tool] = append(out[tool], template)
		}
		for _, tool := range agentdefinition.CoreTools() {
			out[tool] = append(out[tool], template)
		}
	}
	require.NotEmpty(t, out)

	return out
}

func TestEveryToolADeskRunsOnIsOneAnAgentMayCall(t *testing.T) {
	t.Parallel()

	permissions := toolPermissions(t)
	for name, templates := range backgroundTools(t) {
		// A core tool that acts for the person in the conversation is
		// withheld from every run nobody is watching, so no desk runs on it.
		if permissions[name].selfScoped {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			needed, ok := permissions[name]
			require.Truef(t, ok,
				"%v run on %s and nothing declares what it needs; a tool the scan "+
					"cannot read is one this check silently passes",
				templates, name,
			)

			require.Truef(t, permission.IsAgentAllowed(needed.resource, needed.operation),
				"%s needs %s on %s and no agent may hold it, yet %v run on it",
				name, needed.operation, needed.resource, templates,
			)
		})
	}
}

// The allow-list is the ceiling on what an agent can ever do, so an entry no
// tool claims is capability granted to nothing — and capability granted to
// nothing is capability the next tool inherits by accident.
//
// The claim is any registered tool rather than the shipped starters, because
// an organization builds its own desks from the same catalog: a tool nobody
// has put on an agent yet is still one they may.
func TestTheAllowListGrantsNothingNoToolClaims(t *testing.T) {
	t.Parallel()

	// The runtime writes these itself, outside any tool.
	usedOutsideTools := map[permission.Resource]map[permission.Operation]string{
		permission.ResourceAgentRun: {
			permission.OpRead:   "the runtime opens and reads its own runs",
			permission.OpCreate: "the runtime opens its own runs",
		},
		permission.ResourceAgentProposal: {
			permission.OpRead:   "the loop reads back what it proposed",
			permission.OpCreate: "the loop raises proposals, not a tool",
		},
		permission.ResourceAgentException: {
			permission.OpRead: "the loop reads its own exceptions",
		},
		permission.ResourceAgentMemory: {
			permission.OpUpdate: "remember overwrites what it wrote before",
		},
		permission.ResourceBillingQueue: {
			permission.OpRead: "agentsubjectservice loads the item a run is about",
		},
		permission.ResourceBankReceiptWorkItem: {
			permission.OpRead: "agentsubjectservice loads the work item beside the receipt",
		},
		permission.ResourceShipment: {
			permission.OpRead: "agentsubjectservice loads the shipment a run is about",
		},
		permission.ResourceShipmentMove: {
			permission.OpRead: "agentsubjectservice loads the move a run is about",
		},
		permission.ResourceWorker: {
			permission.OpRead: "agentsubjectservice loads the driver a run is about",
		},
		permission.ResourceDocument: {
			permission.OpRead: "agentsubjectservice loads the document a run is about",
		},
		permission.ResourceInsight: {
			permission.OpRead: "agentsubjectservice loads the finding a run is about",
		},
		permission.ResourceBankReceipt: {
			permission.OpRead: "agentsubjectservice loads the receipt a run is about",
		},
	}

	needed := map[permission.Resource]map[permission.Operation]struct{}{}
	for _, tool := range toolPermissions(t) {
		if needed[tool.resource] == nil {
			needed[tool.resource] = map[permission.Operation]struct{}{}
		}
		needed[tool.resource][tool.operation] = struct{}{}
	}

	unclaimed := make([]string, 0)
	for resource, ops := range permission.AgentAllowedPermissions() {
		for _, operation := range ops {
			if _, ok := needed[resource][operation]; ok {
				continue
			}
			if _, ok := usedOutsideTools[resource][operation]; ok {
				continue
			}
			unclaimed = append(unclaimed, string(operation)+" on "+string(resource))
		}
	}
	sort.Strings(unclaimed)

	require.Emptyf(t, unclaimed,
		"an agent may do these, and no tool claims them; remove the entries, or list "+
			"each in usedOutsideTools with what writes it: %v", unclaimed,
	)
}
