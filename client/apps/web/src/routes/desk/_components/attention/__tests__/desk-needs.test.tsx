import type { DeskAttention, DeskAttentionFilter } from "@/lib/graphql/desk-attention";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DecisionFilters } from "../../decisions/decision-filters";
import { DeskNeeds } from "../desk-needs";
import type { RecordLens } from "../lens";
import type { NeedsFilters } from "../needs-filters";

const can = vi.hoisted(() => ({ watch: true, decide: true }));
const asked = vi.hoisted(() => ({ lists: [] as DeskAttentionFilter[] }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" ? can.decide : can.watch,
    isLoading: false,
  }),
}));

const ATTENTION: DeskAttention = {
  groups: [],
  counts: { everything: 9, watchtower: 5, triage: 0, decisions: 4 },
  kinds: [],
  hasMore: false,
  seenAt: 0,
  seesEveryone: true,
};

vi.mock("@/lib/queries/desk-attention", () => ({
  deskAttention: {
    list: (filter: DeskAttentionFilter) => ({
      queryKey: ["deskAttention", "list", filter],
      queryFn: async () => {
        asked.lists.push(filter);
        return ATTENTION;
      },
    }),
  },
}));

vi.mock("@/lib/queries/watchtower", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/watchtower")>()),
  watchtower: {
    cleared: (since: number) => ({
      queryKey: ["watchtower", "cleared", since],
      queryFn: async () => 2,
    }),
  },
}));

vi.mock("../needs-attention", () => ({
  NeedsAttention: ({
    lens,
    shown,
    filters,
    keysPaused,
  }: {
    lens: RecordLens;
    shown: boolean;
    filters: NeedsFilters;
    keysPaused: boolean;
  }) => (
    <div
      data-testid="by-record"
      data-lens={lens}
      data-shown={shown}
      data-search={filters.search}
      data-paused={keysPaused}
    />
  ),
}));

vi.mock("../../decisions/use-pending-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../decisions/use-pending-decisions")>()),
  usePendingDecisionSummary: (enabled: boolean) => ({
    data: enabled
      ? {
          total: 7,
          byAgent: [
            { agentDefinitionId: "agd_billing", agentName: "Billing clerk", count: 5 },
            { agentDefinitionId: "agd_dispatch", agentName: "Dispatch planner", count: 2 },
          ],
          byTool: [
            { toolName: "reassign_move", count: 4 },
            { toolName: "plan", count: 3 },
          ],
          oldestAt: null,
        }
      : undefined,
  }),
  usePendingDecisions: (_filter: unknown, enabled: boolean) => ({
    data: enabled ? { nodes: [], totalCount: 7 } : undefined,
  }),
  useRecentDecisions: (_since: number, enabled: boolean) => ({
    data: enabled ? [{ id: "adec_1" }] : undefined,
  }),
}));

vi.mock("../../decisions/decision-flow", () => ({
  DecisionFlow: ({
    shown,
    filter,
    keysPaused,
  }: {
    shown: boolean;
    filter: DecisionFilters;
    keysPaused: boolean;
  }) => (
    <div
      data-testid="decisions"
      data-shown={shown}
      data-filter={`${filter.search}|${filter.agentId}|${filter.tool}`}
      data-paused={keysPaused}
    />
  ),
}));

afterEach(() => {
  can.watch = true;
  can.decide = true;
  asked.lists = [];
});

function renderAt(search: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/desk/needs${search}`]}>
        <NuqsTestingAdapter searchParams={search}>
          <DeskNeeds />
        </NuqsTestingAdapter>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const tab = (name: string | RegExp) => screen.getByRole("tab", { name });

describe("DeskNeeds, one frame for every lens", () => {
  it("lists by record under Everything and Watchtower, with what waits in each lens on the tabs", async () => {
    renderAt("?lens=watchtower");

    expect(screen.getByTestId("by-record")).toHaveAttribute("data-lens", "Watchtower");
    expect(await screen.findByRole("tab", { name: "Watchtower 5" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(tab("Everything 9")).toBeInTheDocument();
    expect(tab("Decisions 4")).toBeInTheDocument();
    expect(
      await screen.findByRole("progressbar", { name: "2 of 11 cleared today" }),
    ).toBeInTheDocument();
  });

  it("reads nothing of Decisions until the lens is opened", async () => {
    renderAt("");

    await screen.findByRole("tab", { name: "Everything 9" });
    expect(screen.queryByTestId("decisions")).toBeNull();
    expect(tab("Decisions 4")).toHaveAttribute("aria-selected", "false");
  });

  it("keeps the same head on the page when the lens changes, and moves only its marker", async () => {
    renderAt("");
    const tabs = screen.getByRole("tablist", { name: "Lens" });
    const search = screen.getByRole("textbox", { name: "Search what needs you" });
    await screen.findByRole("tab", { name: "Everything 9" });

    fireEvent.click(tab("Decisions 4"));

    expect(await screen.findByTestId("decisions")).toHaveAttribute("data-shown", "true");
    expect(screen.getByRole("tablist", { name: "Lens" })).toBe(tabs);
    expect(screen.getByRole("textbox", { name: "Search what needs you" })).toBe(search);
    expect(tab("Decisions 4")).toHaveAttribute("aria-selected", "true");
    expect(tab("Everything 9")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByRole("progressbar", { name: "1 of 8 decided" })).toBeInTheDocument();
  });

  it("keeps the lens that was left, hidden, and shows it again as it was", async () => {
    renderAt("?lens=watchtower");
    const byRecord = screen.getByTestId("by-record");

    fireEvent.click(await screen.findByRole("tab", { name: "Decisions 4" }));
    await screen.findByTestId("decisions");

    expect(screen.getByTestId("by-record")).toBe(byRecord);
    expect(byRecord).not.toBeVisible();
    expect(byRecord).toHaveAttribute("data-shown", "false");
    expect(byRecord).toHaveAttribute("data-lens", "Watchtower");

    fireEvent.click(tab("Watchtower 5"));

    await waitFor(() => expect(byRecord).toBeVisible());
    expect(byRecord).toHaveAttribute("data-shown", "true");
    expect(screen.getByTestId("decisions")).not.toBeVisible();
  });

  it("reads a lens ahead when the pointer reaches its tab", async () => {
    renderAt("");
    await screen.findByRole("tab", { name: "Watchtower 5" });
    asked.lists = [];

    fireEvent.pointerEnter(tab("Watchtower 5"));

    await waitFor(() => expect(asked.lists.map((filter) => filter.lens)).toEqual(["Watchtower"]));
    expect(screen.getByTestId("by-record")).toHaveAttribute("data-lens", "Everything");
  });

  it("opens on Decisions, counted from the queue, for someone who may not read the Watchtower", async () => {
    can.watch = false;
    renderAt("");

    expect(await screen.findByTestId("decisions")).toBeInTheDocument();
    expect(screen.queryByTestId("by-record")).toBeNull();
    expect(screen.getAllByRole("tab")).toHaveLength(1);
    expect(tab("Decisions 7")).toBeInTheDocument();
    expect(asked.lists).toEqual([]);
  });

  it("narrows the decision queue to the words, the agent and the kind of change the address names", async () => {
    renderAt("?lens=decisions&q=acme&agent=agd_billing&change=plan");

    expect(await screen.findByTestId("decisions")).toHaveAttribute(
      "data-filter",
      "acme|agd_billing|plan",
    );
    expect(screen.getByRole("textbox", { name: "Search what needs you" })).toHaveValue("acme");
    expect(screen.getByRole("group", { name: "Filters" })).toHaveTextContent(
      "Billing clerkPlansClear",
    );
    expect(screen.getByRole("button", { name: "Filter, 2 on" })).toBeInTheDocument();
  });

  it("takes one decision filter off by its token and leaves the other and the search", async () => {
    renderAt("?lens=decisions&q=acme&agent=agd_billing&change=reassign_move");
    await screen.findByTestId("decisions");

    fireEvent.click(screen.getByRole("button", { name: "Remove the filter Billing clerk" }));

    await waitFor(() =>
      expect(screen.getByTestId("decisions")).toHaveAttribute("data-filter", "acme||reassign_move"),
    );
  });

  it("searches every lens by the same words, which stay when the lens changes", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderAt("");
    await screen.findByRole("tab", { name: "Decisions 4" });

    fireEvent.change(screen.getByRole("textbox", { name: "Search what needs you" }), {
      target: { value: "acme" },
    });
    await act(() => vi.advanceTimersByTimeAsync(300));
    vi.useRealTimers();
    await waitFor(() =>
      expect(screen.getByTestId("by-record")).toHaveAttribute("data-search", "acme"),
    );

    fireEvent.click(tab(/^Decisions/u));

    expect(await screen.findByTestId("decisions")).toHaveAttribute("data-filter", "acme||");
    expect(screen.getByRole("textbox", { name: "Search what needs you" })).toHaveValue("acme");
  });

  it("offers whose cases under a lens that lists by record, and not under Decisions", async () => {
    renderAt("");

    expect(await screen.findByRole("group", { name: "Whose cases" })).toBeInTheDocument();

    fireEvent.click(tab("Decisions 4"));
    await screen.findByTestId("decisions");

    expect(screen.queryByRole("group", { name: "Whose cases" })).toBeNull();
  });

  it("stands the lens's keys down while the filter is open", async () => {
    renderAt("");
    await screen.findByRole("tab", { name: "Everything 9" });
    expect(screen.getByTestId("by-record")).toHaveAttribute("data-paused", "false");

    act(() => void fireEvent.keyDown(document.body, { key: "f" }));

    await waitFor(() =>
      expect(screen.getByTestId("by-record")).toHaveAttribute("data-paused", "true"),
    );
  });

  it("falls back to Everything when the address asks for a lens the person may not read", () => {
    can.decide = false;
    renderAt("?lens=decisions");

    expect(screen.getByTestId("by-record")).toHaveAttribute("data-lens", "Everything");
    expect(screen.getByTestId("by-record")).toHaveAttribute("data-shown", "true");
    expect(screen.queryByRole("tab", { name: /Decisions/u })).toBeNull();
  });
});
