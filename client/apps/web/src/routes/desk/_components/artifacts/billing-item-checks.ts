import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  BillingCheck,
  BillingCheckKey,
  BillingIssue,
  BillingQueueHoldReason,
  BillingQueueItem,
  BillingQueueStatus,
  BillingQueueSummary,
} from "@trenova/shared/types/billing-queue";

/**
 * Where an item stands for the person looking at it: still to review, approved
 * and waiting to post, posted, held, or somewhere the desk only reports.
 */
export type ItemStage = "review" | "approved" | "posted" | "held" | "other";

export function itemStage(status: BillingQueueStatus): ItemStage {
  switch (status) {
    case "ReadyForReview":
    case "InReview":
      return "review";
    case "Approved":
      return "approved";
    case "Posted":
      return "posted";
    case "OnHold":
      return "held";
    default:
      return "other";
  }
}

export const HOLD_REASONS: readonly BillingQueueHoldReason[] = [
  "WaitingOnPaperwork",
  "CustomerDispute",
  "RateQuestion",
];

export function holdReasonLabel(
  reason: BillingQueueHoldReason | null | undefined,
  t: TranslateFn,
): string {
  switch (reason) {
    case "WaitingOnPaperwork":
      return t("Waiting on paperwork");
    case "CustomerDispute":
      return t("Customer dispute");
    case "RateQuestion":
      return t("Rate question");
    default:
      return "";
  }
}

/** The status pill's words, the way the queue table and the item both say it. */
export function stageLabel(
  status: BillingQueueStatus,
  hold: BillingQueueHoldReason | null | undefined,
  t: TranslateFn,
): string {
  switch (itemStage(status)) {
    case "review":
      return t("Ready for review");
    case "approved":
      return t("Approved");
    case "posted":
      return t("Posted");
    case "held": {
      const reason = holdReasonLabel(hold, t);
      return reason ? t("On hold · {0}", reason) : t("On hold");
    }
    default:
      switch (status) {
        case "Exception":
          return t("Exception");
        case "SentBackToOps":
          return t("Sent back to ops");
        case "Canceled":
          return t("Canceled");
        default:
          return status;
      }
  }
}

export function checkTitle(key: BillingCheckKey, t: TranslateFn): string {
  switch (key) {
    case "biller":
      return t("Biller");
    case "charges":
      return t("Charges match the rate con");
    case "pod":
      return t("Proof of delivery");
    case "terms":
      return t("Bill-to and terms");
    case "duplicate":
      return t("Not a duplicate");
  }
}

function shortDate(seconds: unknown): string {
  if (typeof seconds !== "number" || seconds <= 0) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

export function paymentTermLabel(term: string, t: TranslateFn): string {
  if (term === "DueOnReceipt") return t("Due on receipt");
  const days = /^Net(\d+)$/u.exec(term);
  return days ? t("Net {0}", Number(days[1])) : term;
}

/**
 * A check's finding in the person's language. The server decides each check
 * and names what it found with a code; the wording is the reader's. A code
 * the desk does not know yet reads as the server's own line.
 */
export function checkDetail(check: BillingCheck, t: TranslateFn): string {
  const facts = check.facts ?? {};
  const fact = (key: string) => {
    const value = facts[key];
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
  };
  switch (check.code) {
    case "unassigned":
      return t("Nobody is assigned");
    case "assigned":
      return typeof facts.biller === "string" && facts.biller !== "" ? facts.biller : check.detail;
    case "matches":
      return t("Every line is on the rate con");
    case "no_rate_con":
      return t("Billed as rated");
    case "signed": {
      const at = shortDate(facts.at);
      return at ? t("Signed · {0}", at) : t("Signed");
    }
    case "on_file":
      return t("On file");
    case "not_required":
      return t("Not required for this customer");
    case "terms": {
      const term = paymentTermLabel(fact("paymentTerm"), t);
      const contact = fact("contact");
      return contact ? `${contact} · ${term}` : term;
    }
    case "unique":
      return t("No other invoice for {0}", fact("shipment"));
    default:
      return check.detail;
  }
}

/**
 * Why the action bar's main button is not available, in a few words; empty
 * when it is. Approved items are one step from the customer, so the bar says
 * what posting does instead.
 */
export function barReason(item: BillingQueueItem, t: TranslateFn): string {
  const stage = itemStage(item.status);
  if (stage === "held") return t("Release the hold to continue");
  if (stage === "approved") return t("Posting sends it to the customer and can't be undone");
  if (stage !== "review") return "";
  switch (item.review?.blocker ?? "") {
    case "biller":
      return t("Assign a biller first");
    case "issue":
      return t("Settle the flagged check first");
    case "":
      return item.review ? "" : t("Checking…");
    default:
      return t("Not ready for review");
  }
}

/** The issue a failing check waits on, if it has one. */
export function issueOf(item: BillingQueueItem, check: BillingCheck): BillingIssue | null {
  if (!check.issueId) return null;
  return item.review?.issues.find((issue) => issue.id === check.issueId) ?? null;
}

export function initials(name: string): string {
  return name
    .split(/\s+/u)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "")
    .join("");
}

export function money(amount: string | number | null | undefined, currency = "USD"): string {
  const figure = Number(amount);
  if (amount === null || amount === undefined || amount === "" || !Number.isFinite(figure)) {
    return "—";
  }
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

/** How the selection in a queue table splits for the bulk bar. */
export type BulkSplit = {
  /** Selected items every check clears; Approve takes these. */
  ready: string[];
  /** Selected items still under review that need a person; Review opens the first. */
  needs: string[];
  /** Selected items already approved, posted, held or otherwise out of review. */
  other: number;
};

export function bulkSplit(
  selected: readonly string[],
  summaries: ReadonlyMap<string, BillingQueueSummary>,
): BulkSplit {
  const split: BulkSplit = { ready: [], needs: [], other: 0 };
  for (const id of selected) {
    const summary = summaries.get(id);
    if (!summary || itemStage(summary.status) !== "review") {
      split.other += 1;
    } else if (summary.ready) {
      split.ready.push(id);
    } else {
      split.needs.push(id);
    }
  }
  return split;
}

/** The header checkbox: every row on, some on, or none. */
export function selectionState(
  rowIds: readonly string[],
  selected: readonly string[],
): "all" | "some" | "none" {
  if (rowIds.length > 0 && rowIds.every((id) => selected.includes(id))) return "all";
  return selected.length > 0 ? "some" : "none";
}

export function toggleOne(selected: readonly string[], id: string): string[] {
  return selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id];
}

export function toggleAll(rowIds: readonly string[], selected: readonly string[]): string[] {
  return selectionState(rowIds, selected) === "all" ? [] : [...rowIds];
}

/**
 * Seconds left on a bulk approval's undo, counted to when the server commits
 * rather than from when the button was pressed, so a slow request never shows
 * more time than there is.
 */
export function undoSecondsLeft(commitAt: number, nowMs: number): number {
  return Math.max(0, Math.ceil(commitAt - nowMs / 1000));
}
