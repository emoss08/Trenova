import type { AgentMemoryKind } from "@trenova/graphql/generated/graphql";

/**
 * The client copy of the server's `agentmemory.MemoryKind.MaxContentChars`: a Procedure is a
 * runbook and holds up to 12,000 characters, every other memory up to 4,000. A prompt shows
 * the first 1,200 and names the memory's id so the agent can read the rest.
 */
export const MEMORY_CONTENT_LIMIT = 4000;
export const PROCEDURE_CONTENT_LIMIT = 12000;

export function memoryContentLimit(kind: AgentMemoryKind | null | undefined): number {
  return kind === "Procedure" ? PROCEDURE_CONTENT_LIMIT : MEMORY_CONTENT_LIMIT;
}
