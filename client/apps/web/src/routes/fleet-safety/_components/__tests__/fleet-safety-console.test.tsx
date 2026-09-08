import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import FleetSafetyConsole from "../fleet-safety-console";

const { fetchFleetSafety } = vi.hoisted(() => ({ fetchFleetSafety: vi.fn() }));

vi.mock("@/lib/graphql/fleet-safety", () => ({
  fetchFleetSafety,
  FLEET_SAFETY_KEY: "fleet-safety",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

function basic(over: Record<string, unknown> = {}) {
  return {
    basic: "UnsafeDriving",
    violations: 0,
    events: 0,
    weightedScore: 0,
    outOfService: 0,
    inferred: false,
    ...over,
  };
}

const baseSummary = {
  asOf: 1_800_000_000,
  windowMonths: 12,
  workers: 20,
  averageScore: 86,
  atRisk: 2,
  watch: 3,
  totalEvents: 14,
  totalPoints: 31,
  openEvents: 4,
  outOfServiceOrders: 1,
  basicsInferred: false,
  ratings: [
    { rating: "Excellent", workers: 15 },
    { rating: "AtRisk", workers: 2 },
  ],
  basics: [
    basic({ basic: "UnsafeDriving", violations: 4, weightedScore: 60 }),
    basic({ basic: "HOSCompliance", violations: 1, weightedScore: 30 }),
    basic({ basic: "DriverFitness" }),
    basic({ basic: "ControlledSubstances" }),
    basic({ basic: "VehicleMaintenance" }),
    basic({ basic: "HazmatCompliance" }),
    basic({ basic: "CrashIndicator" }),
  ],
  kinds: [
    { kind: "Accident", events: 3, points: 18, preventable: 2, outOfService: 0, open: 1 },
    { kind: "Inspection", events: 11, points: 13, preventable: 0, outOfService: 1, open: 3 },
  ],
  terminals: [
    {
      fleetCodeId: "fc_1",
      code: "SOUTH",
      description: "Southern lanes",
      color: "#2563eb",
      workers: 12,
      atRisk: 2,
      watch: 1,
      averageScore: 81,
    },
    {
      fleetCodeId: "fc_2",
      code: "NORTH",
      description: "",
      color: "",
      workers: 8,
      atRisk: 0,
      watch: 2,
      averageScore: 93,
    },
  ],
  trend: [
    {
      periodStart: 1_790_000_000,
      events: 2,
      accidents: 1,
      preventable: 0,
      citations: 0,
      inspections: 1,
      outOfService: 0,
      points: 4,
    },
    {
      periodStart: 1_792_000_000,
      events: 5,
      accidents: 2,
      preventable: 1,
      citations: 1,
      inspections: 2,
      outOfService: 1,
      points: 12,
    },
  ],
  worst: [
    {
      workerId: "wrk_1",
      name: "Ada Byrne",
      fleetCodeId: "fc_1",
      fleetCode: "SOUTH",
      fleetColor: "#2563eb",
      rating: "AtRisk",
      score: 42,
      activePoints: 11,
      events: 3,
      lastEventAt: 1_799_000_000,
    },
  ],
  best: [
    {
      workerId: "wrk_2",
      name: "Cal Diaz",
      fleetCodeId: "fc_1",
      fleetCode: "SOUTH",
      fleetColor: "#2563eb",
      rating: "Excellent",
      score: 100,
      activePoints: 0,
      events: 0,
      lastEventAt: null,
    },
  ],
};

function renderConsole(summary: unknown = baseSummary) {
  fetchFleetSafety.mockResolvedValue(summary);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <FleetSafetyConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("FleetSafetyConsole", () => {
  // The loading state is the loaded page drawn in grey: the four KPI cards,
  // all seven BASIC rows and the three panels, at the sizes they take once the
  // roll-up lands, so the page does not reflow when the numbers arrive.
  it("draws the page's own shape while the roll-up is still being read", () => {
    fetchFleetSafety.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <FleetSafetyConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading fleet safety");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const basics = within(loading).getByRole("region", { name: "CSA BASICs", ...hidden });
    expect(within(basics).getAllByRole("listitem", hidden)).toHaveLength(7);
    for (const name of ["Events by month", "By terminal", "Needs attention", "Best records"]) {
      expect(within(loading).getByRole("region", { name, ...hidden })).toBeInTheDocument();
    }
  });

  it("heads the page with drivers by rating, the average, the window and out-of-service orders", async () => {
    renderConsole();

    expect(await screen.findByLabelText("Drivers", { selector: "span" })).toHaveTextContent("20");
    expect(screen.getByRole("img", { name: /Drivers by rating/ })).toHaveAccessibleName(
      "Drivers by rating: Excellent 15, Good 0, Watch 0, At risk 2",
    );
    expect(screen.getByLabelText("Average score", { selector: "span" })).toHaveTextContent("86");
    expect(screen.getByText("2 at risk · 3 on watch")).toBeInTheDocument();
    expect(screen.getByLabelText("Events in window", { selector: "span" })).toHaveTextContent("14");
    expect(screen.getByText("4 open · 2 preventable")).toBeInTheDocument();
    expect(screen.getByLabelText("Out of service", { selector: "span" })).toHaveTextContent("1");
  });

  // A scorecard that hides its zeroes reads as a shorter list every month, and
  // a BASIC missing from the page is not the same as a BASIC at zero.
  it("lists all seven BASICs in the agency's order, read against the fleet's own worst", async () => {
    renderConsole();

    const card = await screen.findByRole("region", { name: "CSA BASICs" });
    const bars = within(card).getAllByRole("img");
    expect(bars.map((bar) => bar.getAttribute("aria-label"))).toEqual([
      "Unsafe Driving: 60 weighted",
      "Hours of Service: 30 weighted",
      "Driver Fitness: 0 weighted",
      "Controlled Substances: 0 weighted",
      "Vehicle Maintenance: 0 weighted",
      "Hazmat Compliance: 0 weighted",
      "Crash Indicator: 0 weighted",
    ]);
    expect(within(card).getByText("Highest")).toBeInTheDocument();
    expect(within(card).getByText("Elevated")).toBeInTheDocument();
    expect(within(card).getByText("· 4 violations")).toBeInTheDocument();
  });

  // A number reached from the kind of event is an estimate. Presenting it as
  // an inspection result would be worse than showing nothing.
  it("says so when the BASICs were inferred rather than cited", async () => {
    renderConsole({
      ...baseSummary,
      basicsInferred: true,
      basics: [basic({ basic: "CrashIndicator", events: 3, weightedScore: 18, inferred: true })],
    });

    expect(
      await screen.findByText(/reached from the kind of event rather than from violations/),
    ).toBeInTheDocument();
    expect(screen.getByText("Inferred")).toBeInTheDocument();
  });

  it("ranks the drivers who need attention and the ones who do not", async () => {
    renderConsole();

    const attention = await screen.findByRole("region", { name: "Needs attention" });
    expect(within(attention).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(attention).getByText("At risk")).toBeInTheDocument();
    expect(within(attention).getByText("· 11 pts")).toBeInTheDocument();

    const best = screen.getByRole("region", { name: "Best records" });
    expect(within(best).getByText("Cal Diaz")).toBeInTheDocument();
    expect(within(best).queryByText("Ada Byrne")).not.toBeInTheDocument();
  });

  // The question after "how is the fleet" is "is it one yard": the terminal
  // list narrows the whole page, and the worst yard comes first.
  it("breaks the fleet down by terminal, worst first, and narrows the page to one", async () => {
    const user = userEvent.setup();
    renderConsole();

    const section = await screen.findByRole("region", { name: "By terminal" });
    const rows = within(section).getAllByRole("button", { pressed: false });
    expect(rows.map((row) => row.getAttribute("aria-pressed"))).toEqual(["false", "false"]);
    expect(within(rows[0]).getByText("SOUTH")).toBeInTheDocument();
    expect(within(rows[0]).getByText("Southern lanes")).toBeInTheDocument();
    expect(within(rows[0]).getByText("2 at risk")).toBeInTheDocument();
    expect(within(rows[0]).getByText("12 · avg 81")).toBeInTheDocument();
    expect(within(rows[1]).getByText("NORTH")).toBeInTheDocument();

    await user.click(rows[0]);
    await waitFor(() =>
      expect(fetchFleetSafety).toHaveBeenLastCalledWith(
        { windowMonths: 12, fleetCodeId: "fc_1", rankLimit: 10 },
        expect.anything(),
      ),
    );
    expect(await screen.findByRole("button", { name: "Clear terminal SOUTH" })).toBeInTheDocument();
  });

  it("re-reads the fleet for a different window", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("region", { name: "CSA BASICs" });
    await user.click(
      within(screen.getByRole("radiogroup", { name: "Counting window" })).getByRole("radio", {
        name: "6 months",
      }),
    );

    await waitFor(() =>
      expect(fetchFleetSafety).toHaveBeenLastCalledWith(
        { windowMonths: 6, fleetCodeId: null, rankLimit: 10 },
        expect.anything(),
      ),
    );
  });

  it("lists what the events were made of", async () => {
    renderConsole();

    const kinds = await screen.findByRole("list", { name: "Events by kind" });
    expect(within(kinds).getByText("Accidents")).toBeInTheDocument();
    expect(within(kinds).getByText("3 · 2 preventable · 1 open")).toBeInTheDocument();
    expect(within(kinds).getByText("11 · 1 out of service · 3 open")).toBeInTheDocument();
    expect(within(kinds).getByText("14 · 31 points")).toBeInTheDocument();
  });

  it("says nothing happened rather than drawing an empty chart", async () => {
    renderConsole({ ...baseSummary, trend: [] });

    expect(await screen.findByText("No safety events in this window.")).toBeInTheDocument();
  });
});
