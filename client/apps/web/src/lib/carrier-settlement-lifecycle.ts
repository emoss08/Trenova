import type { CarrierSettlementStatus } from "@trenova/shared/types/carrier-settlement";

export type CarrierSettlementLifecycleAction = "Submit" | "Approve" | "Post" | "MarkPaid";

export const carrierLifecycleEligibility: Record<
  CarrierSettlementLifecycleAction,
  CarrierSettlementStatus[]
> = {
  Submit: ["Draft"],
  Approve: ["Draft", "PendingApproval"],
  Post: ["Approved"],
  MarkPaid: ["Posted"],
};

export const carrierLifecycleVerbs: Record<CarrierSettlementLifecycleAction, string> = {
  Submit: "submitted",
  Approve: "approved",
  Post: "posted",
  MarkPaid: "marked paid",
};

export const carrierSettlementLifecycleChoices = [
  {
    value: "Submit",
    label: "Submit for Approval",
    color: "#2563eb",
    description: "Moves draft carrier statements into the approval queue.",
  },
  {
    value: "Approve",
    label: "Approve",
    color: "#15803d",
    description: "Approves draft and pending carrier settlements, locking their totals.",
  },
  {
    value: "Post",
    label: "Post to GL",
    color: "#9333ea",
    description:
      "Journalizes approved settlements — purchased transportation against accounts payable.",
  },
] as const;

export function eligibleCarrierSettlements<T extends { status: string }>(
  rows: T[],
  action: CarrierSettlementLifecycleAction,
): T[] {
  return rows.filter((row) =>
    carrierLifecycleEligibility[action].includes(row.status as CarrierSettlementStatus),
  );
}
