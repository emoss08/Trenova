import type { PortalSettlementLine } from "@trenova/shared/lib/graphql/driver-portal";

export const settlementCategoryOrder = [
  "Earning",
  "GuaranteeTopUp",
  "Reimbursement",
  "Adjustment",
  "CarryForward",
  "Deduction",
  "EscrowContribution",
  "AdvanceRecovery",
] as const;

export type SettlementCategory = (typeof settlementCategoryOrder)[number];

export const settlementCategoryLabels: Record<string, string> = {
  Earning: "Earnings",
  GuaranteeTopUp: "Guarantee top-up",
  Reimbursement: "Reimbursements",
  Adjustment: "Adjustments",
  CarryForward: "Carry forward",
  Deduction: "Deductions",
  EscrowContribution: "Escrow",
  AdvanceRecovery: "Advance recovery",
};

const deductionCategories = new Set<string>([
  "Deduction",
  "EscrowContribution",
  "AdvanceRecovery",
]);

export function isDeductionCategory(category: string): boolean {
  return deductionCategories.has(category);
}

export type SettlementLineGroup<T extends Pick<PortalSettlementLine, "category">> = {
  category: SettlementCategory;
  lines: T[];
};

export function groupSettlementLines<T extends Pick<PortalSettlementLine, "category">>(
  lines: readonly T[],
): SettlementLineGroup<T>[] {
  return settlementCategoryOrder
    .map((category) => ({
      category,
      lines: lines.filter((line) => line.category === category),
    }))
    .filter((group) => group.lines.length > 0);
}

export function signedLineAmountMinor(
  line: Pick<PortalSettlementLine, "category" | "amountMinor">,
): number {
  return isDeductionCategory(line.category) ? -line.amountMinor : line.amountMinor;
}

export function lineAmountVariant(category: string): "negative" | "neutral" {
  return isDeductionCategory(category) ? "negative" : "neutral";
}

export function lineDetail(
  line: Pick<PortalSettlementLine, "proNumber" | "quantity" | "rate">,
): string {
  return [
    line.proNumber || null,
    Number(line.quantity) > 0 && Number(line.rate) > 0 ? `${line.quantity} × ${line.rate}` : null,
  ]
    .filter(Boolean)
    .join(" · ");
}

export function canWithdrawDispute(status: string): boolean {
  return status === "Open" || status === "InReview";
}

export function escrowProgressPercent(balanceMinor: number, targetAmountMinor: number): number {
  if (targetAmountMinor <= 0) return 0;
  return Math.min(100, Math.max(0, (balanceMinor / targetAmountMinor) * 100));
}
