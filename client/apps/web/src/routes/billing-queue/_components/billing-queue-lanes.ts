import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BillingQueueLane } from "@trenova/shared/types/billing-queue";

/** Each open lane's place on the queue's bar and beside its heading. */
export const LANE_DOT: Record<BillingQueueLane, string> = {
  NeedsYou: "bg-warning",
  Ready: "bg-success",
  Waiting: "bg-foreground-subtle",
  Checking: "bg-border-strong",
  Held: "bg-foreground-muted",
  Approved: "bg-brand",
  Closed: "bg-border-strong",
};

/** Lanes that start folded: nobody works them item by item. */
export const LANES_FOLDED_BY_DEFAULT: readonly BillingQueueLane[] = ["Held", "Approved"];

export function laneLabel(lane: BillingQueueLane, t: TranslateFn): string {
  switch (lane) {
    case "NeedsYou":
      return t("Needs you");
    case "Ready":
      return t("Ready to approve");
    case "Waiting":
      return t("Waiting on others");
    case "Checking":
      return t("Being checked");
    case "Held":
      return t("On hold");
    case "Approved":
      return t("Approved, not posted");
    case "Closed":
      return t("Closed");
  }
}
