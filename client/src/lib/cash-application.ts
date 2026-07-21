import type { AROpenItem } from "@/lib/graphql/accounts-receivable";
import type { CashApplicationRow } from "@/types/customer-payment";

export function toMinor(amount: number): number {
  return Math.round((Number.isFinite(amount) ? amount : 0) * 100);
}

export function openItemToApplicationRow(
  item: AROpenItem,
  checked: boolean,
): CashApplicationRow {
  return {
    invoiceId: item.invoiceId,
    invoiceNumber: item.invoiceNumber,
    invoiceDate: item.invoiceDate,
    dueDate: item.dueDate,
    daysPastDue: item.daysPastDue,
    openAmountMinor: item.openAmountMinor,
    checked,
    appliedAmount: 0,
    shortPayAmount: 0,
  };
}

export type CashApplicationTotals = {
  appliedMinor: number;
  shortPayMinor: number;
  unappliedMinor: number;
  overAppliedRows: number[];
  isOverBudget: boolean;
};

export function computeApplicationTotals(
  rows: CashApplicationRow[],
  budgetMinor: number,
): CashApplicationTotals {
  let appliedMinor = 0;
  let shortPayMinor = 0;
  const overAppliedRows: number[] = [];

  rows.forEach((row, index) => {
    if (!row.checked) return;
    const applied = toMinor(row.appliedAmount);
    const shortPay = toMinor(row.shortPayAmount);
    appliedMinor += applied;
    shortPayMinor += shortPay;
    if (applied + shortPay > row.openAmountMinor) {
      overAppliedRows.push(index);
    }
  });

  return {
    appliedMinor,
    shortPayMinor,
    unappliedMinor: Math.max(budgetMinor - appliedMinor, 0),
    overAppliedRows,
    isOverBudget: appliedMinor > budgetMinor,
  };
}

export function allocateBudget(
  rows: CashApplicationRow[],
  budgetMinor: number,
): CashApplicationRow[] {
  const hasChecked = rows.some((row) => row.checked);
  let remaining = Math.max(budgetMinor, 0);

  return rows.map((row) => {
    const eligible = hasChecked ? row.checked : true;
    if (!eligible || remaining <= 0) {
      return eligible ? { ...row, appliedAmount: 0, shortPayAmount: 0 } : row;
    }
    const appliedMinor = Math.min(row.openAmountMinor, remaining);
    remaining -= appliedMinor;
    return {
      ...row,
      checked: true,
      appliedAmount: appliedMinor / 100,
      shortPayAmount: 0,
    };
  });
}
