import { describe, expect, it } from "vitest";
import {
  URGENCY_ORDER,
  availabilityMeta,
  formatMiles,
  groupMovesByUrgency,
  pickupState,
  severityMeta,
  urgencyMeta,
  verdictMeta,
  workerInitials,
} from "../dispatch-vocabulary";

describe("urgency", () => {
  it("orders buckets from most to least urgent", () => {
    expect(URGENCY_ORDER).toEqual(["Late", "Now", "Today", "Tomorrow", "Planned"]);
  });

  it("marks a closed pickup window as the most severe", () => {
    expect(urgencyMeta("Late").variant).toBe("danger");
  });

  it("names the groups by the server's rolling rule, not by the calendar", () => {
    expect(URGENCY_ORDER.map((bucket) => urgencyMeta(bucket).label)).toEqual([
      "Late",
      "Due now",
      "Within 24 hours",
      "Within 48 hours",
      "Later",
    ]);
  });

  it("falls back to Planned for an unrecognized bucket", () => {
    expect(urgencyMeta("Whatever")).toBe(urgencyMeta("Planned"));
  });
});

describe("availability and verdict", () => {
  it("reads an open driver as active and a blocked one as inactive", () => {
    expect(availabilityMeta("Open").variant).toBe("success");
    expect(availabilityMeta("Blocked").variant).toBe("danger");
  });

  it("keeps the verdict vocabulary the assignment dialog already uses", () => {
    expect(verdictMeta("feasible").label).toBe("Feasible");
    expect(verdictMeta("tight").label).toBe("Tight");
    expect(verdictMeta("infeasible").label).toBe("Infeasible");
    expect(verdictMeta("nonsense").label).toBe("Unknown");
  });

  it("labels a blocking finding distinctly from an advisory one", () => {
    expect(severityMeta("Block").label).toBe("Blocking");
    expect(severityMeta("Warn").label).toBe("Warning");
    expect(severityMeta("Info").label).toBe("Info");
  });
});

describe("pickupState", () => {
  const now = 1_800_000_000;
  const clock = (seconds: number) => `t${seconds - now}`;
  const move = (overrides: Partial<Parameters<typeof pickupState>[0]>) => ({
    urgency: "Now",
    originWindowStart: now - 3600,
    originWindowEnd: now + 6 * 60,
    originActualArrival: 0,
    destinationWindowStart: now + 86_400,
    destinationWindowEnd: null,
    ...overrides,
  });

  it("reads an open window as open with the time left, in warning, not as late", () => {
    expect(pickupState(move({}), now, clock)).toEqual({
      tone: "warning",
      headline: "Open · closes in 6m",
      window: "window t-3600–t360",
    });
  });

  it("reads a closed window as how long ago it closed, in danger", () => {
    expect(
      pickupState(move({ urgency: "Late", originWindowEnd: now - 25 * 60 }), now, clock),
    ).toMatchObject({ tone: "danger", headline: "Closed 25m ago" });
  });

  it("counts down to a window opening within four hours", () => {
    expect(
      pickupState(
        move({ originWindowStart: now + 40 * 60, originWindowEnd: now + 3 * 3600 }),
        now,
        clock,
      ),
    ).toMatchObject({ tone: "neutral", headline: "Opens in 40m" });
  });

  it("names the opening time for a window further out", () => {
    expect(
      pickupState(
        move({ urgency: "Today", originWindowStart: now + 6 * 3600, originWindowEnd: null }),
        now,
        clock,
      ),
    ).toEqual({ tone: "neutral", headline: "Opens t21600", window: "window t21600" });
  });

  it("reads the delivery window once the truck has arrived at pickup", () => {
    expect(
      pickupState(
        move({
          urgency: "Late",
          originActualArrival: now - 7200,
          destinationWindowStart: now - 3 * 3600,
          destinationWindowEnd: now - 2 * 3600,
        }),
        now,
        clock,
      ),
    ).toMatchObject({ tone: "danger", headline: "Delivery closed 2h 0m ago" });
  });

  it("says when there is no appointment rather than inventing a time", () => {
    expect(pickupState(move({ originWindowStart: 0, originWindowEnd: null }), now, clock)).toEqual({
      tone: "neutral",
      headline: "No appointment",
      window: "no appointment",
    });
  });
});

describe("value formatting", () => {
  // Absent data has to be visibly absent. Rendering it as 0 would read as a real
  // measurement — a truck with no GPS fix is not a truck with zero empty miles.
  it("renders missing miles as a dash", () => {
    expect(formatMiles(null)).toBe("—");
    expect(formatMiles(undefined)).toBe("—");
  });

  it("rounds miles for scanning", () => {
    expect(formatMiles(123.7)).toBe("124 mi");
  });
});

describe("workerInitials", () => {
  it("takes the first letter of each name", () => {
    expect(workerInitials("Jane", "Doe")).toBe("JD");
  });

  it("falls back to a placeholder when both names are empty", () => {
    expect(workerInitials("", "")).toBe("?");
  });
});

describe("groupMovesByUrgency", () => {
  it("buckets moves in urgency order and defaults unknown urgencies to Planned", () => {
    const grouped = groupMovesByUrgency([
      { urgency: "Now", id: "a" },
      { urgency: "Late", id: "b" },
      { urgency: "Whatever", id: "c" },
    ]);
    expect([...grouped.keys()]).toEqual([...URGENCY_ORDER]);
    expect(grouped.get("Late")?.map((move) => move.id)).toEqual(["b"]);
    expect(grouped.get("Now")?.map((move) => move.id)).toEqual(["a"]);
    expect(grouped.get("Planned")?.map((move) => move.id)).toEqual(["c"]);
  });
});
