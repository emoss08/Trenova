import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentSafetyDocument,
  AgentSafetyFieldsFragmentDoc,
  AgentSafetySummaryDocument,
  AgentToolAutonomyFieldsFragmentDoc,
  AgentToolPolicyConnectionDocument,
  AgentToolPolicyFieldsFragmentDoc,
  DataTablePageInfoFieldsFragmentDoc,
  type AgentAutonomyAnswer,
  type AgentEgressClass,
  type AgentExternalRead,
  type AgentReachWarningKind,
  type AgentSafetyFieldsFragment,
  type AgentSafetySummaryQuery,
  type AgentToolAutonomyFieldsFragment,
  type AgentToolKind,
  type AgentToolPolicyConnectionInput,
  type AgentToolPolicyFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type AgentToolPolicy = AgentToolPolicyFieldsFragment;
export type AgentToolAutonomy = AgentToolAutonomyFieldsFragment;
export type AgentSafetySummary = AgentSafetySummaryQuery["agentSafetySummary"];

export type AgentToolSafety = {
  policyName: string;
  policy: AgentToolPolicy;
  clean: AgentToolAutonomy;
  tainted: AgentToolAutonomy;
};

export type AgentSafety = Omit<AgentSafetyFieldsFragment, "tools"> & {
  tools: AgentToolSafety[];
};

export type {
  AgentAutonomyAnswer,
  AgentEgressClass,
  AgentExternalRead,
  AgentReachWarningKind,
  AgentToolKind,
};

type RequestOptions = { signal?: AbortSignal };

/** Whether a tool runs without a person on some agent, or on none, or either. */
export type ToolPolicyAttendance = "all" | "alone" | "attended";

/** What the tool rules are narrowed by; "all" and a blank search leave a filter open. */
export type ToolPolicyFilter = {
  search: string;
  egress: AgentEgressClass | "all";
  resource: string;
  kind: AgentToolKind | "all";
  attendance: ToolPolicyAttendance;
};

export const ALL_TOOL_POLICIES: ToolPolicyFilter = {
  search: "",
  egress: "all",
  resource: "all",
  kind: "all",
  attendance: "all",
};

export type ToolPolicyPageRequest = {
  first: number;
  after: string | null;
  /** The count walks every rule on the server, so only the first page of a scope asks for it. */
  includeTotalCount: boolean;
};

export type ToolPolicyPage = {
  items: AgentToolPolicy[];
  endCursor: string | null;
  hasNextPage: boolean;
  /** Only read when the page asked for it. */
  totalCount: number | null;
};

/**
 * The connection input for a filter and a page. Only what narrows is sent: an
 * open filter, a blank search and a missing cursor are left out rather than
 * sent empty, so two pages that ask the same thing share one cache entry.
 */
export function toolPolicyConnectionInput(
  filter: ToolPolicyFilter,
  page: Pick<ToolPolicyPageRequest, "first" | "after">,
): AgentToolPolicyConnectionInput {
  const input: AgentToolPolicyConnectionInput = { first: page.first };
  if (page.after) {
    input.after = page.after;
  }
  const search = filter.search.trim();
  if (search) {
    input.query = search;
  }
  if (filter.egress !== "all") {
    input.egress = filter.egress;
  }
  if (filter.resource !== "all") {
    input.resource = filter.resource;
  }
  if (filter.kind !== "all") {
    input.kind = filter.kind;
  }
  if (filter.attendance !== "all") {
    input.runsWithoutPerson = filter.attendance === "alone";
  }

  return input;
}

/** One page of tool rules, by name, searched and filtered on the server. */
export async function fetchToolPolicyPage(
  filter: ToolPolicyFilter,
  page: ToolPolicyPageRequest,
  options?: RequestOptions,
): Promise<ToolPolicyPage> {
  const data = await requestGraphQL({
    document: AgentToolPolicyConnectionDocument,
    operationName: "AgentToolPolicyConnection",
    variables: {
      input: toolPolicyConnectionInput(filter, page),
      includeTotalCount: page.includeTotalCount,
    },
    signal: options?.signal,
  });
  const connection = data.agentToolPolicyConnection;
  const pageInfo = getFragmentData(DataTablePageInfoFieldsFragmentDoc, connection.pageInfo);

  return {
    items: connection.edges.map((edge) =>
      getFragmentData(AgentToolPolicyFieldsFragmentDoc, edge.node),
    ),
    endCursor: pageInfo.endCursor ?? null,
    hasNextPage: pageInfo.hasNextPage,
    totalCount: connection.totalCount ?? null,
  };
}

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

/** Only the agents named; an empty list reads nothing rather than every agent. */
export async function fetchAgentSafety(
  agentIds: readonly string[],
  options?: RequestOptions,
): Promise<AgentSafety[]> {
  if (agentIds.length === 0) {
    return [];
  }

  const data = await requestGraphQL({
    document: AgentSafetyDocument,
    operationName: "AgentSafety",
    variables: { agentIds: [...agentIds] },
    signal: options?.signal,
  });

  return data.agentSafety.map((entry) => {
    const safety = getFragmentData(AgentSafetyFieldsFragmentDoc, entry);
    return {
      ...safety,
      tools: safety.tools.map((tool) => ({
        policyName: tool.policyName,
        policy: getFragmentData(AgentToolPolicyFieldsFragmentDoc, tool.policy),
        clean: getFragmentData(AgentToolAutonomyFieldsFragmentDoc, tool.clean),
        tainted: getFragmentData(AgentToolAutonomyFieldsFragmentDoc, tool.tainted),
      })),
    };
  });
}
