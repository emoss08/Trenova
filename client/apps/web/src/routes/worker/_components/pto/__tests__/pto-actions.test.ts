import type { PTOStatus } from "@trenova/shared/types/worker";
import { describe, expect, it } from "vitest";
import {
  bulkPayloadToOutcome,
  canApplyPTOAction,
  canTransitionPTO,
  ptoIds,
  splitPTOByAction,
} from "../pto-actions";

describe("PTO transitions", () => {
  it.each<[PTOStatus, PTOStatus, boolean]>([
    ["Requested", "Approved", true],
    ["Requested", "Rejected", true],
    ["Requested", "Cancelled", true],
    ["Approved", "Cancelled", true],
    ["Approved", "Approved", false],
    ["Approved", "Rejected", false],
    ["Rejected", "Approved", false],
    ["Rejected", "Cancelled", false],
    ["Cancelled", "Approved", false],
    ["Cancelled", "Cancelled", false],
  ])("%s -> %s is %s", (from, to, allowed) => {
    expect(canTransitionPTO(from, to)).toBe(allowed);
  });

  it("maps bulk actions onto the same rules the server enforces", () => {
    expect(canApplyPTOAction("Requested", "Approve")).toBe(true);
    expect(canApplyPTOAction("Approved", "Approve")).toBe(false);
    expect(canApplyPTOAction("Approved", "Cancel")).toBe(true);
    expect(canApplyPTOAction("Rejected", "Reject")).toBe(false);
  });

  it("does not treat an unknown status as eligible", () => {
    expect(canTransitionPTO("Pending" as PTOStatus, "Approved")).toBe(false);
  });
});

describe("splitPTOByAction", () => {
  const rows = [
    { id: "a", status: "Requested" as const },
    { id: "b", status: "Approved" as const },
    { id: "c", status: "Rejected" as const },
    { id: "d", status: "Cancelled" as const },
  ];

  it("keeps only requested rows for approval and preserves order", () => {
    const { eligible, ineligible } = splitPTOByAction(rows, "Approve");
    expect(ptoIds(eligible)).toEqual(["a"]);
    expect(ptoIds(ineligible)).toEqual(["b", "c", "d"]);
  });

  it("allows requested and approved rows to be cancelled", () => {
    const { eligible, ineligible } = splitPTOByAction(rows, "Cancel");
    expect(ptoIds(eligible)).toEqual(["a", "b"]);
    expect(ptoIds(ineligible)).toEqual(["c", "d"]);
  });

  it("drops rows without an id when collecting ids", () => {
    expect(ptoIds([{ id: "x" }, { id: undefined }, { id: "" }])).toEqual(["x"]);
  });
});

describe("bulkPayloadToOutcome", () => {
  it("separates successes and failures keeping the server error text", () => {
    expect(
      bulkPayloadToOutcome({
        successCount: 1,
        failureCount: 1,
        results: [
          { ptoId: "a", success: true, error: "" },
          { ptoId: "b", success: false, error: "PTO is rejected and cannot be approved" },
        ],
      }),
    ).toEqual({
      succeeded: ["a"],
      failed: [{ id: "b", error: "PTO is rejected and cannot be approved" }],
    });
  });
});
