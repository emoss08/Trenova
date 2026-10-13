import type { BadgeTone } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type {
  CollectionActivityKind,
  CollectionPriority,
  CollectionPromise,
  PromiseStatus,
} from "@/types/collection";

/** A promise is judged by what was paid since it was made, so the tone follows it. */
export const PROMISE_STATUS_TONE: Record<PromiseStatus, BadgeTone> = {
  Open: "info",
  Kept: "success",
  Broken: "danger",
};

export const PROMISE_STATUS_LABEL: Record<PromiseStatus, string> = defineLabels({
  Open: "Promised",
  Kept: "Kept",
  Broken: "Broken",
});

export const PRIORITY_LABEL: Record<CollectionPriority, string> = defineLabels({
  Critical: "Critical",
  High: "High",
  Normal: "Normal",
  Low: "Low",
});

export const CREDIT_STATUS_LABEL: Record<string, string> = defineLabels({
  Active: "Good standing",
  Warning: "Near limit",
  Hold: "On hold",
  Suspended: "Suspended",
  Review: "Under review",
});

/** Priority is a severity ordering, so it spends the severity tones. */
export const PRIORITY_TONE: Record<CollectionPriority, BadgeTone> = {
  Critical: "danger",
  High: "warning",
  Normal: "neutral",
  Low: "neutral",
};

export const ACTIVITY_KIND_LABEL: Record<CollectionActivityKind, string> = defineLabels({
  Note: "Note",
  Call: "Call",
  Email: "Email",
  PromiseToPay: "Promise to pay",
  CollectorAssigned: "Collector",
  FollowUpScheduled: "Follow-up",
});

/** How far a promise has been paid, as a whole percentage capped at 100. */
export function promisePaidShare(
  promise: Pick<CollectionPromise, "amountMinor" | "paidSinceMinor">,
) {
  if (promise.amountMinor <= 0) return 0;
  return Math.min(100, Math.floor((promise.paidSinceMinor / promise.amountMinor) * 100));
}
