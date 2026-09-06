import type { BulkOutcome } from "@/lib/bulk-outcome";
import type {
  PTOBulkAction,
  PTOBulkActionPayload,
  PTOStatus,
  WorkerPTO,
} from "@trenova/shared/types/worker";

export const PTO_ACTION_TARGET: Record<PTOBulkAction, PTOStatus> = {
  Approve: "Approved",
  Reject: "Rejected",
  Cancel: "Cancelled",
};

const ALLOWED_TRANSITIONS: Record<PTOStatus, readonly PTOStatus[]> = {
  Requested: ["Approved", "Rejected", "Cancelled"],
  Approved: ["Cancelled"],
  Rejected: [],
  Cancelled: [],
};

export function canTransitionPTO(from: PTOStatus, to: PTOStatus): boolean {
  return ALLOWED_TRANSITIONS[from]?.includes(to) ?? false;
}

export function canApplyPTOAction(status: PTOStatus, action: PTOBulkAction): boolean {
  return canTransitionPTO(status, PTO_ACTION_TARGET[action]);
}

export type PTOActionSplit<T extends Pick<WorkerPTO, "status">> = {
  eligible: T[];
  ineligible: T[];
};

export function splitPTOByAction<T extends Pick<WorkerPTO, "status">>(
  rows: readonly T[],
  action: PTOBulkAction,
): PTOActionSplit<T> {
  const eligible: T[] = [];
  const ineligible: T[] = [];
  for (const row of rows) {
    if (canApplyPTOAction(row.status, action)) {
      eligible.push(row);
    } else {
      ineligible.push(row);
    }
  }
  return { eligible, ineligible };
}

export const PTO_ACTION_LABELS: Record<
  PTOBulkAction,
  { label: string; loadingLabel: string; verbPast: string; noneEligible: string }
> = {
  Approve: {
    label: "Approve",
    loadingLabel: "Approving...",
    verbPast: "Approved",
    noneEligible: "Only requested PTO can be approved.",
  },
  Reject: {
    label: "Reject",
    loadingLabel: "Rejecting...",
    verbPast: "Rejected",
    noneEligible: "Only requested PTO can be rejected.",
  },
  Cancel: {
    label: "Cancel",
    loadingLabel: "Cancelling...",
    verbPast: "Cancelled",
    noneEligible: "Only requested or approved PTO can be cancelled.",
  },
};

export function ptoIds(rows: readonly Pick<WorkerPTO, "id">[]): string[] {
  const ids: string[] = [];
  for (const row of rows) {
    if (row.id) ids.push(row.id);
  }
  return ids;
}

export function bulkPayloadToOutcome(payload: PTOBulkActionPayload): BulkOutcome {
  const succeeded: string[] = [];
  const failed: BulkOutcome["failed"] = [];
  for (const result of payload.results) {
    if (result.success) {
      succeeded.push(result.ptoId);
    } else {
      failed.push({ id: result.ptoId, error: result.error });
    }
  }
  return { succeeded, failed };
}
