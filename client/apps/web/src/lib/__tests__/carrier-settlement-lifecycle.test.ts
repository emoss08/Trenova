import { describe, expect, it } from "vitest";
import {
  carrierLifecycleEligibility,
  carrierLifecycleVerbs,
  eligibleCarrierSettlements,
} from "../carrier-settlement-lifecycle";

const rows = [
  { id: "1", status: "Draft" },
  { id: "2", status: "PendingApproval" },
  { id: "3", status: "Approved" },
  { id: "4", status: "Posted" },
  { id: "5", status: "Paid" },
  { id: "6", status: "Voided" },
];

describe("eligibleCarrierSettlements", () => {
  it("submits only drafts", () => {
    expect(eligibleCarrierSettlements(rows, "Submit").map((row) => row.id)).toEqual(["1"]);
  });

  it("approves only settlements pending approval, matching the server transition matrix", () => {
    expect(eligibleCarrierSettlements(rows, "Approve").map((row) => row.id)).toEqual(["2"]);
  });

  it("never lets a draft skip the approval queue", () => {
    expect(eligibleCarrierSettlements([{ id: "d", status: "Draft" }], "Approve")).toEqual([]);
  });

  it("posts only approved settlements", () => {
    expect(eligibleCarrierSettlements(rows, "Post").map((row) => row.id)).toEqual(["3"]);
  });

  it("marks paid only posted settlements", () => {
    expect(eligibleCarrierSettlements(rows, "MarkPaid").map((row) => row.id)).toEqual(["4"]);
  });

  it("returns an empty list when nothing qualifies", () => {
    expect(eligibleCarrierSettlements([{ id: "x", status: "Voided" }], "Submit")).toEqual([]);
  });

  it("never treats terminal statuses as eligible for any action", () => {
    for (const action of Object.keys(carrierLifecycleEligibility) as Array<
      keyof typeof carrierLifecycleEligibility
    >) {
      const eligible = eligibleCarrierSettlements(rows, action).map((row) => row.status);
      expect(eligible).not.toContain("Paid");
      expect(eligible).not.toContain("Voided");
    }
  });

  it("has a verb for every action", () => {
    for (const action of Object.keys(carrierLifecycleEligibility)) {
      expect(carrierLifecycleVerbs[action as keyof typeof carrierLifecycleVerbs]).toBeTruthy();
    }
  });
});
