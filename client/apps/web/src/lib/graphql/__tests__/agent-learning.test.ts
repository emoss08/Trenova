import { DESK_MEMORIES_KEY } from "@/lib/graphql/desk-memories";
import { queries } from "@/lib/queries";
import {
  AgentMemoryTableDocument,
  AgentReflectionsDocument,
} from "@trenova/graphql/generated/graphql";
import { RESOURCE_QUERY_KEY_MAP, queryKeyPrefix } from "@trenova/shared/hooks/realtime-patching";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchAgentMemorySuggestions } from "../agent-memories";
import { fetchRecentAgentReflections } from "../agent-reflections";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables?: { input?: Record<string, unknown>; includeTotalCount?: boolean };
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

/*
A lesson an agent offers in a person's own conversation is theirs to keep or
turn down on the Desk. AI Control's queue holds only what reaches beyond one
person: a lesson for everyone using an agent, and one for the organization.
*/
describe("fetchAgentMemorySuggestions", () => {
  it("asks for suggestions kept for an agent or the organization, newest first", async () => {
    requestGraphQLMock.mockResolvedValue({ agentMemories: { edges: [] } } as never);
    const controller = new AbortController();

    await fetchAgentMemorySuggestions({ signal: controller.signal });

    const call = lastCall();
    expect(call.document).toBe(AgentMemoryTableDocument);
    expect(call.signal).toBe(controller.signal);
    expect(call.variables?.includeTotalCount).toBe(false);
    expect(call.variables?.input).toMatchObject({
      fieldFilters: [
        { field: "status", operator: "eq", value: "Suggested" },
        { field: "scope", operator: "in", value: ["Agent", "Organization"] },
      ],
      sort: [{ field: "createdAt", direction: "desc" }],
    });
  });
});

/*
Most look backs are skipped because nothing in the work called for one. The
panel lists only those that decided something or could not finish.
*/
describe("fetchRecentAgentReflections", () => {
  it("asks for completed and failed look backs, newest first", async () => {
    requestGraphQLMock.mockResolvedValue({ agentReflections: { edges: [] } } as never);
    const controller = new AbortController();

    await expect(fetchRecentAgentReflections({ signal: controller.signal })).resolves.toEqual([]);

    const call = lastCall();
    expect(call.document).toBe(AgentReflectionsDocument);
    expect(call.signal).toBe(controller.signal);
    expect(call.variables?.input).toMatchObject({
      fieldFilters: [{ field: "status", operator: "in", value: ["Completed", "Failed"] }],
      sort: [{ field: "createdAt", direction: "desc" }],
    });
  });
});

function reaches(resource: string, queryKey: readonly unknown[]): boolean {
  return (RESOURCE_QUERY_KEY_MAP[resource] ?? []).some((root) =>
    queryKeyPrefix(root).every((part, index) => queryKey[index] === part),
  );
}

/*
When a look back keeps a lesson from a conversation, agentreflectionservice
publishes resource "agent_memory", action "learned", to that person
(services/tms/internal/core/ports/services/agentreflection.go). The note under
the reply and the Desk's memory list both have to move with it.
*/
describe("agent_memory realtime", () => {
  it("reads the conversation's replies again, so the note appears under one", () => {
    expect(reaches("agent_memory", queries.assistant.messages("athr_1").queryKey)).toBe(true);
  });

  it("reads every Desk memory list and card again", () => {
    expect(reaches("agent_memory", [DESK_MEMORIES_KEY, "by-ids", ["amem_1"]])).toBe(true);
    expect(reaches("agent_memory", [DESK_MEMORIES_KEY, "list", { scope: null }])).toBe(true);
  });
});
