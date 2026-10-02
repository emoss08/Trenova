import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskHome, type DeskHomeProps } from "../desk-home";

const state = vi.hoisted(() => ({
  canDecide: true,
  permissionsLoading: false,
  briefing: { settled: true, value: null as unknown },
  summary: { isPending: false, total: 2 },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" ? state.canDecide : true,
    isLoading: state.permissionsLoading,
  }),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    assistant: {
      activeTurns: () => ({ queryKey: ["active-turns"], queryFn: () => ({ items: [] }) }),
    },
    watchtower: {
      counts: () => ({
        queryKey: ["watchtower-counts"],
        queryFn: () => ({ unseen: 0, unseenCritical: 0 }),
      }),
      feed: () => ({
        queryKey: ["watchtower-feed"],
        queryFn: () => ({ items: [], endCursor: null, hasNextPage: false, seenAt: 0 }),
      }),
    },
    briefing: {
      today: () => ({
        queryKey: ["briefing-today", state.briefing.settled],
        queryFn: () =>
          state.briefing.settled ? state.briefing.value : new Promise(() => undefined),
      }),
    },
  },
}));

vi.mock("../decisions/use-pending-decisions", () => ({
  usePendingDecisionSummary: (enabled: boolean) =>
    enabled && state.summary.isPending
      ? { isPending: true, data: undefined }
      : {
          isPending: !enabled,
          data: enabled ? { total: state.summary.total, byAgent: [], oldestAt: null } : undefined,
        },
}));

vi.mock("@/hooks/use-attention", () => ({
  useAttentionSummary: () => ({ data: undefined }),
}));

vi.mock("@/components/assistant/use-active-turns", () => ({
  useLiveThreadIds: () => new Set<string>(),
}));

vi.mock("@/components/assistant/use-askable-agent", () => ({
  useAskableAgent: () => ({
    agent: null,
    choose: () => undefined,
    recency: { ids: [], lastUsedAt: {} },
    choices: { isLoading: true, isError: false, recent: [], items: [], refetch: () => undefined },
    noneAvailable: false,
  }),
}));

vi.mock("../briefing-panel", () => ({ BriefingPanel: () => null }));

const agent = { id: "agent-1", name: "Dispatch" } as DeskHomeProps["agents"][number];

function renderHome(props: Partial<DeskHomeProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskHome
          agents={[agent]}
          threads={[]}
          isLoading={false}
          isStarting={false}
          onStart={() => undefined}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * The greeting is known before anything loads; the line under it is the one
 * the figures decide, so that is the line held as a skeleton rather than
 * said and then taken back.
 */
async function expectHeadlineSkeleton() {
  const heading = await screen.findByRole("heading", { level: 1 });
  const second = heading.parentElement?.querySelector('[data-slot="desk-headline"]');
  expect(second?.querySelector('[data-slot="skeleton"]')).not.toBeNull();
  expect(second).toHaveTextContent("");
  expect(heading).toHaveTextContent(/^Good (morning|afternoon|evening)$/);
  expect(screen.queryByText(/waiting on you/)).toBeNull();
  expect(screen.queryByText("How can I help today?")).toBeNull();
  expect(document.querySelector('[data-slot="desk-loading-mark"]')).toBeNull();
}

beforeEach(() => {
  state.canDecide = true;
  state.permissionsLoading = false;
  state.briefing = { settled: true, value: null };
  state.summary = { isPending: false, total: 2 };
});

describe("Desk home headline while its figures load", () => {
  it("holds the headline as a skeleton while the agents and conversations load", async () => {
    renderHome({ isLoading: true });

    await expectHeadlineSkeleton();
  });

  it("holds the headline as a skeleton while permissions are still arriving", async () => {
    state.permissionsLoading = true;
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("waits for the briefing, whose headline would replace the computed one", async () => {
    state.briefing = { settled: false, value: null };
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("waits for what is waiting on someone who decides", async () => {
    state.summary.isPending = true;
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("does not wait on a decision summary someone may not see", async () => {
    state.canDecide = false;
    state.summary.isPending = true;
    renderHome();

    expect(await screen.findByText("How can I help today?")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { level: 1 }).querySelector('[data-slot="skeleton"]'),
    ).toBeNull();
  });

  it("keeps the rest of the page, with the ask box as its own skeleton, while it waits", async () => {
    renderHome({ isLoading: true });

    await expectHeadlineSkeleton();
    const heading = screen.getByRole("heading", { level: 1 });
    expect(heading.closest("header")).not.toBeNull();
    expect(heading).toHaveClass("font-display");
    expect(document.querySelector('[aria-busy] [data-slot="skeleton"]')).not.toBeNull();
  });

  it("opens on the headline its figures support once everything is in", async () => {
    renderHome();

    expect(await screen.findByText("2 decisions are waiting on you.")).toBeInTheDocument();
    const heading = screen.getByRole("heading", { level: 1 });
    expect(heading).toHaveTextContent(/^Good (morning|afternoon|evening)$/);
    const second = heading.parentElement?.querySelector('[data-slot="desk-headline"]');
    expect(second).toHaveTextContent("2 decisions are waiting on you.");
    expect(second?.querySelector('[data-slot="skeleton"]')).toBeNull();
  });
});
