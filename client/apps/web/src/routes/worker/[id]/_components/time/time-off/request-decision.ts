import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import type { PTOStatus, WorkerPTO } from "@trenova/shared/types/worker";

export type RequestDecision =
  | { kind: "waiting" }
  | { kind: "decided"; actor: string | null; note: string | null };

type DecisionSource = Pick<
  WorkerPTO,
  | "status"
  | "autoApproved"
  | "approver"
  | "rejector"
  | "cancelledBy"
  | "rejectionReason"
  | "cancellationReason"
>;

function text(value: string | null | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

/**
 * What the Decision column says for one request: only a requested entry is waiting. Approved
 * names who approved it (or that it was approved automatically), rejected names who rejected it
 * and why, cancelled names who cancelled it and why when a reason was given. An actor the entry
 * does not carry is null, drawn as a dash.
 */
export function requestDecision(t: TranslateFn, entry: DecisionSource): RequestDecision {
  switch (entry.status) {
    case "Requested":
      return { kind: "waiting" };
    case "Approved":
      return {
        kind: "decided",
        actor:
          text(entry.approver?.name) ?? (entry.autoApproved ? t("Approved automatically") : null),
        note: null,
      };
    case "Rejected":
      return {
        kind: "decided",
        actor: text(entry.rejector?.name),
        note: text(entry.rejectionReason),
      };
    case "Cancelled":
      return {
        kind: "decided",
        actor: text(entry.cancelledBy?.name),
        note: text(entry.cancellationReason),
      };
    default:
      return { kind: "decided", actor: null, note: null };
  }
}

export const REQUEST_STATUS_LABELS = defineLabels({
  Requested: "Requested",
  Approved: "Approved",
  Rejected: "Rejected",
  Cancelled: "Cancelled",
});

export const REQUEST_STATUS_VARIANTS: Record<PTOStatus, BadgeVariant> = {
  Requested: "info",
  Approved: "success",
  Rejected: "danger",
  Cancelled: "neutral",
};
