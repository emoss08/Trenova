import type { CarrierSettlementStatus } from "@trenova/shared/types/carrier-settlement";

export type CarrierSettlementLifecycleAction = "Submit" | "Approve" | "Post" | "MarkPaid";

// Mirrors the server's carrier settlement transition matrix: approval is only
// valid from PendingApproval — drafts must be submitted first (the driver-side
// Draft→Approve shortcut does not exist for carrier settlements).
export const carrierLifecycleEligibility: Record<
  CarrierSettlementLifecycleAction,
  CarrierSettlementStatus[]
> = {
  Submit: ["Draft"],
  Approve: ["PendingApproval"],
  Post: ["Approved"],
  MarkPaid: ["Posted"],
};

export const carrierLifecycleVerbs: Record<CarrierSettlementLifecycleAction, string> = {
  Submit: "submitted",
  Approve: "approved",
  Post: "posted",
  MarkPaid: "marked paid",
};

export function eligibleCarrierSettlements<T extends { status: string }>(
  rows: T[],
  action: CarrierSettlementLifecycleAction,
): T[] {
  return rows.filter((row) =>
    carrierLifecycleEligibility[action].includes(row.status as CarrierSettlementStatus),
  );
}
