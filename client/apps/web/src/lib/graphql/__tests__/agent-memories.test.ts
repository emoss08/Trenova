import { AgentMemoryUsageDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AGENT_MEMORY_LIST_KEY,
  agentMemoryUsageQueryKey,
  fetchAgentMemoryUsage,
  memoryUsageNearCap,
} from "../agent-memories";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

describe("agent memory usage", () => {
  beforeEach(() => {
    requestGraphQLMock.mockReset();
  });

  it("reads the count and the cap from the server", async () => {
    const usage = { activeCount: 4100, activeSoftCap: 5000, warnAt: 4000 };
    requestGraphQLMock.mockResolvedValueOnce({ agentMemoryUsage: usage } as never);

    await expect(fetchAgentMemoryUsage()).resolves.toEqual(usage);
    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({
        document: AgentMemoryUsageDocument,
        operationName: "AgentMemoryUsage",
      }),
    );
  });

  it("warns from the server's threshold, not a number of its own", () => {
    expect(memoryUsageNearCap(undefined)).toBe(false);
    expect(memoryUsageNearCap({ activeCount: 3999, activeSoftCap: 5000, warnAt: 4000 })).toBe(
      false,
    );
    expect(memoryUsageNearCap({ activeCount: 4000, activeSoftCap: 5000, warnAt: 4000 })).toBe(true);
    expect(memoryUsageNearCap({ activeCount: 120, activeSoftCap: 150, warnAt: 100 })).toBe(true);
  });

  it("is refreshed by whatever refreshes the memory list", () => {
    expect(agentMemoryUsageQueryKey[0]).toBe(AGENT_MEMORY_LIST_KEY);
  });
});
