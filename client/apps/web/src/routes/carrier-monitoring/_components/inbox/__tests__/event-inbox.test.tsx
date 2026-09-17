import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import type { FetchCarrierIntelEventInboxArgs } from "@/lib/graphql/carrier-monitoring-table";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { EventInbox } from "../event-inbox";

const mocks = vi.hoisted(() => ({
  fetchInbox: vi.fn(),
  fetchEvent: vi.fn(),
  acknowledge: vi.fn(),
  toastSuccess: vi.fn(),
  toastInfo: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-monitoring-table", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-monitoring-table")>()),
  fetchCarrierIntelEventInbox: mocks.fetchInbox,
  fetchCarrierIntelEventCarrierSummary: vi.fn().mockResolvedValue(null),
}));

vi.mock("@/lib/graphql/carrier-intelligence", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-intelligence")>()),
  acknowledgeCarrierIntelEvents: mocks.acknowledge,
  fetchCarrierIntelEvent: mocks.fetchEvent,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    carrierIntelSettings: {
      settings: () => ({
        queryKey: ["carrierIntelSettings", "settings"],
        queryFn: async () => ({
          carrierIntelRuleCatalog: [
            {
              code: "insurance.bipd_below_required",
              label: "Liability coverage below requirement",
              description: "BIPD on file is below what the carrier must carry.",
            },
          ],
        }),
      }),
      monitoringStatus: () => ({ queryKey: ["carrierIntelSettings", "monitoring-status"] }),
    },
  },
}));

vi.mock("@/lib/graphql/select-options", () => ({
  fetchGraphQLSelectOptions: vi.fn().mockResolvedValue({ results: [], count: 0 }),
}));

vi.mock("@/hooks/use-select-option", () => ({
  useSelectOption: () => ({ option: null, isLoading: false }),
}));

vi.mock("@/hooks/use-media-query", () => ({
  useMediaQuery: () => false,
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, info: mocks.toastInfo, error: vi.fn() },
}));

const NOW = Math.floor(Date.now() / 1000);

function event(overrides: Partial<CarrierIntelEvent>): CarrierIntelEvent {
  return {
    id: "cie_1",
    subjectType: "Carrier",
    subjectId: "car_1",
    carrierId: "car_1",
    dotNumber: "1234567",
    subjectName: "Blue Ridge Freight",
    provider: "CarrierOK",
    source: "SnapshotDiff",
    category: "Safety",
    fieldPath: "safety.rating",
    fieldLabel: "Safety rating",
    ruleCode: null,
    ruleLabel: null,
    severity: "High",
    action: null,
    priorValue: "NotRated",
    currentValue: "Satisfactory",
    summary: "Safety rating changed",
    vendorChangedAt: null,
    detectedAt: NOW - 60,
    status: "Open",
    acknowledgedById: null,
    acknowledgedBy: null,
    acknowledgedAt: null,
    resolvedById: null,
    resolvedBy: null,
    resolvedAt: null,
    resolution: null,
    resolutionNote: null,
    snapshotId: null,
    version: 1,
    createdAt: NOW - 60,
    updatedAt: NOW - 60,
    ...overrides,
  };
}

const EVENTS: CarrierIntelEvent[] = [
  event({ id: "cie_1" }),
  event({
    id: "cie_2",
    subjectName: "Summit Carriers",
    carrierId: "car_2",
    category: "Insurance",
    fieldPath: null,
    fieldLabel: null,
    ruleCode: "insurance.bipd_below_required",
    ruleLabel: "Liability coverage below requirement",
    severity: "Critical",
    priorValue: null,
    currentValue: "BIPD $500,000 is below the $750,000 required",
    summary: "BIPD $500,000 is below the $750,000 required",
    detectedAt: NOW - 120,
  }),
  event({
    id: "cie_3",
    subjectName: "Coastal Haulers",
    carrierId: "car_3",
    fieldPath: "insurance.bipdOnFile",
    fieldLabel: "BIPD coverage on file",
    category: "Insurance",
    priorValue: "750000",
    currentValue: "500000",
    detectedAt: NOW - 180,
  }),
];

function lastInboxArgs(): FetchCarrierIntelEventInboxArgs {
  const calls = mocks.fetchInbox.mock.calls;
  return calls[calls.length - 1][0] as FetchCarrierIntelEventInboxArgs;
}

function renderInbox({
  canUpdate = true,
  searchParams,
}: { canUpdate?: boolean; searchParams?: string } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <NuqsTestingAdapter
          hasMemory
          resetUrlUpdateQueueOnMount={false}
          searchParams={searchParams}
        >
          <MemoryRouter>{children}</MemoryRouter>
        </NuqsTestingAdapter>
      </QueryClientProvider>
    );
  }
  return render(<EventInbox canUpdate={canUpdate} scopeCounts={{ attention: 3 }} />, {
    wrapper: Wrapper,
  });
}

function row(title: RegExp | string) {
  return screen
    .getAllByRole("option")
    .find((node) => node.hasAttribute("data-event-id") && within(node).queryByText(title));
}

function focusedRowId(): string | null {
  return (
    screen
      .getAllByRole("option")
      .find((node) => node.getAttribute("data-focused") === "true")
      ?.getAttribute("data-event-id") ?? null
  );
}

beforeEach(() => {
  mocks.fetchInbox.mockResolvedValue({ events: EVENTS, hasNextPage: false, endCursor: null });
  mocks.acknowledge.mockResolvedValue(1);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("EventInbox rows", () => {
  it("titles rows by rule label or by formatted field change, never by raw code", async () => {
    renderInbox();

    expect(await screen.findByText("Safety rating: Not rated → Satisfactory")).toBeInTheDocument();
    expect(await screen.findByText("Liability coverage below requirement")).toBeInTheDocument();
    expect(
      screen.getByText("BIPD coverage on file: $750,000.00 → $500,000.00"),
    ).toBeInTheDocument();
    expect(screen.queryByText("insurance.bipd_below_required")).toBeNull();
  });
});

describe("EventInbox deep link", () => {
  it("opens an event from the list without fetching it again", async () => {
    renderInbox({ searchParams: "?event=cie_3" });

    const dialog = await screen.findByRole("dialog");
    expect(
      within(dialog).getAllByRole("heading", {
        name: "BIPD coverage on file: $750,000.00 → $500,000.00",
      }).length,
    ).toBeGreaterThan(0);
    expect(mocks.fetchEvent).not.toHaveBeenCalled();
  });

  it("fetches an event outside the current list and shows who acted on it", async () => {
    mocks.fetchEvent.mockResolvedValue(
      event({
        id: "cie_9",
        subjectName: "Prairie Line Transport",
        status: "Resolved",
        acknowledgedAt: NOW - 50,
        acknowledgedById: "usr_1",
        acknowledgedBy: { id: "usr_1", name: "Dana Whitfield" },
        resolvedAt: NOW - 40,
        resolvedById: "usr_2",
        resolvedBy: { id: "usr_2", name: "Marcus Lee" },
        resolution: "NoActionRequired",
      }),
    );
    renderInbox({ searchParams: "?event=cie_9" });

    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Prairie Line Transport")).toBeInTheDocument();
    expect(within(dialog).getByText("Dana Whitfield")).toBeInTheDocument();
    expect(within(dialog).getByText("Marcus Lee")).toBeInTheDocument();
    expect(mocks.fetchEvent).toHaveBeenCalledWith("cie_9", expect.anything());
  });

  it("says calmly when the linked event does not exist", async () => {
    mocks.fetchEvent.mockResolvedValue(null);
    renderInbox({ searchParams: "?event=cie_missing" });

    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Change not found")).toBeInTheDocument();
  });
});

describe("EventInbox keyboard", () => {
  it("moves the focus with j and k and stays inside the list", async () => {
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    expect(focusedRowId()).toBeNull();

    fireEvent.keyDown(window, { key: "j" });
    expect(focusedRowId()).toBe("cie_1");

    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "ArrowDown" });
    expect(focusedRowId()).toBe("cie_3");

    fireEvent.keyDown(window, { key: "j" });
    expect(focusedRowId()).toBe("cie_3");

    fireEvent.keyDown(window, { key: "k" });
    expect(focusedRowId()).toBe("cie_2");
  });

  it("selects with x, acknowledges the selection with e, and clears it with Escape", async () => {
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "x" });
    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "x" });

    expect(await screen.findByText("2 selected")).toBeInTheDocument();
    expect(row(/Safety rating/)?.getAttribute("data-selected")).toBe("true");

    fireEvent.keyDown(window, { key: "x" });
    expect(await screen.findByText("1 selected")).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(screen.queryByText("1 selected")).toBeNull());

    fireEvent.keyDown(window, { key: "x" });
    fireEvent.keyDown(window, { key: "k" });
    fireEvent.keyDown(window, { key: "x" });
    expect(await screen.findByText("2 selected")).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "e" });

    await waitFor(() => expect(mocks.acknowledge).toHaveBeenCalledWith(["cie_1", "cie_2"]));
    await waitFor(() => expect(screen.queryByText("2 selected")).toBeNull());
  });

  it("acknowledges the focused event with e when nothing is selected", async () => {
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "e" });

    await waitFor(() => expect(mocks.acknowledge).toHaveBeenCalledWith(["cie_2"]));
  });

  it("does not acknowledge without update permission", async () => {
    renderInbox({ canUpdate: false });
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "e" });

    expect(mocks.acknowledge).not.toHaveBeenCalled();
  });

  it("opens the focused event with Enter and ignores keys typed into the search box", async () => {
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    fireEvent.keyDown(screen.getByRole("searchbox", { name: "Search events" }), { key: "j" });
    expect(focusedRowId()).toBeNull();

    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "j" });
    fireEvent.keyDown(window, { key: "Enter" });

    const dialog = await screen.findByRole("dialog");
    expect(
      within(dialog).getAllByRole("heading", { name: "Liability coverage below requirement" })
        .length,
    ).toBeGreaterThan(0);
    expect(
      within(dialog).getByText("BIPD on file is below what the carrier must carry."),
    ).toBeInTheDocument();
  });
});

describe("EventInbox filters", () => {
  it("adds a severity chip that narrows the query and clears it again", async () => {
    const user = userEvent.setup();
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    expect(lastInboxArgs().filter).toEqual({ statuses: ["Open"] });

    await user.click(screen.getByRole("button", { name: "Filter" }));
    await user.click(await screen.findByRole("option", { name: "Severity" }));
    await user.click(await screen.findByRole("option", { name: "Critical" }));

    await waitFor(() =>
      expect(lastInboxArgs().filter).toEqual({ statuses: ["Open"], severities: ["Critical"] }),
    );

    await user.keyboard("{Escape}");
    const clear = await screen.findByRole("button", { name: "Clear filter" });
    await user.click(clear);

    await waitFor(() => expect(lastInboxArgs().filter).toEqual({ statuses: ["Open"] }));
    expect(screen.queryByRole("button", { name: "Clear filter" })).toBeNull();
  });

  it("sends the source filter as a field filter and the search as the query", async () => {
    const user = userEvent.setup();
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    await user.type(screen.getByRole("searchbox", { name: "Search events" }), "summit");
    await waitFor(() => expect(lastInboxArgs().query).toBe("summit"));

    await user.click(screen.getByRole("button", { name: "Filter" }));
    await user.click(await screen.findByRole("option", { name: "Source" }));
    await user.click(await screen.findByRole("option", { name: "Rule evaluation" }));

    await waitFor(() =>
      expect(lastInboxArgs().fieldFilters).toEqual([
        { field: "source", operator: "in", value: ["RuleEvaluation"] },
      ]),
    );
  });

  it("offers a way out when filters empty the list", async () => {
    const user = userEvent.setup();
    renderInbox();
    await screen.findByText("Safety rating: Not rated → Satisfactory");

    mocks.fetchInbox.mockResolvedValue({ events: [], hasNextPage: false, endCursor: null });
    await user.type(screen.getByRole("searchbox", { name: "Search events" }), "nobody");

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    await waitFor(() => expect(lastInboxArgs().query).toBeNull());
  });
});
