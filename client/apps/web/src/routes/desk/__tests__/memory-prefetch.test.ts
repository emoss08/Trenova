import { agentProfileQueryKey } from "@/lib/graphql/agent-profile";
import { DESK_MEMORIES_KEY } from "@/lib/graphql/desk-memories";
import type { LoaderFunctionArgs } from "react-router";
import { describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
  fetchAgentProfile: vi.fn(),
  fetchDeskMemorySettings: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-profile", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-profile")>()),
  fetchAgentProfile: api.fetchAgentProfile,
}));

vi.mock("@/lib/graphql/desk-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/desk-memories")>()),
  fetchDeskMemorySettings: api.fetchDeskMemorySettings,
}));

vi.mock("../_components/memory/desk-memory-page", () => ({
  DeskMemoryPage: () => null,
}));

const loaderArgs = {
  request: new Request("http://localhost/desk/memory"),
  params: {},
  context: undefined,
} as unknown as LoaderFunctionArgs;

// The route warms exactly what the page reads on mount, under the keys the page's own
// queries use, so the card and the settings paint with the page instead of after it.
describe("desk memory prefetch", () => {
  it("warms the person's own profile and the memory settings under the page's keys", async () => {
    const { prefetch } = await import("@/routes/desk/memory-page");
    const list = await prefetch(loaderArgs);
    const signal = new AbortController().signal;

    expect(list.map((entry) => entry.queryKey)).toEqual([
      agentProfileQueryKey("mine"),
      [DESK_MEMORIES_KEY, "settings"],
    ]);

    for (const entry of list) {
      await entry.queryFn?.({ signal } as never);
    }
    expect(api.fetchAgentProfile).toHaveBeenCalledWith("mine", { signal });
    expect(api.fetchDeskMemorySettings).toHaveBeenCalledWith(signal);
  });
});
