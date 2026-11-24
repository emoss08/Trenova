import type { DriverSettlementHoldReason } from "@trenova/graphql/generated/graphql";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";

export const HOLD_REASON_LABELS = defineLabels({
  RateDispute: "Rate dispute",
  MissingPaperwork: "Missing paperwork",
  NegativeNet: "Negative net",
  DeductionReview: "Deduction under review",
  Other: "Other",
} satisfies Record<DriverSettlementHoldReason, string>);

export const HOLD_REASONS: readonly DriverSettlementHoldReason[] = [
  "RateDispute",
  "MissingPaperwork",
  "NegativeNet",
  "DeductionReview",
  "Other",
];

/** The reasons whose hold needs a note saying what it waits on. */
export const HOLD_REASONS_NEEDING_A_NOTE: readonly DriverSettlementHoldReason[] = ["Other"];

export const MAX_HOLD_NOTE_LENGTH = 500;

/** The statuses a settlement may be held in: a posted one is paid through its journal. */
export const HOLDABLE_STATUSES: readonly string[] = ["Draft", "PendingApproval", "Approved"];

export type HoldState = {
  status: string;
  heldAt?: number | null;
};

/** A settlement is held while it has a moment it was held at; the reason may be withheld. */
export function isHeld(settlement: { heldAt?: number | null }): boolean {
  return settlement.heldAt != null;
}

export function canBeHeld(settlement: HoldState): boolean {
  return !isHeld(settlement) && HOLDABLE_STATUSES.includes(settlement.status);
}

export function holdReasonLabel(reason: DriverSettlementHoldReason | null | undefined): string {
  return reason ? HOLD_REASON_LABELS[reason] : "";
}

/** Said wherever a bulk action leaves held settlements out, so none is skipped unseen. */
export function heldLeftOut(count: number): string {
  return translate(
    "{0, plural, one {# settlement on hold was left out} other {# settlements on hold were left out}}",
    count,
  );
}
