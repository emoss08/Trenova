import type { PortalSettlementLine } from "@trenova/shared/lib/graphql/driver-portal";
import { describe, expect, it } from "vitest";

import {
  canWithdrawDispute,
  escrowProgressPercent,
  groupSettlementLines,
  isDeductionCategory,
  lineAmountVariant,
  lineDetail,
  settlementCategoryLabels,
  settlementCategoryOrder,
  signedLineAmountMinor,
} from "./settlement";

describe("groupSettlementLines", () => {
  it("groups lines in the canonical category order regardless of input order", () => {
    const groups = groupSettlementLines([
      { category: "Deduction" },
      { category: "Earning" },
      { category: "EscrowContribution" },
      { category: "Earning" },
    ]);

    expect(groups.map((group) => group.category)).toEqual([
      "Earning",
      "Deduction",
      "EscrowContribution",
    ]);
    expect(groups[0].lines).toHaveLength(2);
  });

  it("drops empty categories and categories outside the catalog", () => {
    const unknown = "SomethingElse" as PortalSettlementLine["category"];
    expect(groupSettlementLines([{ category: unknown }])).toEqual([]);
    expect(groupSettlementLines([])).toEqual([]);
  });

  it("has a label for every ordered category", () => {
    for (const category of settlementCategoryOrder) {
      expect(settlementCategoryLabels[category]).toBeTruthy();
    }
  });
});

describe("signed line amounts", () => {
  it("negates deduction-family categories", () => {
    for (const category of ["Deduction", "EscrowContribution", "AdvanceRecovery"] as const) {
      expect(isDeductionCategory(category)).toBe(true);
      expect(signedLineAmountMinor({ category, amountMinor: 12_500 })).toBe(-12_500);
      expect(lineAmountVariant(category)).toBe("negative");
    }
  });

  it("keeps earning-family categories positive", () => {
    for (const category of [
      "Earning",
      "Reimbursement",
      "GuaranteeTopUp",
      "CarryForward",
    ] as const) {
      expect(isDeductionCategory(category)).toBe(false);
      expect(signedLineAmountMinor({ category, amountMinor: 12_500 })).toBe(12_500);
      expect(lineAmountVariant(category)).toBe("neutral");
    }
  });
});

describe("lineDetail", () => {
  it("joins pro number and quantity times rate", () => {
    expect(lineDetail({ proNumber: "PRO-1", quantity: "412", rate: "0.55" })).toBe(
      "PRO-1 · 412 × 0.55",
    );
  });

  it("omits the quantity segment when quantity or rate is not positive", () => {
    expect(lineDetail({ proNumber: "PRO-1", quantity: "0", rate: "0.55" })).toBe("PRO-1");
    expect(lineDetail({ proNumber: "PRO-1", quantity: "412", rate: "0" })).toBe("PRO-1");
  });

  it("omits the pro number when empty", () => {
    expect(lineDetail({ proNumber: "", quantity: "2", rate: "50" })).toBe("2 × 50");
    expect(lineDetail({ proNumber: "", quantity: "0", rate: "0" })).toBe("");
  });
});

describe("canWithdrawDispute", () => {
  it("allows withdrawal only while open or in review", () => {
    expect(canWithdrawDispute("Open")).toBe(true);
    expect(canWithdrawDispute("InReview")).toBe(true);
    expect(canWithdrawDispute("Resolved")).toBe(false);
    expect(canWithdrawDispute("Denied")).toBe(false);
    expect(canWithdrawDispute("Withdrawn")).toBe(false);
  });
});

describe("escrowProgressPercent", () => {
  it("returns zero without a positive target", () => {
    expect(escrowProgressPercent(50_000, 0)).toBe(0);
    expect(escrowProgressPercent(50_000, -1)).toBe(0);
  });

  it("computes the funded percentage", () => {
    expect(escrowProgressPercent(25_000, 100_000)).toBe(25);
    expect(escrowProgressPercent(100_000, 100_000)).toBe(100);
  });

  it("clamps to the 0-100 range", () => {
    expect(escrowProgressPercent(150_000, 100_000)).toBe(100);
    expect(escrowProgressPercent(-5_000, 100_000)).toBe(0);
  });
});
