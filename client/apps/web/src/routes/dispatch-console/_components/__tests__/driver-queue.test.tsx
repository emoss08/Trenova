import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { stubLayout } from "@/test/layout";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DriverQueue, visibleDrivers, whenFree } from "../driver-queue";
import type { QueueRead } from "../move-queue";

const NOW = 1_800_000_000;
const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d)\}/g, (_, i) => String(args[Number(i)]))) as never;

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
    ...overrides,
  } as DispatchBoardDriver;
}

const ok: QueueRead = { isPending: false, isError: false, error: null, refetch: vi.fn() };

describe("visibleDrivers", () => {
  const open = driver({ workerId: "wrk_open", firstName: "Ana", availability: "Open" });
  const soon = driver({
    workerId: "wrk_soon",
    firstName: "Ben",
    availability: "Working",
    projectedTimeAvailable: NOW + 600,
  });
  const later = driver({
    workerId: "wrk_later",
    firstName: "Cal",
    availability: "Working",
    projectedTimeAvailable: NOW + 7200,
    tractorCode: "T-207",
    fleetCodeName: "Gary",
  });

  it("puts open capacity first and the soonest free first within a group", () => {
    expect(visibleDrivers([later, soon, open], "all", "").map((d) => d.workerId)).toEqual([
      "wrk_open",
      "wrk_soon",
      "wrk_later",
    ]);
  });

  it("finds a driver by name, tractor or fleet", () => {
    expect(visibleDrivers([open, later], "all", "t-207")).toEqual([later]);
    expect(visibleDrivers([open, later], "all", "gary")).toEqual([later]);
    expect(visibleDrivers([open, later], "all", "ana")).toEqual([open]);
  });

  it("narrows to the availability a figure asked for", () => {
    expect(visibleDrivers([open, soon], "open", "")).toEqual([open]);
  });
});

describe("whenFree", () => {
  it("says when a working driver frees up, rather than only that they are working", () => {
    expect(whenFree(driver({ availability: "Open" }), t)).toBe("Free now");
    expect(whenFree(driver({ availability: "Blocked" }), t)).toBe("Blocked");
    expect(whenFree(driver({ availability: "Working", projectedTimeAvailable: 0 }), t)).toBe(
      "On a load",
    );
  });
});

describe("DriverQueue", () => {
  let restoreLayout = () => {};
  beforeEach(() => {
    restoreLayout = stubLayout(800);
  });
  afterEach(() => {
    restoreLayout();
    cleanup();
  });

  function renderQueue(
    drivers: DispatchBoardDriver[],
    read: QueueRead = ok,
    search = "",
    truncated = false,
  ) {
    render(
      <DriverQueue
        drivers={drivers}
        truncated={truncated}
        read={read}
        filter="all"
        search={search}
        selectedWorkerId={null}
        onSelectDriver={vi.fn()}
        onClearFilters={vi.fn()}
      />,
    );
  }

  it("groups drivers under their availability, open first", () => {
    renderQueue([
      driver({ workerId: "wrk_w", firstName: "Will", availability: "Working" }),
      driver({ workerId: "wrk_o", firstName: "Olga", availability: "Open" }),
    ]);

    const text = screen.getByRole("listbox").textContent ?? "";
    expect(text.indexOf("Open")).toBeLessThan(text.indexOf("Olga"));
    expect(text.indexOf("Olga")).toBeLessThan(text.indexOf("Working"));
    expect(text.indexOf("Working")).toBeLessThan(text.indexOf("Will"));
  });

  it("says why a blocked driver can't take work", () => {
    renderQueue([
      driver({
        availability: "Blocked",
        findings: [
          {
            code: "MED_CARD",
            severity: "Block",
            field: "medical",
            message: "Medical card expired",
            regulation: null,
          },
        ] as DispatchBoardDriver["findings"],
      }),
    ]);

    const row = screen.getByRole("option");
    expect(row).toHaveTextContent("Can't be dispatched");
    expect(row).toHaveTextContent("Medical card expired");
  });

  it("draws a failed read as a failure, never as an empty roster", () => {
    renderQueue([], { ...ok, isError: true, error: new Error("timeout") });

    expect(screen.getByText("The drivers didn't load")).toBeInTheDocument();
    expect(screen.queryByText("No drivers on the board")).toBeNull();
  });

  it("offers to clear a search that matched nobody", () => {
    renderQueue([driver()], ok, "nobody");

    expect(screen.getByText("No drivers match")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Clear filters" })).toBeInTheDocument();
  });

  it("says when the board lists only the first drivers", () => {
    renderQueue([driver({ workerId: "wrk_a" }), driver({ workerId: "wrk_b" })], ok, "", true);

    expect(
      screen.getByText(
        "The board lists only the first 2 drivers, available drivers first. Filter by fleet to see the rest.",
      ),
    ).toBeInTheDocument();
  });

  it("says nothing about a limit when every driver is listed", () => {
    renderQueue([driver()]);

    expect(screen.queryByText(/The board lists only the first/)).toBeNull();
  });
});
