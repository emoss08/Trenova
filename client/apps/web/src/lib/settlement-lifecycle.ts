import type { BulkSettlementActionType } from "@trenova/graphql/generated/graphql";
import type { DriverSettlementStatus } from "@trenova/shared/types/driver-pay";

export const bulkActionEligibility: Record<BulkSettlementActionType, DriverSettlementStatus[]> = {
  Submit: ["Draft"],
  Approve: ["Draft", "PendingApproval"],
  Post: ["Approved"],
  MarkPaid: ["Posted"],
};

export const bulkActionVerbs: Record<BulkSettlementActionType, string> = {
  Submit: "submitted",
  Approve: "approved",
  Post: "posted",
  MarkPaid: "marked paid",
};

export const settlementLifecycleChoices = [
  {
    value: "Submit",
    label: "Submit for Approval",
    color: "#2563eb",
    description: "Moves drafts into the approval queue.",
  },
  {
    value: "Approve",
    label: "Approve",
    color: "#15803d",
    description: "Approves drafts and pending settlements, applying deduction side effects.",
  },
  {
    value: "Post",
    label: "Post to GL",
    color: "#9333ea",
    description: "Journalizes approved settlements to the general ledger.",
  },
] as const;

export function eligibleSettlements<T extends { status: string }>(
  rows: T[],
  action: BulkSettlementActionType,
): T[] {
  return rows.filter((row) =>
    bulkActionEligibility[action].includes(row.status as DriverSettlementStatus),
  );
}
