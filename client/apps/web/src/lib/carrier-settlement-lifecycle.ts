import type { BulkOutcomeMessages } from "@/lib/bulk-outcome";
import { translate } from "@trenova/shared/i18n/runtime";
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

export function carrierLifecycleMessages(
  action: CarrierSettlementLifecycleAction,
): BulkOutcomeMessages {
  switch (action) {
    case "Submit":
      return {
        succeeded: (count) =>
          translate(
            "{0, plural, one {# settlement submitted} other {# settlements submitted}}",
            count,
          ),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# settlement submitted} other {# settlements submitted}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected settlement failed} other {All # selected settlements failed}}",
            failed,
          ),
      };
    case "Approve":
      return {
        succeeded: (count) =>
          translate(
            "{0, plural, one {# settlement approved} other {# settlements approved}}",
            count,
          ),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# settlement approved} other {# settlements approved}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected settlement failed} other {All # selected settlements failed}}",
            failed,
          ),
      };
    case "Post":
      return {
        succeeded: (count) =>
          translate("{0, plural, one {# settlement posted} other {# settlements posted}}", count),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# settlement posted} other {# settlements posted}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected settlement failed} other {All # selected settlements failed}}",
            failed,
          ),
      };
    default:
      return {
        succeeded: (count) =>
          translate(
            "{0, plural, one {# settlement marked paid} other {# settlements marked paid}}",
            count,
          ),
        partial: (succeeded, failed) =>
          translate(
            "{0, plural, one {# settlement marked paid} other {# settlements marked paid}}, {1} failed",
            succeeded,
            failed,
          ),
        allFailed: (failed) =>
          translate(
            "{0, plural, one {The selected settlement failed} other {All # selected settlements failed}}",
            failed,
          ),
      };
  }
}

export function eligibleCarrierSettlements<T extends { status: string }>(
  rows: T[],
  action: CarrierSettlementLifecycleAction,
): T[] {
  return rows.filter((row) =>
    carrierLifecycleEligibility[action].includes(row.status as CarrierSettlementStatus),
  );
}
