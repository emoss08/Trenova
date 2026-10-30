import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { stubLayout, stubScrollTo } from "@/test/layout";
import { DndContext } from "@dnd-kit/core";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { CapacityRail } from "../capacity-rail";

// Availability values are dispatchconsoleservice.DriverAvailability; the board sends them
// as plain strings (`availability: String!`).
function driver(
  workerId: string,
  firstName: string,
  lastName: string,
  availability: string,
  overrides: Partial<DispatchBoardDriver> = {},
): DispatchBoardDriver {
  return {
    workerId,
    firstName,
    lastName,
    workerType: "Employee",
    driverType: "OTR",
    fleetCodeId: null,
    fleetCodeName: "",
    city: "Dallas",
    stateAbbreviation: "TX",
    postalCode: "75201",
    profilePicUrl: "",
    assignmentBlocked: "",
    availableForDispatch: availability !== "Blocked",
    tractorId: null,
    tractorCode: "",
    tractorTypeId: null,
    tractorAvailable: true,
    openAssignments: 0,
    availability,
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
    ...overrides,
  } as DispatchBoardDriver;
}

// Two open drivers so a count is never confused with presence, and no one finishing so
// one lens is empty.
const drivers = [
  driver("wrk_01", "Ana", "Ruiz", "Open"),
  driver("wrk_02", "Ben", "Cole", "Open"),
  driver("wrk_03", "Cara", "Diaz", "Working"),
  driver("wrk_04", "Dev", "Shah", "Blocked", {
    findings: [
      {
        code: "MedicalExpired",
        severity: "Block",
        field: "",
        message: "Medical card expired",
        regulation: "",
      },
    ],
  }),
  driver("wrk_05", "Eli", "Park", "TimeOff"),
];

const allNames = ["Ana Ruiz", "Ben Cole", "Cara Diaz", "Dev Shah", "Eli Park"];

function renderRail(searchParams = "") {
  return render(
    <NuqsTestingAdapter hasMemory searchParams={searchParams}>
      <DndContext>
        <CapacityRail
          drivers={drivers}
          isLoading={false}
          selectedWorkerId={null}
          onSelectDriver={() => undefined}
        />
      </DndContext>
    </NuqsTestingAdapter>,
  );
}

function shownNames() {
  return allNames.filter((name) => screen.queryByText(name) !== null);
}

// The rail is virtualized, and jsdom lays nothing out: without a size no row would mount.
let restoreLayout = () => {};

describe("capacity rail filter", () => {
  beforeEach(() => {
    restoreLayout = stubLayout(320);
  });

  afterEach(() => {
    restoreLayout();
    cleanup();
  });

  it("lists every availability lens with its driver count", async () => {
    const user = userEvent.setup();
    renderRail();

    await user.click(screen.getByRole("combobox", { name: "Filter drivers by availability" }));

    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
      "All5",
      "Open2",
      "Finishing0",
      "Working1",
      "Blocked1",
      "Time off1",
    ]);
  });

  it("shows the active lens on the closed filter without a count", () => {
    renderRail("?capacity=open");

    expect(
      screen.getByRole("combobox", { name: "Filter drivers by availability" }),
    ).toHaveTextContent(/^Open$/);
  });

  it("narrows the rail to the availability picked from the filter", async () => {
    const user = userEvent.setup();
    renderRail();

    await user.click(screen.getByRole("combobox", { name: "Filter drivers by availability" }));
    await user.click(screen.getByRole("option", { name: /^Working/ }));

    expect(shownNames()).toEqual(["Cara Diaz"]);
    expect(screen.getByText("1 of 5")).toBeInTheDocument();
  });

  it("applies the search and the availability lens together", async () => {
    const user = userEvent.setup();
    renderRail("?capacity=open");

    await user.type(
      screen.getByRole("textbox", { name: "Search drivers by name, tractor code, or fleet" }),
      "ben",
    );

    expect(shownNames()).toEqual(["Ben Cole"]);
    expect(screen.getByText("1 of 5")).toBeInTheDocument();
  });

  it("clears the lens and the search from a view that matches no one", async () => {
    const user = userEvent.setup();
    renderRail("?capacity=blocked&q=nobody");

    expect(screen.getByText("No drivers match this view.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));

    expect(shownNames()).toEqual(allNames);
    expect(screen.getByText("5 of 5")).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Search drivers by name, tractor code, or fleet" }),
    ).toHaveValue("");
  });

  it("offers no reset when the rail is unfiltered and simply empty", () => {
    render(
      <NuqsTestingAdapter hasMemory>
        <DndContext>
          <CapacityRail
            drivers={[]}
            isLoading={false}
            selectedWorkerId={null}
            onSelectDriver={() => undefined}
          />
        </DndContext>
      </NuqsTestingAdapter>,
    );

    expect(screen.getByText("No drivers match this view.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).toBeNull();
  });
});

// Every driver is open, so the rail orders them by when they free up: the index.
const roster = Array.from({ length: 200 }, (_, index) =>
  driver(`wrk_${index}`, "Driver", String(index).padStart(3, "0"), "Open", {
    projectedTimeAvailable: 1_720_000_000 + index,
  }),
);

function renderRoster(selectedWorkerId: string | null) {
  const ui = (selected: string | null) => (
    <NuqsTestingAdapter hasMemory>
      <DndContext>
        <CapacityRail
          drivers={roster}
          isLoading={false}
          selectedWorkerId={selected}
          onSelectDriver={() => undefined}
        />
      </DndContext>
    </NuqsTestingAdapter>
  );
  const view = render(ui(selectedWorkerId));
  return { select: (next: string | null) => view.rerender(ui(next)) };
}

describe("capacity rail on a large roster", () => {
  let restoreScroll = () => {};

  beforeEach(() => {
    restoreLayout = stubLayout(320);
    restoreScroll = stubScrollTo();
  });

  afterEach(() => {
    restoreScroll();
    restoreLayout();
    cleanup();
  });

  it("mounts only the drivers near the viewport", () => {
    renderRoster(null);

    expect(screen.getByText("Driver 000")).toBeInTheDocument();
    expect(screen.queryByText("Driver 199")).toBeNull();
    expect(screen.getByText("200 of 200")).toBeInTheDocument();
  });

  it("scrolls a driver selected elsewhere into the rail", async () => {
    const { select } = renderRoster(null);
    expect(screen.queryByText("Driver 150")).toBeNull();

    select("wrk_150");

    await waitFor(() => expect(screen.getByText("Driver 150")).toBeInTheDocument());
    expect(screen.queryByText("Driver 000")).toBeNull();
  });
});
