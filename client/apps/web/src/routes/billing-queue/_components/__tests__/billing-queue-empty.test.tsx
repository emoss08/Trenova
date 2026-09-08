import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import BillingQueueDetailPane from "../billing-queue-detail-pane";
import { BillingQueueSidebar } from "../billing-queue-sidebar";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  presets: vi.fn(),
}));

vi.mock("../../billing-queue-queries", () => ({
  BILLING_QUEUE_LIST_KEY: "billing-queue-list",
  BILLING_QUEUE_FILTER_PRESETS_KEY: "billing-queue-filter-presets",
  billingQueueFilterPresetsQuery: () => ({
    queryKey: ["billing-queue-filter-presets"],
    queryFn: mocks.presets,
  }),
  billingQueueListQuery: (filters: Record<string, unknown>) => ({
    queryKey: ["billing-queue-list", JSON.stringify(filters)],
    queryFn: () => mocks.list(filters),
  }),
}));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateStatus: vi.fn(), deleteFilterPreset: vi.fn() } },
}));
vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));
vi.mock("@/components/fields/multi-select-field", () => ({
  MultiSelectAutocomplete: () => null,
}));
vi.mock("@/components/fields/autocomplete/autocomplete", () => ({ Autocomplete: () => null }));
vi.mock("@/routes/shipment/_components/comments", () => ({ default: () => null }));
vi.mock("@/components/audit-tab", () => ({ default: () => null }));

function renderWithin(ui: ReactElement, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
          {ui}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("billing queue empty states", () => {
  it("says how the queue fills when nothing is waiting", async () => {
    mocks.list.mockResolvedValue({ results: [], count: 0 });
    mocks.presets.mockResolvedValue([]);
    renderWithin(<BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />);

    expect(await screen.findByText("Nothing waiting")).toBeInTheDocument();
    expect(screen.getByText(/ready to bill/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // An empty list under a filter is a different thing from an empty queue:
  // the way forward is to widen the filter, and the button does exactly that.
  it("offers to clear the filters when they are what emptied the list", async () => {
    mocks.list.mockResolvedValue({ results: [], count: 0 });
    mocks.presets.mockResolvedValue([]);
    const onUrlUpdate = vi.fn();
    const user = userEvent.setup();
    renderWithin(
      <BillingQueueSidebar selectedItemId={null} onSelectItem={vi.fn()} />,
      "?status=OnHold&query=PRO-1",
      onUrlUpdate,
    );

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    const last = onUrlUpdate.mock.calls.at(-1)?.[0] as { searchParams: URLSearchParams };
    expect(last.searchParams.get("status")).toBeNull();
    expect(last.searchParams.get("query")).toBeNull();
  });

  it("draws the review pane it is waiting to become when nothing is open", () => {
    renderWithin(
      <BillingQueueDetailPane
        selectedItemId={null}
        onDocumentSelect={vi.fn()}
        onAutoAdvance={vi.fn()}
      />,
    );

    expect(screen.getByText("Nothing open")).toBeInTheDocument();
    expect(screen.getByText(/press J/)).toBeInTheDocument();
  });
});
