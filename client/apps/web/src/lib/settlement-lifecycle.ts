import type { BulkSettlementActionType } from "@trenova/graphql/generated/graphql";
import type { DriverSettlementStatus } from "@trenova/shared/types/driver-pay";

export const bulkActionEligibility: Record<BulkSettlementActionType, DriverSettlementStatus[]> = {
  Submit: ["Draft"],
  Approve: ["Draft", "PendingApproval"],
  Post: ["Approved"],
  MarkPaid: ["Posted"],
};

export const settlementLifecycleChoices = [
  {
    value: "Submit",
    label: "Submit for approval",
    color: "var(--info)",
    description: "Moves drafts into the approval queue.",
  },
  {
    value: "Approve",
    label: "Approve",
    color: "var(--success)",
    description: "Approves drafts and pending settlements, applying deduction side effects.",
  },
  {
    value: "Post",
    label: "Post to GL",
    color: "var(--accent-violet)",
    description: "Journalizes approved settlements to the general ledger.",
  },
] as const;

/** The actions a hold refuses: each moves the settlement toward paying the driver. */
export const holdRefusedActions: readonly BulkSettlementActionType[] = [
  "Approve",
  "Post",
  "MarkPaid",
];

function inEligibleStatus(row: { status: string }, action: BulkSettlementActionType): boolean {
  return bulkActionEligibility[action].includes(row.status as DriverSettlementStatus);
}

function heldFor(row: { heldAt?: number | null }, action: BulkSettlementActionType): boolean {
  return row.heldAt != null && holdRefusedActions.includes(action);
}

/** The rows an action applies to: in a status it takes, and not on hold when the hold refuses it. */
export function eligibleSettlements<T extends { status: string; heldAt?: number | null }>(
  rows: T[],
  action: BulkSettlementActionType,
): T[] {
  return rows.filter((row) => inEligibleStatus(row, action) && !heldFor(row, action));
}

/** The rows an action would take but for their hold, so the bar can say they are left out. */
export function heldOutSettlements<T extends { status: string; heldAt?: number | null }>(
  rows: T[],
  action: BulkSettlementActionType,
): T[] {
  return rows.filter((row) => inEligibleStatus(row, action) && heldFor(row, action));
}
