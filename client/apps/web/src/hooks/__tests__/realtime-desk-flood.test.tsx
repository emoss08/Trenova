import { TestRouter } from "@trenova/shared/testing/test-router";
import { assistant } from "@/lib/queries/assistant";
import { QueryClient, QueryClientProvider, QueryObserver } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const handlers = vi.hoisted(() => new Map<string, (data: unknown) => void>());

vi.mock("@trenova/shared/services/realtime", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/services/realtime")>()),
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

const loadEvents = vi.hoisted(() => ({ failures: 0, attempts: 0 }));

// Loads the real event reader, after failing as many times as a test asks: a network blip, or
// a release that replaced the file under the open tab.
vi.mock("@/lib/realtime-events-loader", () => ({
  loadRealtimeEvents: () => {
    loadEvents.attempts += 1;
    if (loadEvents.failures > 0) {
      loadEvents.failures -= 1;
      return Promise.reject(new TypeError("Failed to fetch dynamically imported module"));
    }
    return import("@/lib/realtime-events");
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
    loadEvents.failures = 0;
    loadEvents.attempts = 0;
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
        queryKey: assistant.proposals(thread).queryKey,
        queryFn: counted(`proposals:${thread}`),
      });
      await client.fetchQuery({
        queryKey: assistant.artifacts(thread)._ctx.summary.queryKey,
        queryFn: counted(`artifacts:${thread}`),
      });
    }
    fetches.clear();

    for (const [name, queryKey] of [
      ["proposals", assistant.proposals(OPEN).queryKey],
      ["artifacts", assistant.artifacts(OPEN)._ctx.summary.queryKey],
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

  function render() {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <TestRouter>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </TestRouter>
    );
    return renderHook(() => useRealtimeConnection(), { wrapper });
  }

  // The hook loads what reads each event once it connects; these tests start after it.
  async function mount() {
    const mounted = render();
    await act(async () => {
      await vi.dynamicImportSettled();
    });
    return mounted;
  }

  const total = (prefix: string) =>
    Array.from(fetches.entries())
      .filter(([name]) => name.startsWith(prefix))
      .reduce((sum, [, count]) => sum + count, 0);

  it("fetches nothing for conversations that are not on screen", async () => {
    await mount();

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
    await mount();

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

  it("handles events that arrive before the event reader has loaded, once it has", async () => {
    render();
    emit("assistant_artifact", { id: "aart_early", threadId: OPEN });

    await act(async () => {
      await vi.dynamicImportSettled();
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(total(`artifacts:${OPEN}`)).toBe(1);
  });

  it("events still reach the router when its first load fails and a retry succeeds", async () => {
    loadEvents.failures = 1;
    render();
    emit("assistant_artifact", { id: "aart_before_retry", threadId: OPEN });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000);
      await vi.dynamicImportSettled();
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(loadEvents.attempts).toBe(2);
    expect(total(`artifacts:${OPEN}`)).toBe(1);
  });

  it("stops retrying the load once the connection closes", async () => {
    loadEvents.failures = Number.POSITIVE_INFINITY;
    const { unmount } = render();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(20_000);
    });
    const attemptsBeforeClose = loadEvents.attempts;
    expect(attemptsBeforeClose).toBeGreaterThan(1);
    unmount();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_000);
    });
    expect(loadEvents.attempts).toBe(attemptsBeforeClose);
  });

  it("refetches what every screen leans on when more events arrive than it holds", async () => {
    const queue = new QueryObserver(client, {
      queryKey: ["billingQueue"],
      queryFn: counted("billingQueue"),
      staleTime: Infinity,
    });
    unsubscribes.push(queue.subscribe(() => {}));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    fetches.clear();
    loadEvents.failures = 1;
    render();

    for (let i = 0; i < 201; i += 1) {
      handlers.get("resource.invalidation")?.({
        organizationId: "org_other",
        businessUnitId: BU,
        resource: "agent_proposal",
        action: "updated",
        recordId: `ap_${i}`,
      });
    }

    await act(async () => {
      await vi.advanceTimersByTimeAsync(10_000);
      await vi.dynamicImportSettled();
      await vi.advanceTimersByTimeAsync(5_000);
    });

    expect(total("billingQueue")).toBe(1);
  });
});
