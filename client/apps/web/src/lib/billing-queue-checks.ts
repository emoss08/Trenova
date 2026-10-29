import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  BillingCheck,
  BillingCheckKey,
  BillingIssue,
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

/** The issue a failing check waits on, if it has one. */
export function issueOf(item: BillingQueueItem, check: BillingCheck): BillingIssue | null {
  if (!check.issueId) return null;
  return item.review?.issues.find((issue) => issue.id === check.issueId) ?? null;
}
