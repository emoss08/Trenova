import {
  AgentMemoryCountDocument,
  AgentMemoryTableDocument,
  CreateAgentMemoryDocument,
  SetAgentMemoryStatusDocument,
  UpdateAgentMemoryDocument,
  type AgentMemoryInput,
  type AgentMemoryStatus,
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
