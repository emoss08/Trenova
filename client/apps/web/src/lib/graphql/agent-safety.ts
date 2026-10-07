import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentSafetyDocument,
  AgentSafetyHeaderFieldsFragmentDoc,
  AgentSafetySummaryDocument,
  AgentToolPolicyFieldsFragmentDoc,
  DataTablePageInfoFieldsFragmentDoc,
  AgentToolRuleTableDocument,
  AgentToolSafetyTableDocument,
  type AgentAutonomyAnswer,
  type AgentEgressClass,
  type AgentExternalRead,
  type AgentReachWarningKind,
  type AgentSafetyHeaderFieldsFragment,
  type AgentSafetySummaryQuery,
  type AgentToolAutonomyFieldsFragment,
  type AgentToolKind,
  type AgentToolPolicyFieldsFragment,
  type AgentToolRuleTableQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export type AgentToolPolicy = AgentToolPolicyFieldsFragment;
export type AgentToolAutonomy = AgentToolAutonomyFieldsFragment;
export type AgentSafetySummary = AgentSafetySummaryQuery["agentSafetySummary"];
export type AgentSafetyHeader = AgentSafetyHeaderFieldsFragment;

export type {
  AgentAutonomyAnswer,
  AgentEgressClass,
  AgentExternalRead,
  AgentReachWarningKind,
  AgentToolKind,
};

type RequestOptions = { signal?: AbortSignal };

export const AGENT_TOOL_RULE_LIST_KEY = "agent-tool-rule-list";
export const AGENT_TOOL_SAFETY_LIST_KEY = "agent-tool-safety-list";

/** Every tool's rule, paged, searched, filtered and sorted on the server. */
export const agentToolRuleTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentToolRuleTableDocument,
  operationName: "AgentToolRuleTable",
  connectionKey: "agentToolRuleConnection",
});

export type AgentToolRuleRow = DataTableConfigRow<typeof agentToolRuleTableGraphQLConfig>;

/**
 * What the agents being compared make of each tool they hold. The agents are
 * part of the query rather than a filter a person can clear, because the
 * server assesses only the agents it is named.
 */
export function createAgentToolSafetyTableGraphQLConfig(agentIds: readonly string[]) {
  return defineDataTableGraphQLConfig({
    document: AgentToolSafetyTableDocument,
    operationName: "AgentToolSafetyTable",
    connectionKey: "agentToolSafetyConnection",
    extraVariables: { agentIds: [...agentIds] },
  });
}

export type AgentToolSafetyRow = DataTableConfigRow<
  ReturnType<typeof createAgentToolSafetyTableGraphQLConfig>
>;

/** The section's figures, counted on the server over every tool and agent. */
export async function fetchAgentSafetySummary(
  options?: RequestOptions,
): Promise<AgentSafetySummary> {
  const data = await requestGraphQL({
    document: AgentSafetySummaryDocument,
    operationName: "AgentSafetySummary",
    signal: options?.signal,
  });

  return data.agentSafetySummary;
}

/**
 * Who can reach each named agent and what that warns of. The tools are read
 * by the table, so the header asks for none of them. An empty list reads
 * nothing rather than every agent.
 */
export async function fetchAgentSafetyHeaders(
  agentIds: readonly string[],
  options?: RequestOptions,
): Promise<AgentSafetyHeader[]> {
  if (agentIds.length === 0) {
    return [];
  }

  const data = await requestGraphQL({
    document: AgentSafetyDocument,
    operationName: "AgentSafety",
    variables: { agentIds: [...agentIds] },
    signal: options?.signal,
  });

  return data.agentSafety.map((entry) =>
    getFragmentData(AgentSafetyHeaderFieldsFragmentDoc, entry),
  );
}

/** The most rows one page of the tool rules may hold. */
const TOOL_RULE_PAGE = 100;

/**
 * Every tool's rule, by tool name, read a page at a time until the last: what it can
 * reach, how far it may ever run on its own, and whether it reads outside text.
 */
export async function fetchToolRules(
  options?: RequestOptions,
): Promise<Map<string, AgentToolPolicy>> {
  const rules = new Map<string, AgentToolPolicy>();
  let after: string | null = null;
  for (;;) {
    const data: AgentToolRuleTableQuery = await requestGraphQL({
      document: AgentToolRuleTableDocument,
      operationName: "AgentToolRuleTable",
      variables: {
        input: { first: TOOL_RULE_PAGE, ...(after ? { after } : {}) },
        includeTotalCount: false,
      },
      signal: options?.signal,
    });
    const page = data.agentToolRuleConnection;
    for (const edge of page.edges) {
      const rule = getFragmentData(AgentToolPolicyFieldsFragmentDoc, edge.node);
      rules.set(rule.name, rule);
    }
    const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, page.pageInfo);
    const next = pageInfo.endCursor;
    if (!pageInfo.hasNextPage || !next || next === after) {
      return rules;
    }
    after = next;
  }
}
