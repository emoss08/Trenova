import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AgentMemoryCountDocument,
  AgentMemoryTableDocument,
  AgentMemoryTableRowFieldsFragmentDoc,
  ApproveAgentMemorySuggestionDocument,
  CreateAgentMemoryDocument,
  DismissAgentMemorySuggestionDocument,
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

  return data.createAgentMemory;
}

export async function updateAgentMemory(id: string, input: AgentMemoryInput) {
  const data = await requestGraphQL({
    document: UpdateAgentMemoryDocument,
    operationName: "UpdateAgentMemory",
    variables: { id, input },
  });

  return data.updateAgentMemory;
}

export async function setAgentMemoryStatus(id: string, status: AgentMemoryStatus) {
  const data = await requestGraphQL({
    document: SetAgentMemoryStatusDocument,
    operationName: "SetAgentMemoryStatus",
    variables: { id, status },
  });

  return data.setAgentMemoryStatus;
}

/** How many memories agents are currently reading, for the rail. */
export async function fetchActiveAgentMemoryCount(options?: {
  signal?: AbortSignal;
}): Promise<number> {
  const data = await requestGraphQL({
    document: AgentMemoryCountDocument,
    operationName: "AgentMemoryCount",
    variables: {
      input: { first: 1, fieldFilters: [{ field: "status", operator: "eq", value: "Active" }] },
    },
    signal: options?.signal,
  });

  return data.agentMemories.totalCount ?? 0;
}

export type AgentMemorySuggestion = AgentMemoryTableRowFieldsFragment;

export const AGENT_MEMORY_SUGGESTIONS_KEY = "agent-memory-suggestions";

/** The most suggestions the review list reads at once. */
const SUGGESTION_PAGE_SIZE = 50;

/** Memories drawn from feedback that wait for an administrator, newest first. */
export async function fetchAgentMemorySuggestions(options?: {
  signal?: AbortSignal;
}): Promise<AgentMemorySuggestion[]> {
  const data = await requestGraphQL({
    document: AgentMemoryTableDocument,
    operationName: "AgentMemoryTable",
    variables: {
      input: {
        first: SUGGESTION_PAGE_SIZE,
        fieldFilters: [{ field: "status", operator: "eq", value: "Suggested" }],
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
