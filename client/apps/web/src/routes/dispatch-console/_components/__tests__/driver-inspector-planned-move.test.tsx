import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type {
  DispatchBoardDriver,
  DispatchDriverPlannedMove,
} from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DriverInspector } from "../driver-inspector";

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    driverMoves: (input: unknown) => ({
      queryKey: ["test-driver-moves", input] as const,
      queryFn: async () => [],
    }),
  },
}));

afterEach(cleanup);

// Written from DispatchDriverPlannedMove in dispatch_console.graphqls.
const plannedMove: DispatchDriverPlannedMove = {
  moveId: "smv_7",
  proNumber: "PRO-7",
  costCents: 14_200,
  valueCents: 3_100,
  generatedAt: 1_800_000_000,
};

// Written from DispatchBoardDriver in dispatch_console.graphqls.
function driver(overrides: Partial<DispatchBoardDriver> = {}): DispatchBoardDriver {
  return {
    workerId: "wrk_1",
    firstName: "Dana",
    lastName: "Ortiz",
    workerType: "Employee",
    driverType: "OTR",
    fleetCodeId: "fc_1",
    fleetCodeName: "Chicago",
    city: "Joliet",
    stateAbbreviation: "IL",
    postalCode: "60431",
    profilePicUrl: "",
    assignmentBlocked: "",
    availableForDispatch: true,
    tractorId: "trc_1",
    tractorCode: "T-114",
    tractorTypeId: null,
    tractorAvailable: true,
    openAssignments: 0,
    availability: "Open",
    dutyStatus: "",
    driveRemainingMs: 0,
    shiftRemainingMs: 0,
    cycleRemainingMs: 0,
    breakRemainingMs: 0,
    hosRecordedAt: 0,
    hosIsStale: false,
    latitude: null,
    longitude: null,
    formattedLocation: "",
    positionRecordedAt: 0,
    projectedTimeAvailable: 0,
    committedMiles: 0,
    committedRevenue: 0,
    commitments: [],
    timeOff: [],
    findings: [],
    plannedMove: null,
    lastHomeAt: null,
    ...overrides,
  } as DispatchBoardDriver;
}

function renderInspector(value: DispatchBoardDriver, onSelectMove = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <DriverInspector
        driver={value}
        range={{ start: 0, end: 86_400 }}
        onSelectMove={onSelectMove}
        onClose={vi.fn()}
      />
    </QueryClientProvider>,
  );
  return onSelectMove;
}

describe("DriverInspector planned move", () => {
  it("says which move the last plan put the driver on and what it costs", () => {
    renderInspector(driver({ plannedMove }));

    expect(screen.getByText("Plan").nextElementSibling).toHaveTextContent(
      "Planned on PRO-7 for $142",
    );
  });

  it("says what the driver is worth to the plan", () => {
    renderInspector(driver({ plannedMove }));

    expect(screen.getByText("Worth $31 to the plan")).toBeInTheDocument();
  });

  it("leaves out the driver's worth when losing them would cost the plan nothing", () => {
    renderInspector(driver({ plannedMove: { ...plannedMove, valueCents: 0 } }));

    expect(screen.getByText("Planned on PRO-7 for $142")).toBeInTheDocument();
    expect(screen.queryByText(/to the plan/)).toBeNull();
  });

  it("opens the planned move when its line is clicked", async () => {
    const onSelectMove = renderInspector(driver({ plannedMove }));

    await userEvent.click(screen.getByRole("button", { name: "Planned on PRO-7 for $142" }));

    expect(onSelectMove).toHaveBeenCalledWith("smv_7");
  });

  it("draws no plan line for a driver the last plan did not use", () => {
    renderInspector(driver({ plannedMove: null }));

    expect(screen.queryByText("Plan")).toBeNull();
    expect(screen.queryByText(/Planned on/)).toBeNull();
  });
});

describe("DriverInspector last home", () => {
  it("says when the driver was last home", () => {
    const lastHomeAt = 1_762_300_800;
    renderInspector(driver({ lastHomeAt }));

    expect(screen.getByText("Last home").nextElementSibling).toHaveTextContent(
      formatUnixDateMedium(lastHomeAt),
    );
  });

  it("says nothing about home when no record says when the driver was there", () => {
    renderInspector(driver({ lastHomeAt: null }));

    expect(screen.queryByText("Last home")).toBeNull();
  });
});
