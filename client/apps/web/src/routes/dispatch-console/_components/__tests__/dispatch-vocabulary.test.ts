import { describe, expect, it } from "vitest";
import {
  URGENCY_ORDER,
  availabilityMeta,
  formatMiles,
  formatMinutesToPickup,
  formatMoney,
  scoreTone,
  severityMeta,
  urgencyMeta,
  verdictMeta,
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
  it("renders missing miles and money as a dash", () => {
    expect(formatMiles(null)).toBe("—");
    expect(formatMiles(undefined)).toBe("—");
    expect(formatMoney(null)).toBe("—");
  });

  it("rounds miles and money for scanning", () => {
    expect(formatMiles(123.7)).toBe("124 mi");
    expect(formatMoney(1234.56)).toBe("$1,235");
  });
});

describe("scoreTone", () => {
  it("separates strong, middling, and weak scores", () => {
    expect(scoreTone(90)).not.toBe(scoreTone(60));
    expect(scoreTone(60)).not.toBe(scoreTone(20));
  });
});
