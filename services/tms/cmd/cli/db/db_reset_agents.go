package db

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory/agentmemorydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust/agenttooltrustdb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var dbResetAgentsCmd = &cobra.Command{
	Use:   "reset-agents",
	Short: "Clear what still names removed agent tools and restore the system agents",
	Long: `Agents became scoped policies and the per-entity read tools became records_query and
record_get. The migration switched every agent off with a scope that denies everything; this
command finishes the move:

  - system agents (the universal agent, the default desks, page assistants and the
    system automations) get their template's scope and pins back, and the universal agent,
    the desks and the page assistants are switched on again;
  - agents people built stay switched off as an archive, to be set up again on the new model;
  - memories, earned trust and tool rule overrides that name a tool that no longer exists
    are deleted, since none of them can apply again.

Examples:
  trenova db reset-agents --dry-run   # Count what would change
  trenova db reset-agents             # Apply`,
	RunE: runResetAgents,
}

const resetAgentsReason = "move every organization's agents onto scoped policies"

func init() {
	DbCmd.AddCommand(dbResetAgentsCmd)
}

func runResetAgents(_ *cobra.Command, _ []string) error {
	ctx := dbscope.WithSystem(context.Background(), resetAgentsReason)
	db, err := openDB(ctx)
	if err != nil {
		return fmt.Errorf("open the database: %w", err)
	}
	defer db.Close()

	current, err := currentToolNames()
	if err != nil {
		return err
	}

	restored, archived, err := restoreSystemAgents(ctx, db)
	if err != nil {
		return err
	}
	color.Cyan("→ %d system agent(s) restored to their template's scope; %d built agent(s) left switched off",
		restored, archived)

	memories, trust, overrides, err := clearRemovedTools(ctx, db, current)
	if err != nil {
		return err
	}
	verb := "Deleted"
	if dryRun {
		verb = "Would delete"
	}
	color.Cyan("→ %s %d memorie(s), %d earned-trust row(s) and %d rule override(s) naming removed tools",
		verb, memories, trust, overrides)
	color.Green("✓ Done. Agents people built must be set up again on the new model.")

	return nil
}

func currentToolNames() ([]string, error) {
	tools, err := registered.Build()
	if err != nil {
		return nil, fmt.Errorf("build the tool registry: %w", err)
	}
	names := make([]string, 0, len(tools.Queries)+len(tools.Actions)+len(tools.Runtime))
	for _, tool := range tools.Queries {
		if !agentcapability.IsBacking(tool.Name()) {
			names = append(names, tool.Name())
		}
	}
	for _, tool := range tools.Actions {
		names = append(names, tool.Name())
	}
	for idx := range tools.Runtime {
		names = append(names, tools.Runtime[idx].Name)
	}

	return names, nil
}

func restoreSystemAgents(ctx context.Context, db *keel.DB) (restored, archived int, err error) {
	d := agentdefinitiondb.Definitions
	definitions, err := d.AllTenants().All(ctx, db)
	if err != nil {
		return 0, 0, fmt.Errorf("list agent definitions: %w", err)
	}
	switchOn := make([]string, 0, 8)
	for _, template := range agentpreset.DefaultDesks() {
		switchOn = append(switchOn, agentpreset.DeskKey(template))
	}
	switchOn = append(switchOn, agentdefinition.PageAgentKeys()...)
	for _, definition := range definitions {
		if !definition.IsSystem() {
			archived++
			continue
		}
		definition.Scope = agentpreset.ScopeFor(definition.Template)
		definition.PinnedTools = agentpreset.PinnedFor(definition.Template)
		if slices.Contains(switchOn, definition.SystemKey) {
			definition.Enabled = true
			definition.DisabledAt = nil
			definition.DisabledByID = nil
		}
		restored++
		if dryRun {
			continue
		}
		if err = d.Save(ctx, db, definition, d.Scope, d.PinnedTools, d.Enabled, d.DisabledAt, d.DisabledByID); err != nil {
			return restored, archived, fmt.Errorf("restore agent %s: %w", definition.Name, err)
		}
	}

	return restored, archived, nil
}

func clearRemovedTools(
	ctx context.Context,
	db *keel.DB,
	current []string,
) (memories, trust, overrides int64, err error) {
	m := agentmemorydb.Memories
	t := agenttooltrustdb.ToolTrusts
	o := agenttooltrustdb.ToolRuleOverrides
	staleMemory := keel.And(m.ToolName.IsNotNull(), m.ToolName.Ne(""), m.ToolName.NotIn(current...))
	staleTrust := t.ToolName.NotIn(current...)
	staleOverride := o.ToolName.NotIn(current...)
	if dryRun {
		if memories, err = m.AllTenants().Where(staleMemory).Count(ctx, db); err != nil {
			return 0, 0, 0, err
		}
		if trust, err = t.AllTenants().Where(staleTrust).Count(ctx, db); err != nil {
			return 0, 0, 0, err
		}
		overrides, err = o.AllTenants().Where(staleOverride).Count(ctx, db)

		return memories, trust, overrides, err
	}
	if memories, err = m.AllTenants().Where(staleMemory).Delete(ctx, db); err != nil {
		return 0, 0, 0, fmt.Errorf("delete memories naming removed tools: %w", err)
	}
	if trust, err = t.AllTenants().Where(staleTrust).Delete(ctx, db); err != nil {
		return 0, 0, 0, fmt.Errorf("delete earned trust for removed tools: %w", err)
	}
	if overrides, err = o.AllTenants().Where(staleOverride).Delete(ctx, db); err != nil {
		return 0, 0, 0, fmt.Errorf("delete rule overrides for removed tools: %w", err)
	}

	return memories, trust, overrides, nil
}
