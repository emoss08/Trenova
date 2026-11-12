import type { PreviewFieldChange } from "@/lib/graphql/agent-preview";
import { describe, expect, it } from "vitest";
import { isBareId, readableField, readableWallClock } from "../plan-values";

function field(overrides: Partial<PreviewFieldChange> = {}): PreviewFieldChange {
  return {
    path: "windowStart",
    label: "Window opens",
    valueType: "text",
    before: null,
    after: null,
    beforeRef: null,
    afterRef: null,
    withheld: false,
    volatile: false,
    truncated: false,
    changedSinceProposed: false,
    proposedBefore: null,
    projectedFromStep: 0,
    ...overrides,
  };
}

describe("a plan step's values", () => {
  it("reads a zoneless wall-clock time at the hour it names, not the reader's", () => {
    expect(readableWallClock("2026-11-10T13:00")).toBe("Nov 10, 1:00 PM");
    expect(readableWallClock("2026-11-10T02:45:25")).toBe("Nov 10, 2:45 AM");
  });

  it("leaves anything that is not a wall-clock time as it came", () => {
    expect(readableWallClock("Rate discrepancy")).toBe("Rate discrepancy");
    expect(readableWallClock("2026-11-10")).toBe("2026-11-10");
    expect(readableWallClock("2026-11-10T13:00:00Z")).toBe("2026-11-10T13:00:00Z");
    expect(readableWallClock(42)).toBe(42);
    expect(readableWallClock(null)).toBeNull();
  });

  it("makes the before, the after and the value at proposal time readable together", () => {
    const shown = readableField(
      field({
        before: "2026-11-10T02:45:25",
        after: "2026-11-10T13:00",
        proposedBefore: "2026-11-10T02:30",
      }),
    );

    expect(shown.before).toBe("Nov 10, 2:45 AM");
    expect(shown.after).toBe("Nov 10, 1:00 PM");
    expect(shown.proposedBefore).toBe("Nov 10, 2:30 AM");
  });

  it("calls a value a bare id only when every side is a key with no name beside it", () => {
    const id = "cus_01M717A1YJ0N96WTX0S22JS511";
    expect(isBareId(field({ after: id }))).toBe(true);
    expect(isBareId(field({ before: id, after: "shp_01M71R3J9VPPG7YMXBHHNHM119" }))).toBe(true);
    expect(isBareId(field({ before: id, after: "Acme Foods" }))).toBe(false);
    expect(isBareId(field({ after: "Acme Foods" }))).toBe(false);
    expect(isBareId(field())).toBe(false);
  });

  it("keeps an id the server named, and one the reader may not see", () => {
    const id = "cus_01M717A1YJ0N96WTX0S22JS511";
    const ref = { resource: "customer", id, label: "Acme Foods", record: null, withheld: false };

    expect(isBareId(field({ after: id, afterRef: ref }))).toBe(false);
    expect(isBareId(field({ after: id, withheld: true }))).toBe(false);
  });
});
