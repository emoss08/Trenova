import { DataTable } from "@/components/data-table/data-table";
import {
  AGENT_TOOL_RULE_LIST_KEY,
  agentToolRuleTableGraphQLConfig,
  type AgentToolRuleRow,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { getToolRuleColumns } from "./safety-columns";
import { ToolRulePanel } from "./safety-panels";
import { SAFETY_SUMMARY_STALE_MS } from "./safety-figures";

const NO_RESOURCES: readonly string[] = [];

/**
 * Every tool an agent can be given, with the rule the runtime holds it to.
 * The rules are declared beside each tool in code and paged, searched,
 * filtered and sorted on the server; opening a row reads the whole rule.
 */
export default function ToolRulesTable() {
  const t = useT();
  const summary = useQuery({
    ...queries.agentSafety.summary(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });
  const resources = summary.data?.resources ?? NO_RESOURCES;
  const columns = useMemo(() => getToolRuleColumns(t, resources), [resources, t]);

  return (
    <DataTable<AgentToolRuleRow>
      name="Tool Rule"
      queryKey={AGENT_TOOL_RULE_LIST_KEY}
      graphql={agentToolRuleTableGraphQLConfig}
      resource={Resource.AgentDefinition}
      columns={columns}
      TablePanel={ToolRulePanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ name: false, kind: false }}
    />
  );
}
