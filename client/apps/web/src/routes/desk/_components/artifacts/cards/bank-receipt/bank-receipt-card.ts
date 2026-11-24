import type {
  BankReceiptCard,
  BankReceiptCardCandidate,
  BankReceiptCardPayment,
  BankReceiptCardWorkItem,
  BankReceiptStatus,
} from "@/types/bank-receipt";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardCheckState, CardStat, CardTone } from "../../record-card/record-card";
import { cardDay, minorMoney } from "../../record-card/record-card-values";

/** Where the receipt stands, in the reconciliation page's words and colours. */
export function receiptStanding(
  status: BankReceiptStatus,
  t: TranslateFn,
): { label: string; tone: CardTone } {
  switch (status) {
    case "Imported":
      return { label: t("Imported"), tone: "muted" };
    case "Matched":
      return { label: t("Matched"), tone: "success" };
    case "Exception":
      return { label: t("Exception"), tone: "danger" };
  }
}

/** The receipt's name: the reference the bank gave it, else said to have none. */
export function receiptTitle(card: BankReceiptCard, t: TranslateFn): string {
  return card.receipt.referenceNumber.trim() || t("Receipt with no reference");
}

/** What the bank wrote beside the money, when it wrote anything. */
export function receiptMemo(card: BankReceiptCard): string | undefined {
  return card.receipt.memo?.trim() || undefined;
}

/** The facts under the status: when the money came in, and when it was matched. */
export function receiptFacts(card: BankReceiptCard, t: TranslateFn): string[] {
  const { receipt } = card;
  return [
    t("Received {0}", cardDay(receipt.receiptDate)),
    receipt.status === "Matched" && receipt.matchedAt
      ? t("Matched {0}", cardDay(receipt.matchedAt))
      : "",
  ];
}

/** Why the receipt is or is not applied, as one check. */
export type ReceiptCheck = { state: CardCheckState; title: string; detail: string };

export function receiptCheck(card: BankReceiptCard, t: TranslateFn): ReceiptCheck {
  const { receipt } = card;
  switch (receipt.status) {
    case "Exception":
      return {
        state: "bad",
        title: t("No posted payment matches it"),
        detail:
          receipt.exceptionReason?.trim() ||
          t("It was not matched to a posted customer payment when it was imported."),
      };
    case "Imported":
      return {
        state: "info",
        title: t("Not reconciled yet"),
        detail: t("It has been imported and not yet compared with the posted payments."),
      };
    case "Matched":
      return {
        state: "ok",
        title: t("Applied through a posted payment"),
        detail: card.payment
          ? t(
              "Matched to {0}'s payment of {1}",
              paymentPayer(card.payment, t),
              paymentAmount(card.payment),
            )
          : card.paymentHidden
            ? t("It is matched to a customer payment you may not read.")
            : t("The payment it was matched to is gone."),
      };
  }
}

/** Who paid, by the customer on the payment, else said to be unknown. */
export function paymentPayer(payment: BankReceiptCardPayment, t: TranslateFn): string {
  return payment.customerName.trim() || t("an unknown customer");
}

export function paymentAmount(payment: BankReceiptCardPayment): string {
  return minorMoney(payment.amountMinor, payment.currencyCode || "USD");
}

/** How the payment was made, in words. */
export function paymentMethodLabel(method: string, t: TranslateFn): string {
  switch (method) {
    case "ACH":
      return t("ACH");
    case "Check":
      return t("Check");
    case "Wire":
      return t("Wire");
    case "Card":
      return t("Card");
    case "Cash":
      return t("Cash");
    default:
      return t("Payment");
  }
}

/** The payment line under the customer's name: how it was paid, its reference and its day. */
export function paymentLine(payment: BankReceiptCardPayment, t: TranslateFn): string {
  const method = paymentMethodLabel(payment.paymentMethod, t);
  const reference = payment.referenceNumber.trim();
  const day = cardDay(payment.paymentDate);
  return reference
    ? t("{0} {1}, paid {2}", method, reference, day)
    : t("{0}, paid {1}", method, day);
}

/** The payment's three figures: what it was for, what is applied to invoices, what is left. */
export function paymentStats(payment: BankReceiptCardPayment, t: TranslateFn): CardStat[] {
  const currency = payment.currencyCode || "USD";
  return [
    { key: "amount", label: t("Payment"), value: minorMoney(payment.amountMinor, currency) },
    {
      key: "applied",
      label: t("Applied"),
      value: minorMoney(payment.appliedAmountMinor, currency),
    },
    {
      key: "unapplied",
      label: t("Not applied"),
      value: minorMoney(payment.unappliedAmountMinor, currency),
      state: payment.unappliedAmountMinor > 0 ? "warn" : undefined,
    },
  ];
}

/** One invoice the payment went to, with what was applied and any short pay. */
export function applicationLine(
  application: BankReceiptCardPayment["applications"][number],
  currency: string,
  t: TranslateFn,
): { name: string; amount: string; note: string } {
  return {
    name: application.invoiceNumber.trim() || t("An invoice you may not read"),
    amount: minorMoney(application.appliedAmountMinor, currency),
    note:
      application.shortPayAmountMinor > 0
        ? t("{0} short", minorMoney(application.shortPayAmountMinor, currency))
        : "",
  };
}

/** The queue entry's standing, in the reconciliation queue's words. */
export function workItemStanding(
  item: BankReceiptCardWorkItem,
  t: TranslateFn,
): { label: string; tone: CardTone } {
  switch (item.status) {
    case "Open":
      return { label: t("Open"), tone: "warn" };
    case "Assigned":
      return { label: t("Assigned"), tone: "progress" };
    case "InReview":
      return { label: t("In review"), tone: "warn" };
    case "Resolved":
      return { label: t("Resolved"), tone: "success" };
    case "Dismissed":
      return { label: t("Dismissed"), tone: "muted" };
    default:
      return { label: item.status, tone: "muted" };
  }
}

/** Who holds the queue entry, or that nobody does yet. */
export function workItemHolder(item: BankReceiptCardWorkItem, t: TranslateFn): string {
  const name = item.assignedTo.trim();
  if (!name) return t("Nobody has taken it yet");
  return item.assignedAt
    ? t("{0} since {1}", name, cardDay(item.assignedAt))
    : t("Held by {0}", name);
}

/** The open item a candidate is: the invoice by its number, a payment by its customer and reference. */
export function candidateName(candidate: BankReceiptCardCandidate, t: TranslateFn): string {
  if (candidate.kind === "Invoice") {
    return candidate.invoiceNumber.trim() || t("An invoice you may not read");
  }
  const customer = candidate.customerName.trim() || t("an unknown customer");
  const reference = candidate.paymentReference.trim();
  return reference ? t("{0}'s payment {1}", customer, reference) : t("{0}'s payment", customer);
}

/**
 * The receipt against the candidate's open amount, worded by what is true: to the cent only
 * when the amounts are equal and the receipt's own reference names it.
 */
export function candidateStanding(
  candidate: BankReceiptCardCandidate,
  t: TranslateFn,
): { label: string; tone: CardTone } {
  const gap = minorMoney(Math.abs(candidate.differenceMinor));
  switch (candidate.standing) {
    case "Exact":
      return { label: t("To the cent"), tone: "success" };
    case "SameAmount":
      return { label: t("Same amount"), tone: "progress" };
    case "Short":
      return { label: t("{0} short", gap), tone: "warn" };
    case "Over":
      return { label: t("{0} over", gap), tone: "warn" };
  }
}

/** Why the candidate is on the list, one phrase per piece of evidence the receipt carries. */
export function candidateEvidence(candidate: BankReceiptCardCandidate, t: TranslateFn): string[] {
  return candidate.evidence
    .map((code) => {
      switch (code) {
        case "ReferenceExact":
          return t("Its reference is the receipt's");
        case "ReferencePartial":
          return t("Its reference is part of the receipt's");
        case "NumberInReference":
          return t("The receipt's reference names it");
        case "NumberInMemo":
          return t("The bank's memo names it");
        case "AmountExact":
          return candidate.kind === "Invoice"
            ? t("Its open amount is the receipt's")
            : t("Its amount is the receipt's");
        case "AmountNear":
          return t("Its amount is within a dollar of the receipt's");
        case "DateNear":
          return t("Paid within three days of the receipt");
        case "Payer":
          return t("Its customer sent a posted payment like this receipt");
        default:
          return "";
      }
    })
    .filter(Boolean);
}

/** The candidate's open amount: the invoice's balance, or the payment's amount. */
export function candidateOpen(candidate: BankReceiptCardCandidate, t: TranslateFn): string {
  return candidate.kind === "Invoice"
    ? t("{0} open", minorMoney(candidate.openMinor))
    : t("{0} paid", minorMoney(candidate.openMinor));
}

/** What the card says of candidates it leaves out because the reader may not read their kind. */
export function candidatesWithheldNote(card: BankReceiptCard, t: TranslateFn): string {
  const invoices = card.candidatesWithheld.includes("Invoice");
  const payments = card.candidatesWithheld.includes("CustomerPayment");
  if (invoices && payments)
    return t("You may not read invoices or customer payments, so none are shown.");
  if (invoices) return t("You may not read invoices, so invoices it could pay are not shown.");
  if (payments)
    return t("You may not read customer payments, so payments it could match are not shown.");
  return "";
}
