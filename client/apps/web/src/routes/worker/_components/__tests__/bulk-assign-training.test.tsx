import { describe, expect, it } from "vitest";
import { describeBulkAssign } from "../bulk-assign-training-dialog";

type Result = Parameters<typeof describeBulkAssign>[0];

function result(assigned: number, skipped: number, failed: number): Result {
  return {
    assignedCount: assigned,
    skippedCount: skipped,
    failedCount: failed,
    outcomes: [],
  } as unknown as Result;
}

describe("describeBulkAssign", () => {
  it("reports a clean rollout as a count", () => {
    expect(describeBulkAssign(result(24, 0, 0))).toBe("24 assigned");
  });

  // Somebody who already has the course open is not a failure. Rolling it into
  // a failure count would make a second click look like something broke.
  it("counts already-enrolled workers apart from failures", () => {
    expect(describeBulkAssign(result(21, 3, 0))).toBe("21 assigned, 3 already open");
  });

  it("names real failures when there are any", () => {
    expect(describeBulkAssign(result(18, 3, 3))).toBe("18 assigned, 3 already open, 3 failed");
  });

  // A run where every worker already had the course is the second click. It
  // must still read as a result rather than an empty string.
  it("still says something when nothing was newly assigned", () => {
    expect(describeBulkAssign(result(0, 5, 0))).toBe("0 assigned, 5 already open");
  });
});
