import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueRow } from "@/lib/graphql/billing-queue";
import type { BillingQueueLane } from "@trenova/shared/types/billing-queue";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import { BillingQueueSidebar } from "../billing-queue-sidebar";

function lanePage(results: BillingQueueRow[], count = results.length, next: string | null = null) {
  return {
    edges: results.map((node) => ({ cursor: node.id, node })),
    totalCount: count,
    totalCountExact: true,
    pageInfo: { hasNextPage: next !== null, endCursor: next },
  };
}

const mocks = vi.hoisted(() => ({
  lane: vi.fn(),
  stats: vi.fn(),
  presets: vi.fn(),
  assist: vi.fn(),
  recommendations: vi.fn(),
  composeSearch: vi.fn(),
}));

vi.mock("../../billing-queue-queries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../billing-queue-queries")>();
  return {
    ...actual,
    billingQueueFilterPresetsQuery: () => ({
      queryKey: ["billing-queue-filter-presets"],
      queryFn: mocks.presets,
    }),
    billingQueueStatsQuery: () => ({ queryKey: ["billingQueue", "stats"], queryFn: mocks.stats }),
    billingQueueLaneQuery: (lane: BillingQueueLane, filters: Record<string, unknown>) => ({
      queryKey: ["billing-queue-list", "lane", lane, JSON.stringify(filters)],
      queryFn: ({ pageParam }: { pageParam: string | null }) =>
        mocks.lane(lane, filters, pageParam),
      initialPageParam: null,
      getNextPageParam: (page: { pageInfo: { hasNextPage: boolean; endCursor: string | null } }) =>
        page.pageInfo.hasNextPage ? page.pageInfo.endCursor : undefined,
    }),
  };
});

vi.mock("@/services/api", () => ({
  apiService: {
    billingQueueService: {
      getAssist: mocks.assist,
      getRecommendations: mocks.recommendations,
      composeSearch: mocks.composeSearch,
      deleteFilterPreset: vi.fn(),
    },
  },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: Object.assign(vi.fn(), { error: vi.fn(), success: vi.fn(), info: vi.fn() }),
}));
vi.mock("@/components/fields/multi-select-field", () => ({
  MultiSelectAutocomplete: () => null,
}));
vi.mock("@/components/fields/autocomplete/autocomplete", () => ({ Autocomplete: () => null }));
vi.mock("@/routes/shipment/_components/comments", () => ({ default: () => null }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));

const NOW = Math.floor(Date.now() / 1000);

function queued(id: string, lane: BillingQueueLane, extra: Partial<BillingQueueRow> = {}) {
  return {
    id,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: `shp_${id}`,
    billToCustomerId: "cus_acme",
    allocatedTotalAmount: 1200,
    status: "InReview",
    lane,
    billType: "Invoice",
    version: 1,
    createdAt: NOW - 3600,
    updatedAt: NOW,
    packetDocumentIds: [],
    shipment: {
      id: `shp_${id}`,
      proNumber: `PRO-${id}`,
      totalChargeAmount: 1200,
      customerId: "cus_acme",
      customer: { id: "cus_acme", name: "Acme Foods", code: "ACME" },
    },
    billToCustomer: { id: "cus_acme", name: "Acme Foods", code: "ACME" },
    ...extra,
  } as unknown as BillingQueueRow;
}

function stats(lanes: Partial<Record<BillingQueueLane, number>>) {
  const entries = Object.entries(lanes).map(([lane, count]) => ({
    lane,
    count,
    total: String((count ?? 0) * 1200),
  }));
  return {
    lanes: entries,
    openCount: entries.reduce((sum, entry) => sum + (entry.count ?? 0), 0),
    openTotal: "0",
    pulse: { approvedTodayCount: 4, approvedTodayTotal: "9600" },
  };
}

function renderWithin(ui: ReactElement, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate} hasMemory>
          {ui}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

let restoreLayout = () => {};

beforeEach(() => {
  // The lanes are virtualized; a viewport gives the window rows to mount into.
  restoreLayout = stubLayout(720);
  mocks.presets.mockResolvedValue([]);
  mocks.assist.mockResolvedValue({ review: false, brief: false, search: false });
  mocks.recommendations.mockResolvedValue([]);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
});

describe("billing queue sidebar", () => {
  it("says the queue is caught up, with today's approvals, when nothing is open", async () => {
    mocks.stats.mockResolvedValue(stats({}));
    mocks.lane.mockResolvedValue(lanePage([]));
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    expect(await screen.findByText("You're caught up")).toBeInTheDocument();
    expect(screen.getByText("4 items approved today · $9,600.00")).toBeInTheDocument();
  });

  it("draws each open lane in working order with the issue its rows wait on", async () => {
    mocks.stats.mockResolvedValue(stats({ NeedsYou: 1, Ready: 1, Held: 2 }));
    mocks.lane.mockImplementation(async (lane: BillingQueueLane) => {
      if (lane === "NeedsYou") {
        return lanePage([
          queued("1", "NeedsYou", {
            headline: {
              code: "PODMissing",
              checkKey: "pod",
              summary: "Signed POD missing",
              open: 1,
              waiting: false,
            },
          }),
        ]);
      }
      if (lane === "Ready") return lanePage([queued("2", "Ready")]);
      return lanePage([]);
    });
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    expect(await screen.findByText("Signed POD missing")).toBeInTheDocument();
    const headings = screen
      .getAllByRole("button", { expanded: true })
      .map((button) => button.textContent);
    expect(headings).toEqual(["Needs you1$1,200.00", "Ready to approve1$1,200.00"]);
    expect(screen.getByRole("button", { name: /On hold/, expanded: false })).toBeInTheDocument();
    // A folded lane reads its count from the stats and fetches nothing.
    expect(mocks.lane.mock.calls.map(([lane]) => lane)).not.toContain("Held");
    expect(mocks.lane.mock.calls.map(([lane]) => lane)).not.toContain("Approved");
  });

  it("reads the next page of each open lane as its end is reached, not only the last lane's", async () => {
    mocks.stats.mockResolvedValue(stats({ NeedsYou: 3, Ready: 1 }));
    mocks.lane.mockImplementation(
      async (lane: BillingQueueLane, _filters: unknown, after: string | null) => {
        if (lane === "NeedsYou") {
          return after === null
            ? lanePage([queued("1", "NeedsYou"), queued("2", "NeedsYou")], 3, "c-2")
            : lanePage([queued("3", "NeedsYou")], 3);
        }
        if (lane === "Ready") return lanePage([queued("9", "Ready")]);
        return lanePage([]);
      },
    );
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    expect(await screen.findByText("PRO-3")).toBeInTheDocument();
    expect(mocks.lane).toHaveBeenCalledWith("NeedsYou", expect.anything(), "c-2");
    expect(screen.getByText("PRO-9")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Show .* more/u })).not.toBeInTheDocument();
  });

  it("mounts only the rows near the viewport when a lane runs long", async () => {
    mocks.stats.mockResolvedValue(stats({ Ready: 200 }));
    mocks.lane.mockImplementation(async (lane: BillingQueueLane) =>
      lane === "Ready"
        ? lanePage(
            Array.from({ length: 200 }, (_, index) => queued(String(index + 1), "Ready")),
            200,
          )
        : lanePage([]),
    );
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    expect(await screen.findByRole("button", { name: /PRO-1\b/ })).toBeInTheDocument();
    const mounted = screen.getAllByRole("checkbox", { name: /^Select PRO-/ });
    expect(mounted.length).toBeGreaterThan(0);
    expect(mounted.length).toBeLessThan(60);
    expect(screen.queryByRole("checkbox", { name: "Select PRO-200" })).not.toBeInTheDocument();
    expect(screen.getByText("200 items shown")).toBeInTheDocument();
  });

  it("offers to clear the filters when they are what emptied the list", async () => {
    mocks.stats.mockResolvedValue(stats({ NeedsYou: 3 }));
    mocks.lane.mockResolvedValue(lanePage([]));
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />,
      "?status=OnHold&query=PRO-1&lanes=Ready",
      onUrlUpdate,
    );

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("status")).toBeNull();
    expect(last.searchParams.get("query")).toBeNull();
    expect(last.searchParams.get("lanes")).toBeNull();
  });

  it("waits for Enter on a sentence when Nova reads search, and applies what Nova read", async () => {
    mocks.assist.mockResolvedValue({ review: true, brief: true, search: true });
    mocks.stats.mockResolvedValue(stats({ Ready: 1 }));
    mocks.lane.mockResolvedValue(lanePage([]));
    mocks.composeSearch.mockResolvedValue({
      lanes: ["Ready"],
      search: "",
      payerId: "cus_acme",
      payerName: "Acme Foods",
      minAmount: "5000",
      maxAmount: null,
      createdBefore: NOW - 3 * 86400,
      unresolved: "",
    });
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />,
      "",
      onUrlUpdate,
    );

    const box = await screen.findByRole("textbox", { name: "Search or ask Nova" });
    await user.type(box, "acme over 5k ready");
    expect(await screen.findByText("Press Enter to ask Nova")).toBeInTheDocument();
    for (const [, filters] of mocks.lane.mock.calls) {
      expect((filters as { search: string }).search).not.toBe("acme over 5k ready");
    }

    await user.keyboard("{Enter}");
    await waitFor(() =>
      expect(mocks.composeSearch).toHaveBeenCalledWith(
        "acme over 5k ready",
        expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/u),
      ),
    );
    await waitFor(() => {
      const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
      expect(last.searchParams.get("lanes")).toBe("Ready");
      expect(last.searchParams.get("payer")).toBe("cus_acme");
      expect(last.searchParams.get("minAmount")).toBe("5000");
      expect(last.searchParams.get("createdBefore")).toBe(String(NOW - 3 * 86400));
      expect(last.searchParams.get("query")).toBeNull();
    });
    expect(await screen.findByRole("button", { name: "Remove filter Acme Foods" })).toBeVisible();
  });

  it("applies the shipment customer and the dates Nova read from a sentence", async () => {
    mocks.assist.mockResolvedValue({ review: true, brief: true, search: true });
    mocks.stats.mockResolvedValue(stats({ Ready: 1 }));
    mocks.lane.mockResolvedValue(lanePage([]));
    mocks.composeSearch.mockResolvedValue({
      lanes: [],
      search: "",
      shipmentCustomerId: "cus_globex",
      shipmentCustomerName: "Globex",
      delivered: { lastDays: 7 },
      pickedUp: { from: "2026-10-19", to: "2026-10-25" },
      queued: null,
      unresolved: "",
    });
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />,
      "",
      onUrlUpdate,
    );

    const box = await screen.findByRole("textbox", { name: "Search or ask Nova" });
    await user.type(box, "globex shipments delivered this past week{Enter}");

    await waitFor(() => {
      const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
      expect(last.searchParams.get("shipmentCustomer")).toBe("cus_globex");
      expect(last.searchParams.get("delivered")).toBe("last:7");
      expect(last.searchParams.get("pickedUp")).toMatch(/^range:\d+:\d+$/u);
      expect(last.searchParams.get("queued")).toBeNull();
    });
    expect(
      await screen.findByRole("button", { name: "Remove filter Shipments for Globex" }),
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Remove filter Delivered in the last 7 days" }),
    ).toBeVisible();
  });

  it("shimmers while Nova reads a sentence, then shows nothing of it once read", async () => {
    mocks.assist.mockResolvedValue({ review: true, brief: true, search: true });
    mocks.stats.mockResolvedValue(stats({ Ready: 1 }));
    mocks.lane.mockResolvedValue(lanePage([]));
    let answer: (intent: unknown) => void = () => {};
    mocks.composeSearch.mockReturnValue(
      new Promise((resolve) => {
        answer = resolve;
      }),
    );
    const user = userEvent.setup();
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    const box = await screen.findByRole("textbox", { name: "Search or ask Nova" });
    await user.type(box, "acme over 5k ready{Enter}");

    const reading = await screen.findByText("Nova is reading that…");
    expect(reading).toHaveClass("bg-clip-text", "text-transparent");

    answer({
      lanes: ["Ready"],
      search: "",
      payerId: null,
      payerName: null,
      minAmount: null,
      maxAmount: null,
      createdBefore: null,
      unresolved: "",
    });
    await waitFor(() =>
      expect(screen.queryByText("Nova is reading that…")).not.toBeInTheDocument(),
    );
  });

  it("says nothing matches when the queue lists no item to open", () => {
    renderWithin(
      <BillingQueueDetailPane
        selectedItemId={null}
        onDocumentSelect={vi.fn()}
        onAutoAdvance={vi.fn()}
      />,
    );

    expect(screen.getByText("Nothing to open")).toBeInTheDocument();
    expect(
      screen.getByText("No item in the queue matches what you're looking at."),
    ).toBeInTheDocument();
  });
});
