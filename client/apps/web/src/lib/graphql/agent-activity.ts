import {
  AgentDefinitionCountDocument,
  AgentProposalCountDocument,
  AgentRunCountDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { fetchActiveAgentMemoryCount } from "./agent-memories";

type RequestOptions = { signal?: AbortSignal };

export type AgentActivityCounts = {
  agentsTotal: number;
  agentsEnabled: number;
  pendingProposals: number;
  runsLast24h: number;
  memoriesActive: number;
};

const DAY_SECONDS = 24 * 60 * 60;

/**
 * Counts for the AI Control overview, each a `totalCount`-only connection
 * query so the server runs a COUNT and returns no rows.
 */
export async function fetchAgentActivityCounts(
  options?: RequestOptions,
): Promise<AgentActivityCounts> {
  const since = Math.floor(Date.now() / 1000) - DAY_SECONDS;
  const signal = options?.signal;

  const [agentsTotal, agentsEnabled, pendingProposals, runsLast24h, memoriesActive] =
    await Promise.all([
      requestGraphQL({
        document: AgentDefinitionCountDocument,
        operationName: "AgentDefinitionCount",
        variables: { input: { first: 1 } },
        signal,
      }).then((data) => data.agentDefinitions.totalCount ?? 0),
      requestGraphQL({
        document: AgentDefinitionCountDocument,
        operationName: "AgentDefinitionCount",
        variables: {
          input: { first: 1, fieldFilters: [{ field: "enabled", operator: "eq", value: true }] },
        },
        signal,
      }).then((data) => data.agentDefinitions.totalCount ?? 0),
      requestGraphQL({
        document: AgentProposalCountDocument,
        operationName: "AgentProposalCount",
        variables: {
          input: {
            first: 1,
            fieldFilters: [{ field: "status", operator: "eq", value: "Pending" }],
          },
        },
        signal,
      }).then((data) => data.agentProposals.totalCount ?? 0),
      requestGraphQL({
        document: AgentRunCountDocument,
        operationName: "AgentRunCount",
        variables: {
          input: {
            first: 1,
            fieldFilters: [{ field: "createdAt", operator: "gte", value: since }],
          },
        },
        signal,
      }).then((data) => data.agentRuns.totalCount ?? 0),
      fetchActiveAgentMemoryCount({ signal }),
    ]);

  return { agentsTotal, agentsEnabled, pendingProposals, runsLast24h, memoriesActive };
}

/** How many proposals still wait on a person, for the assistant launcher's badge. */
export async function fetchPendingProposalCount(options?: RequestOptions): Promise<number> {
  const data = await requestGraphQL({
    document: AgentProposalCountDocument,
    operationName: "AgentProposalCount",
    variables: {
      input: { first: 1, fieldFilters: [{ field: "status", operator: "eq", value: "Pending" }] },
    },
    signal: options?.signal,
  });

  return data.agentProposals.totalCount ?? 0;
}
