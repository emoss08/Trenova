import { describe, expect, it } from "vitest";
import { canBeHeld, heldLeftOut, holdReasonLabel, isHeld } from "../settlement-hold";
import { eligibleSettlements, heldOutSettlements } from "../settlement-lifecycle";

const rows = [
  { id: "draft", status: "Draft", heldAt: null },
  { id: "held-draft", status: "Draft", heldAt: 1_790_000_000 },
  { id: "pending", status: "PendingApproval", heldAt: null },
  { id: "held-pending", status: "PendingApproval", heldAt: 1_790_000_000 },
  { id: "approved", status: "Approved", heldAt: null },
  { id: "held-approved", status: "Approved", heldAt: 1_790_000_000 },
  { id: "posted", status: "Posted", heldAt: null },
];

const ids = (list: { id: string }[]) => list.map((row) => row.id);

describe("bulk actions and holds", () => {
  it("leaves held settlements out of approving, posting and paying", () => {
    expect(ids(eligibleSettlements(rows, "Approve"))).toEqual(["draft", "pending"]);
    expect(ids(eligibleSettlements(rows, "Post"))).toEqual(["approved"]);
    expect(ids(eligibleSettlements(rows, "MarkPaid"))).toEqual(["posted"]);
  });

  it("names the held settlements each action leaves out, so none is skipped unseen", () => {
    expect(ids(heldOutSettlements(rows, "Approve"))).toEqual(["held-draft", "held-pending"]);
    expect(ids(heldOutSettlements(rows, "Post"))).toEqual(["held-approved"]);
    expect(heldOutSettlements(rows, "MarkPaid")).toEqual([]);
  });

  it("still submits a held draft for approval: a hold waits on paying, not on review", () => {
    expect(ids(eligibleSettlements(rows, "Submit"))).toEqual(["draft", "held-draft"]);
    expect(heldOutSettlements(rows, "Submit")).toEqual([]);
  });

  it("says how many were left out", () => {
    expect(heldLeftOut(1)).toBe("1 settlement on hold was left out");
    expect(heldLeftOut(3)).toBe("3 settlements on hold were left out");
  });
});

describe("whether a settlement is held", () => {
  it("is held while it has a moment it was held at, whether or not the reason may be read", () => {
    expect(isHeld({ heldAt: 1_790_000_000 })).toBe(true);
    expect(isHeld({ heldAt: null })).toBe(false);
    expect(isHeld({})).toBe(false);
  });

  it("can be held only until it is posted, and not twice", () => {
    expect(canBeHeld({ status: "Draft", heldAt: null })).toBe(true);
    expect(canBeHeld({ status: "PendingApproval" })).toBe(true);
    expect(canBeHeld({ status: "Approved", heldAt: null })).toBe(true);
    expect(canBeHeld({ status: "Approved", heldAt: 1 })).toBe(false);
    expect(canBeHeld({ status: "Posted", heldAt: null })).toBe(false);
    expect(canBeHeld({ status: "Paid", heldAt: null })).toBe(false);
    expect(canBeHeld({ status: "Voided", heldAt: null })).toBe(false);
  });

  it("names each reason in words, never its code", () => {
    expect(holdReasonLabel("RateDispute")).toBe("Rate dispute");
    expect(holdReasonLabel("DeductionReview")).toBe("Deduction under review");
    expect(holdReasonLabel(null)).toBe("");
  });
});
