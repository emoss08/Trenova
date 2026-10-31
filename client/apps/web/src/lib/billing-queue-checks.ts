import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  BillingCheck,
  BillingCheckKey,
  BillingIssue,
  BillingQueueHoldReason,
  BillingQueueItem,
} from "@trenova/shared/types/billing-queue";

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
 * Issues that only suggest: a charge the shipment carries proof of but does not bill.
 * The charges show them; no check waits on them and they never hold approval.
 */
const SUGGESTION_CODES: ReadonlySet<string> = new Set(["UnbilledAccessorial"]);

export function isSuggestion(issue: BillingIssue): boolean {
  return SUGGESTION_CODES.has(issue.code);
}

/**
 * The issues a failing check waits on, oldest first: every open one of its
 * kind, so two flagged charges are both there to settle. A check that is
 * clear waits on none, even when it names the issue that settled it, and no
 * check waits on a suggestion.
 */
export function openIssuesOf(item: BillingQueueItem, check: BillingCheck): BillingIssue[] {
  if (check.state === "ok") return [];
  return (
    item.review?.issues.filter(
      (issue) => issue.checkKey === check.key && !issue.resolutionKey && !isSuggestion(issue),
    ) ?? []
  );
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
