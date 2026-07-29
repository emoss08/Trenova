import { describe, expect, it } from "vitest";
import {
  URGENCY_ORDER,
  availabilityMeta,
  formatMiles,
  formatMinutesToPickup,
  groupMovesByUrgency,
  scoreTone,
  severityMeta,
  urgencyMeta,
  verdictMeta,
  workerInitials,
} from "../dispatch-vocabulary";

describe("urgency", () => {
  it("orders buckets from most to least urgent", () => {
    expect(URGENCY_ORDER).toEqual(["Late", "Now", "Today", "Tomorrow", "Planned"]);
  });

  it("marks an already-open pickup window as the most severe", () => {
    expect(urgencyMeta("Late").variant).toBe("inactive");
  });

  it("falls back to Planned for an unrecognized bucket", () => {
    expect(urgencyMeta("Whatever")).toBe(urgencyMeta("Planned"));
  });
});

describe("availability and verdict", () => {
  it("reads an open driver as active and a blocked one as inactive", () => {
    expect(availabilityMeta("Open").variant).toBe("active");
    expect(availabilityMeta("Blocked").variant).toBe("inactive");
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

describe("formatMinutesToPickup", () => {
  it("says how late a move already is", () => {
    expect(formatMinutesToPickup(-40)).toBe("40m late");
    expect(formatMinutesToPickup(-190)).toBe("3h 10m late");
  });

  it("counts down to an upcoming pickup", () => {
    expect(formatMinutesToPickup(45)).toBe("in 45m");
    expect(formatMinutesToPickup(190)).toBe("in 3h 10m");
  });

  it("reads zero as due now rather than as in 0m", () => {
    expect(formatMinutesToPickup(0)).toBe("due now");
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

describe("scoreTone", () => {
  it("separates strong, middling, and weak scores", () => {
    expect(scoreTone(90)).not.toBe(scoreTone(60));
    expect(scoreTone(60)).not.toBe(scoreTone(20));
  });
});
