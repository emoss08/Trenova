import { queries } from "@/lib/queries";
import { QueryClient, QueryClientProvider, QueryObserver } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const handlers = vi.hoisted(() => new Map<string, (data: unknown) => void>());

vi.mock("@trenova/shared/services/realtime", () => ({
  realtimeClient: {
    on: (event: string, handler: (data: unknown) => void) => {
      handlers.set(event, handler);
      return () => handlers.delete(event);
    },
    onStateChange: () => () => {},
    connect: () => {},
    disconnect: () => {},
    getState: () => "connected",
  },
}));

const { useRealtimeConnection } = await import("../use-realtime-connection");

const ORG = "org_1";
const BU = "bu_1";
const OPEN = "athr_open";
const BACKGROUND = Array.from({ length: 23 }, (_, i) => `athr_bg${i}`);

function emit(resource: string, entity: Record<string, unknown>) {
  handlers.get("resource.invalidation")?.({
    organizationId: ORG,
    businessUnitId: BU,
    resource,
    action: "updated",
    recordId: entity.id,
    entity,
  });
}

/*
The bench's turns on 24 conversations sent the open Desk ~360 requests a
minute: every proposal or artifact event refetched the proposals and artifact
lists of every conversation the tab had ever opened, on screen or not, until
the API answered 429. Only what is mounted refetches, an artifact event reaches
only its own conversation, and a burst on one conversation coalesces.
*/
describe("realtime invalidation of Desk conversations", () => {
  let client: QueryClient;
  let fetches: Map<string, number>;
  const unsubscribes: Array<() => void> = [];

  const counted = (name: string) => () => {
    fetches.set(name, (fetches.get(name) ?? 0) + 1);
    return Promise.resolve({ results: [] });
  };

  beforeEach(async () => {
    vi.useFakeTimers();
    fetches = new Map();
    client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    useAuthStore.setState({
      isAuthenticated: true,
      user: {
        id: "usr_1",
        currentOrganizationId: ORG,
        businessUnitId: BU,
      } as never,
    });

    for (const thread of [OPEN, ...BACKGROUND]) {
      await client.fetchQuery({
        queryKey: queries.assistant.proposals(thread).queryKey,
        queryFn: counted(`proposals:${thread}`),
      });
      await client.fetchQuery({
        queryKey: queries.assistant.artifacts(thread)._ctx.summary.queryKey,
        queryFn: counted(`artifacts:${thread}`),
      });
    }
    fetches.clear();

    for (const [name, queryKey] of [
      ["proposals", queries.assistant.proposals(OPEN).queryKey],
      ["artifacts", queries.assistant.artifacts(OPEN)._ctx.summary.queryKey],
    ] as const) {
      const observer = new QueryObserver(client, {
        queryKey,
        queryFn: counted(`${name}:${OPEN}`),
        staleTime: Infinity,
      });
      unsubscribes.push(observer.subscribe(() => {}));
    }
  });

  afterEach(() => {
    unsubscribes.splice(0).forEach((off) => off());
    client.clear();
    vi.useRealTimers();
  });

  function mount() {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <MemoryRouter>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </MemoryRouter>
    );
    return renderHook(() => useRealtimeConnection(), { wrapper });
  }

  const total = (prefix: string) =>
    Array.from(fetches.entries())
      .filter(([name]) => name.startsWith(prefix))
      .reduce((sum, [, count]) => sum + count, 0);

  it("fetches nothing for conversations that are not on screen", async () => {
    mount();

    await act(async () => {
      for (let i = 0; i < 50; i += 1) {
        emit("assistant_artifact", {
          id: `aart_${i}`,
          threadId: BACKGROUND[i % BACKGROUND.length],
        });
        emit("agent_proposal", { id: `ap_${i}` });
        await vi.advanceTimersByTimeAsync(100);
      }
      await vi.advanceTimersByTimeAsync(5_000);
    });

    const background = BACKGROUND.reduce(
      (sum, thread) => sum + total(`proposals:${thread}`) + total(`artifacts:${thread}`),
      0,
    );
    expect(background).toBe(0);
    expect(total(`artifacts:${OPEN}`)).toBe(0);
  });

  it("refetches the open conversation at most about once a second during a burst", async () => {
    mount();

    await act(async () => {
      for (let i = 0; i < 50; i += 1) {
        emit("assistant_artifact", { id: `aart_${i}`, threadId: OPEN });
        emit("agent_proposal", { id: `ap_${i}` });
        await vi.advanceTimersByTimeAsync(100);
      }
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(total(`artifacts:${OPEN}`)).toBeGreaterThanOrEqual(1);
    expect(total(`artifacts:${OPEN}`)).toBeLessThanOrEqual(7);
    expect(total(`proposals:${OPEN}`)).toBeGreaterThanOrEqual(1);
    expect(total(`proposals:${OPEN}`)).toBeLessThanOrEqual(7);
  });
});
