import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
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
  ratings: [{ rating: "Excellent", workers: 15 }],
  basics: [
    basic({ basic: "UnsafeDriving", violations: 4, weightedScore: 60 }),
    basic({ basic: "HOSCompliance", violations: 1, weightedScore: 10 }),
    basic({ basic: "DriverFitness" }),
    basic({ basic: "ControlledSubstances" }),
    basic({ basic: "VehicleMaintenance" }),
    basic({ basic: "HazmatCompliance" }),
    basic({ basic: "CrashIndicator" }),
  ],
  kinds: [
    {
      kind: "Accident",
      events: 3,
      points: 18,
      preventable: 2,
      outOfService: 0,
      open: 1,
    },
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
  it("shows the fleet's standing across the window", async () => {
    renderConsole();

    expect(await screen.findByText("86")).toBeInTheDocument();
    expect(screen.getByText("Drivers")).toBeInTheDocument();
    expect(screen.getByText("20")).toBeInTheDocument();
    expect(screen.getByText("4 open")).toBeInTheDocument();
  });

  // A scorecard that hides its zeroes reads as a shorter list every month, and
  // a BASIC missing from the page is not the same as a BASIC at zero.
  it("lists all seven BASICs in the agency's order", async () => {
    renderConsole();

    expect(await screen.findByText("Unsafe Driving")).toBeInTheDocument();
    for (const label of [
      "Hours of Service",
      "Driver Fitness",
      "Controlled Substances",
      "Vehicle Maintenance",
      "Hazmat Compliance",
      "Crash Indicator",
    ]) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
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

    const attention = (await screen.findByText("Needs attention")).closest(
      "section",
    ) as HTMLElement;
    expect(within(attention).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(attention).getByText("At risk")).toBeInTheDocument();

    const best = screen.getByText("Best records").closest("section") as HTMLElement;
    expect(within(best).getByText("Cal Diaz")).toBeInTheDocument();
  });

  it("breaks the roster down by terminal", async () => {
    renderConsole();

    const section = (await screen.findByText("By terminal")).closest("section") as HTMLElement;
    expect(within(section).getByText("SOUTH")).toBeInTheDocument();
    expect(within(section).getByText("Southern lanes")).toBeInTheDocument();
    expect(within(section).getByText("2 at risk")).toBeInTheDocument();
    expect(within(section).getByText("12 · avg 81")).toBeInTheDocument();
  });

  it("says nothing happened rather than drawing an empty chart", async () => {
    renderConsole({ ...baseSummary, trend: [] });

    expect(await screen.findByText("No safety events in this window.")).toBeInTheDocument();
  });
});
