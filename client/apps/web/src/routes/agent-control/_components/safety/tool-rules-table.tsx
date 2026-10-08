import { DataTable } from "@/components/data-table/data-table";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import {
  AGENT_TOOL_RULE_LIST_KEY,
  agentToolRuleTableGraphQLConfig,
  type AgentToolRuleRow,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { createContext, useContext, useMemo } from "react";
import { getToolRuleColumns, holdersOf, type ToolHolding } from "./safety-columns";
import { SAFETY_SUMMARY_STALE_MS } from "./safety-figures";
import { ToolSheet } from "./tool-sheet";

const NO_RESOURCES: readonly string[] = [];
const NO_HOLDING: ToolHolding = { byTool: new Map(), agents: new Map() };

const HoldingContext = createContext<ToolHolding>(NO_HOLDING);

/** Which agents hold each tool, with the agents themselves, read once for the page. */
export function useToolHolding(): ToolHolding {
  const holders = useQuery({
    ...queries.agentSafety.holders(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });
  const agents = useQuery(queries.assistant.agents(false));

  return useMemo(
    () => ({
      byTool: holders.data ?? new Map(),
      agents: new Map(
        (agents.data ?? []).map((agent: AgentDefinitionRow) => [agent.id, agent] as const),
      ),
    }),
    [agents.data, holders.data],
  );
}

/**
 * Every tool an agent can be given, with the rule the runtime holds it to and the agents
 * that hold it. The rules are declared beside each tool in code and paged, searched,
 * filtered and sorted on the server; opening a row reads the whole rule.
 */
export default function ToolRulesTable() {
  const t = useT();
  const summary = useQuery({
    ...queries.agentSafety.summary(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });
  const holding = useToolHolding();
  const resources = summary.data?.resources ?? NO_RESOURCES;
  const columns = useMemo(() => getToolRuleColumns(t, resources, holding), [holding, resources, t]);

  return (
    <HoldingContext.Provider value={holding}>
      <DataTable<AgentToolRuleRow>
        name="Tool Rule"
        emptyTitle={t("No tool rules match")}
        queryKey={AGENT_TOOL_RULE_LIST_KEY}
        graphql={agentToolRuleTableGraphQLConfig}
        resource={Resource.AgentDefinition}
        columns={columns}
        TablePanel={ToolRulePanel}
        enableCreateAction={false}
        enableReadOnlyPanel
        initialColumnVisibility={{ name: false, kind: false, needs: false, readsExternal: false }}
      />
    </HoldingContext.Provider>
  );
}

function ToolRulePanel({ open, onOpenChange, row }: DataTablePanelProps<AgentToolRuleRow>) {
  const holding = useContext(HoldingContext);

  return (
    <ToolSheet
      policy={open ? row : null}
      holders={row ? holdersOf(holding, row.name) : []}
      onClose={() => onOpenChange(false)}
    />
  );
}
