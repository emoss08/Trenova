import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentMemoryCountDocument,
  AgentMemoryTableDocument,
  AgentMemoryTableRowFieldsFragmentDoc,
  AgentMemoryUsageDocument,
  ApproveAgentMemorySuggestionDocument,
  CreateAgentMemoryDocument,
  DismissAgentMemorySuggestionDocument,
  ReviewAgentMemoryDocument,
  SetAgentMemoryStatusDocument,
  UpdateAgentMemoryDocument,
  type AgentMemoryInput,
  type AgentMemoryKind,
  type AgentMemoryStatus,
  type AgentMemoryTableRowFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const agentMemoryTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AgentMemoryTableDocument,
  operationName: "AgentMemoryTable",
  connectionKey: "agentMemories",
});

export type AgentMemoryRow = DataTableConfigRow<typeof agentMemoryTableGraphQLConfig>;

export const AGENT_MEMORY_LIST_KEY = "agent-memory-list";

export async function createAgentMemory(input: AgentMemoryInput) {
  const data = await requestGraphQL({
    document: CreateAgentMemoryDocument,
    operationName: "CreateAgentMemory",
    variables: { input },
  });

  return getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, data.createAgentMemory);
}

export async function updateAgentMemory(id: string, input: AgentMemoryInput) {
  const data = await requestGraphQL({
    document: UpdateAgentMemoryDocument,
    operationName: "UpdateAgentMemory",
    variables: { id, input },
  });

  return getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, data.updateAgentMemory);
}

export async function setAgentMemoryStatus(id: string, status: AgentMemoryStatus) {
  const data = await requestGraphQL({
    document: SetAgentMemoryStatusDocument,
    operationName: "SetAgentMemoryStatus",
    variables: { id, status },
  });

  return data.setAgentMemoryStatus;
}

/**
 * Says a person has read a memory written after outside content and keeps it:
 * from then on it is followed as the organization's own and taints no turn.
 */
export async function reviewAgentMemory(id: string, version: number) {
  const data = await requestGraphQL({
    document: ReviewAgentMemoryDocument,
    operationName: "ReviewAgentMemory",
    variables: { id, version },
  });

  return getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, data.reviewAgentMemory);
}

export const taintingAgentMemoriesQueryKey = [AGENT_MEMORY_LIST_KEY, "tainting"] as const;

/** The most tainting memories the notice reads at once. */
const TAINTING_PAGE_SIZE = 25;

/**
 * Active memories written after outside content that nobody has reviewed. Each
 * taints every turn that reads it, so those turns hold their money,
 * customer-visible and outside-recipient writes for a person.
 */
export async function fetchTaintingAgentMemories(options?: {
  signal?: AbortSignal;
}): Promise<AgentMemoryTableRowFieldsFragment[]> {
  const data = await requestGraphQL({
    document: AgentMemoryTableDocument,
    operationName: "AgentMemoryTable",
    variables: {
      input: {
        first: TAINTING_PAGE_SIZE,
        fieldFilters: [
          { field: "status", operator: "eq", value: "Active" },
          { field: "tainted", operator: "eq", value: true },
          { field: "reviewedAt", operator: "isnull", value: null },
        ],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
      includeTotalCount: false,
    },
    signal: options?.signal,
  });

  return data.agentMemories.edges.map((edge) =>
    getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, edge.node),
  );
}

/** How many memories agents are currently reading, for the rail. */
export async function fetchActiveAgentMemoryCount(options?: {
  signal?: AbortSignal;
}): Promise<number> {
  return countAgentMemories([{ field: "status", operator: "eq", value: "Active" }], options);
}

export const agentMemoryTotalQueryKey = [AGENT_MEMORY_LIST_KEY, "total"] as const;

/** How many memories the organization keeps in any state, to tell an empty list apart. */
export async function fetchAgentMemoryTotal(options?: { signal?: AbortSignal }): Promise<number> {
  return countAgentMemories([], options);
}

async function countAgentMemories(
  fieldFilters: { field: string; operator: string; value: string }[],
  options?: { signal?: AbortSignal },
): Promise<number> {
  const data = await requestGraphQL({
    document: AgentMemoryCountDocument,
    operationName: "AgentMemoryCount",
    variables: { input: { first: 1, fieldFilters } },
    signal: options?.signal,
  });

  return data.agentMemories.totalCount ?? 0;
}

/**
 * How many memories agents may read against the organization's soft cap. It
 * sits under the list's key, so whatever refreshes the list refreshes it too.
 */
export const agentMemoryUsageQueryKey = [AGENT_MEMORY_LIST_KEY, "usage"] as const;

export type AgentMemoryUsage = {
  activeCount: number;
  activeSoftCap: number;
  warnAt: number;
};

export async function fetchAgentMemoryUsage(options?: {
  signal?: AbortSignal;
}): Promise<AgentMemoryUsage> {
  const data = await requestGraphQL({
    document: AgentMemoryUsageDocument,
    operationName: "AgentMemoryUsage",
    signal: options?.signal,
  });

  return data.agentMemoryUsage;
}

/** Whether the organization keeps enough memories that AI Control should say so. */
export function memoryUsageNearCap(usage: AgentMemoryUsage | undefined): boolean {
  return usage !== undefined && usage.activeCount >= usage.warnAt;
}

export type AgentMemorySuggestion = AgentMemoryTableRowFieldsFragment;

export const AGENT_MEMORY_SUGGESTIONS_KEY = "agent-memory-suggestions";

/** The most suggestions the review list reads at once. */
const SUGGESTION_PAGE_SIZE = 50;

/**
 * Memories that wait for an administrator, newest first: those drawn from
 * feedback and those an agent learned that reach beyond one person. A memory
 * offered to one person in their own conversation is theirs to decide on the Desk.
 */
export async function fetchAgentMemorySuggestions(options?: {
  signal?: AbortSignal;
}): Promise<AgentMemorySuggestion[]> {
  const data = await requestGraphQL({
    document: AgentMemoryTableDocument,
    operationName: "AgentMemoryTable",
    variables: {
      input: {
        first: SUGGESTION_PAGE_SIZE,
        fieldFilters: [
          { field: "status", operator: "eq", value: "Suggested" },
          { field: "scope", operator: "in", value: ["Agent", "Organization"] },
        ],
        sort: [{ field: "createdAt", direction: "desc" }],
      },
      includeTotalCount: false,
    },
    signal: options?.signal,
  });

  return data.agentMemories.edges.map((edge) =>
    getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, edge.node),
  );
}

export async function approveAgentMemorySuggestion(
  id: string,
  input: { content: string; kind?: AgentMemoryKind; version: number },
) {
  const data = await requestGraphQL({
    document: ApproveAgentMemorySuggestionDocument,
    operationName: "ApproveAgentMemorySuggestion",
    variables: { id, input },
  });

  return getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, data.approveAgentMemorySuggestion);
}

export async function dismissAgentMemorySuggestion(id: string, version: number) {
  const data = await requestGraphQL({
    document: DismissAgentMemorySuggestionDocument,
    operationName: "DismissAgentMemorySuggestion",
    variables: { id, version },
  });

  return getFragmentData(AgentMemoryTableRowFieldsFragmentDoc, data.dismissAgentMemorySuggestion);
}
