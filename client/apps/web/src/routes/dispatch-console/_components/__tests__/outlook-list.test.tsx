import type {
  DispatchForecastQuality,
  DispatchFreightOutlook,
  DispatchFreightWindow,
} from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { formatUnixDayTimeRange } from "@trenova/shared/lib/date";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OutlookList } from "../outlook-list";

const mocks = vi.hoisted(() => ({
  outlook: vi.fn(),
  outlookInput: vi.fn(),
  allowed: true,
}));

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    freightOutlook: (variables: { hours: number; limit: number }) => {
      mocks.outlookInput(variables);
      return { queryKey: ["test-freight-outlook", variables] as const, queryFn: mocks.outlook };
    },
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

// DispatchForecastQuality and DispatchFreightWindow as dispatch_console.graphqls defines them.
const ready: DispatchForecastQuality = {
  status: "Ready",
  historyWeeks: 26,
  historyLoads: 980,
  weeksTested: 8,
  loadsTested: 310,
  intervalCoverage: 0.82,
  meanAbsoluteError: 1.42,
  naiveMeanAbsoluteError: 2.1,
  skill: 0.324,
  fittedAt: 1_793_577_600,
};

const START = 4_102_444_800;
const HOUR = 3_600;

const dallas: DispatchFreightWindow = {
  areaLabel: "Dallas, TX",
  windowStart: START,
  windowEnd: START + 12 * HOUR,
  typicalLow: 3,
  typicalHigh: 5,
  likelyLow: 2,
  likelyHigh: 6,
  booked: 1,
  expectedUnbooked: 3.36,
  driversExpected: 1,
  shortfall: 2.36,
  valueCents: 31_000,
  customers: [
    { customerId: "cus_1", name: "Acme Foods", expectedLoads: 3 },
    { customerId: "cus_2", name: "Red River Grain", expectedLoads: 1.4 },
  ],
  explanation: "Near Dallas, TX: 3 to 5 loads usually appear; 1 booked and 1 driver expected.",
};

// Worth more per driver but less short: drawn second because the server ranks it second.
const memphis: DispatchFreightWindow = {
  areaLabel: "Memphis, TN",
  windowStart: START + 4 * HOUR,
  windowEnd: START + 16 * HOUR,
  typicalLow: 2,
  typicalHigh: 2,
  likelyLow: 1,
  likelyHigh: 4,
  booked: 0,
  expectedUnbooked: 2,
  driversExpected: 2,
  shortfall: 0.3,
  valueCents: 52_000,
  customers: [],
  explanation: "Near Memphis, TN: 2 loads usually appear; 2 drivers expected.",
};

function outlook(overrides: Partial<DispatchFreightOutlook> = {}): DispatchFreightOutlook {
  return { quality: ready, generatedAt: START - HOUR, windows: [dallas, memphis], ...overrides };
}

function renderOutlook() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <OutlookList />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const collapsed = (text: string) => text.replace(/\s+/g, " ");

describe("OutlookList", () => {
  beforeEach(() => {
    mocks.allowed = true;
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("asks for the next 48 hours, at most 20 windows, and draws its outline while they load", () => {
    mocks.outlook.mockImplementation(() => new Promise<never>(() => {}));

    renderOutlook();

    expect(mocks.outlookInput).toHaveBeenCalledWith({ hours: 48, limit: 20 });
    expect(screen.getByLabelText("Loading the freight outlook")).toHaveAttribute(
      "aria-busy",
      "true",
    );
  });

  it("lists each window in the server's order with its loads, drivers, shortfall and value", async () => {
    mocks.outlook.mockResolvedValue(outlook());

    renderOutlook();

    const list = await screen.findByRole("list", { name: "Freight outlook" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);

    expect(rows[0]).toHaveTextContent("Dallas, TX");
    expect(rows[0]).toHaveTextContent("Acme Foods and Red River Grain");
    expect(rows[0]).toHaveTextContent(
      collapsed(formatUnixDayTimeRange(dallas.windowStart, dallas.windowEnd)),
    );
    expect(rows[0]).toHaveTextContent("2–6");
    expect(rows[0]).toHaveTextContent("1 booked");
    expect(rows[0]).toHaveTextContent("$310");
    expect(within(rows[0]).getByText("2.4")).toHaveClass("text-warning-foreground");

    expect(rows[1]).toHaveTextContent("Memphis, TN");
    expect(rows[1]).toHaveTextContent("No customer stands out");
    expect(rows[1]).toHaveTextContent("0 booked");
    expect(rows[1]).toHaveTextContent("$520");
    expect(within(rows[1]).getByText("0.3")).not.toHaveClass("text-warning-foreground");
  });

  it("opens a window's explanation from its row, by click or keyboard", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(outlook());

    renderOutlook();

    const list = await screen.findByRole("list", { name: "Freight outlook" });
    const [dallasRow] = within(list).getAllByRole("button");
    expect(dallasRow).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("Usually 3–5 loads in this window.")).not.toBeVisible();
    expect(screen.queryByText(dallas.explanation)).toBeNull();

    dallasRow.focus();
    await user.keyboard("{Enter}");

    expect(dallasRow).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Usually 3–5 loads in this window.")).toBeVisible();
    expect(
      screen.getByText("Mostly from Acme Foods (about 3) and Red River Grain (about 1.4)."),
    ).toBeVisible();
    expect(
      screen.getByText(
        "3.4 still expected to be booked; one more driver ending here is worth $310 to the plan.",
      ),
    ).toBeVisible();

    await user.click(dallasRow);
    expect(dallasRow).toHaveAttribute("aria-expanded", "false");
  });

  it("says nothing more is expected and up to the usual high when the range starts at none", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(
      outlook({
        windows: [{ ...dallas, typicalLow: 0, typicalHigh: 1, expectedUnbooked: 0.01 }],
      }),
    );

    renderOutlook();

    const list = await screen.findByRole("list", { name: "Freight outlook" });
    await user.click(within(list).getAllByRole("button")[0]);

    expect(screen.getByText("Usually up to 1 load in this window.")).toBeVisible();
    expect(
      screen.getByText(
        "Nothing more is expected to be booked; one more driver ending here is worth $310 to the plan.",
      ),
    ).toBeVisible();
  });

  it("says no loads when the window usually brings none", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockResolvedValue(
      outlook({ windows: [{ ...dallas, typicalLow: 0, typicalHigh: 0 }] }),
    );

    renderOutlook();

    const list = await screen.findByRole("list", { name: "Freight outlook" });
    await user.click(within(list).getAllByRole("button")[0]);

    expect(screen.getByText("Usually no loads in this window.")).toBeVisible();
  });

  it("explains a forecast that is not ready instead of listing its windows", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        quality: { ...ready, status: "InsufficientHistory", historyWeeks: 5, historyLoads: 120 },
      }),
    );

    renderOutlook();

    expect(await screen.findByText("Freight forecast: Not enough history yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        "The forecast needs 8 weeks and 200 loads of history; it has 5 weeks and 120 loads.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "See the forecast in Dispatch controls" }),
    ).toHaveAttribute("href", "/admin/dispatch-controls/");
    expect(screen.queryByRole("list", { name: "Freight outlook" })).toBeNull();
    expect(screen.queryByText("Dallas, TX")).toBeNull();
  });

  it("says a forecast that was never fitted is not fitted yet", async () => {
    mocks.outlook.mockResolvedValue(
      outlook({
        quality: { ...ready, status: "InsufficientHistory", fittedAt: null },
        windows: [],
      }),
    );

    renderOutlook();

    expect(await screen.findByText("Freight forecast: Not fitted yet")).toBeInTheDocument();
  });

  it("offers no link to Dispatch controls to someone who may not read them", async () => {
    mocks.allowed = false;
    mocks.outlook.mockResolvedValue(outlook({ quality: { ...ready, status: "NoSkill" } }));

    renderOutlook();

    expect(
      await screen.findByText("Freight forecast: Not better than last week yet"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("says drivers cover the freight ahead when a ready forecast finds no window", async () => {
    mocks.outlook.mockResolvedValue(outlook({ windows: [] }));

    renderOutlook();

    expect(await screen.findByText("Drivers cover the freight ahead")).toBeInTheDocument();
    expect(
      screen.getByText("No freight expected beyond your drivers in the next 48 hours."),
    ).toBeInTheDocument();
  });

  it("draws a failed read as a failure, never as an empty outlook, and tries again", async () => {
    const user = userEvent.setup();
    mocks.outlook.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(outlook());

    renderOutlook();

    expect(await screen.findByText("The outlook didn't load")).toBeInTheDocument();
    expect(screen.queryByText("Drivers cover the freight ahead")).toBeNull();

    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByRole("list", { name: "Freight outlook" })).toBeInTheDocument();
    expect(mocks.outlook).toHaveBeenCalledTimes(2);
  });
});
