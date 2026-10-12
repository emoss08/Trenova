import { AGING_BUCKETS, type AgingBucketTotals } from "@/components/accounting/aging-buckets";
import {
  billingCycleChoices,
  choiceLabel,
  customerFuelSurchargeModeChoices,
  customerPaymentTermChoices,
  invoiceDeliveryChoices,
} from "@/lib/choices";
import type { DeskCustomer, DeskCustomerBilling } from "@/lib/graphql/customer-card";
import { cityStateLabel } from "@/lib/place";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardCheckState, CardTone } from "../../record-card/record-card";

/** Credit used past this share of the limit is worth a look. */
const CREDIT_WATCH = 0.75;
/** Credit used past this share of the limit is about to stop freight. */
const CREDIT_STOP = 0.9;
/** Paying this many days past terms on average is a habit, not a slip. */
const PAY_SLIP_DAYS = 15;
const NET_TERM = /^Net(\d+)$/u;

/** Where an account stands, as the card's head says it. */
export type AccountStanding = {
  tone: CardTone;
  mark: "dot" | "check" | "pause";
  label: string;
};

/**
 * The customer's standing, from the record's own status and its credit: an
 * inactive customer reads as inactive whatever its credit says, and a hold or
 * a suspension reads before a warning or a review.
 */
export function accountStanding(
  status: DeskCustomer["status"],
  creditStatus: DeskCustomerBilling["creditStatus"] | undefined,
  t: TranslateFn,
): AccountStanding {
  if (status !== "Active") {
    return { tone: "muted", mark: "pause", label: t("Inactive") };
  }
  switch (creditStatus) {
    case "Hold":
      return { tone: "warn", mark: "pause", label: t("On credit hold") };
    case "Suspended":
      return { tone: "muted", mark: "pause", label: t("Credit suspended") };
    case "Warning":
      return { tone: "warn", mark: "dot", label: t("Credit warning") };
    case "Review":
      return { tone: "progress", mark: "dot", label: t("Credit under review") };
    default:
      return { tone: "success", mark: "check", label: t("In good standing") };
  }
}

/** The customer's address on one line, as an envelope would carry it. */
export function customerAddress(customer: DeskCustomer): string {
  const place = cityStateLabel(customer.city, customer.state?.abbreviation);
  const locality = [place, customer.postalCode].filter(Boolean).join(" ");
  return [customer.addressLine1, customer.addressLine2, locality]
    .map((part) => part?.trim() ?? "")
    .filter((part) => part !== "")
    .join(", ");
}

/** The days a payment term gives, or null for a term that is not counted in days. */
export function paymentTermDays(term: DeskCustomerBilling["paymentTerm"]): number | null {
  if (term === "DueOnReceipt") return 0;
  const days = NET_TERM.exec(term)?.[1];
  return days === undefined ? null : Number(days);
}

/**
 * How the customer pays against their terms: on time, slipping, or late as a
 * habit. Nothing is judged without a history of payments or a term in days.
 */
export function payPace(
  avgDaysToPay: number,
  term: DeskCustomerBilling["paymentTerm"] | undefined,
): CardCheckState | null {
  const days = term === undefined ? null : paymentTermDays(term);
  if (days === null || avgDaysToPay <= 0) return null;
  if (avgDaysToPay <= days) return "ok";
  return avgDaysToPay - days <= PAY_SLIP_DAYS ? "warn" : "bad";
}

/** One stretch of the aging strip: how much is that far past due, and its share of what is open. */
export type AgingStretch = {
  key: string;
  label: string;
  color: string;
  minor: number;
  share: number;
};

/** What is open, aged into the buckets the receivables pages use, each with its share. */
export function agingStretches(buckets: AgingBucketTotals, t: TranslateFn): AgingStretch[] {
  const total = AGING_BUCKETS.reduce((sum, bucket) => sum + Math.max(0, buckets[bucket.key]), 0);
  return AGING_BUCKETS.map((bucket) => {
    const minor = Math.max(0, buckets[bucket.key]);
    return {
      key: bucket.key,
      label: t(bucket.label),
      color: bucket.color,
      minor,
      share: total > 0 ? minor / total : 0,
    };
  });
}

/** How much of a credit limit is used, and how near the limit stops freight. */
export type CreditUse = { share: number; state: CardCheckState };

/** Credit used against a limit, or null for a customer with no limit to measure against. */
export function creditUse(used: number, limit: number | null | undefined): CreditUse | null {
  if (limit === null || limit === undefined || !Number.isFinite(limit) || limit <= 0) return null;
  const share = Math.max(0, used) / limit;
  return {
    share,
    state: share >= CREDIT_STOP ? "bad" : share >= CREDIT_WATCH ? "warn" : "ok",
  };
}

/** One month of payments received, its month's name, and its height against the best month shown. */
export type CollectionBar = { monthStart: number; label: string; minor: number; height: number };

/** A month's start is a UTC midnight, so it is named in UTC or it reads as the month before. */
const MONTH_NAME: Intl.DateTimeFormatOptions = { month: "short", timeZone: "UTC" };

/**
 * The last `count` calendar months up to the one `now` (unix seconds) falls
 * in, oldest first, each with what was collected in it. The server sends only
 * the months that had a payment; a quiet month is drawn as a quiet month.
 */
export function collectionBars(
  months: readonly { monthStart: number; amountMinor: number }[],
  now: number,
  count: number,
): CollectionBar[] {
  const collected = new Map<number, number>();
  for (const month of months) {
    const start = utcMonthStart(month.monthStart);
    collected.set(start, (collected.get(start) ?? 0) + Math.max(0, month.amountMinor));
  }

  const current = new Date(now * 1000);
  const bars: Omit<CollectionBar, "height">[] = [];
  for (let back = count - 1; back >= 0; back--) {
    const start = Date.UTC(current.getUTCFullYear(), current.getUTCMonth() - back, 1) / 1000;
    bars.push({
      monthStart: start,
      label: new Date(start * 1000).toLocaleDateString(undefined, MONTH_NAME),
      minor: collected.get(start) ?? 0,
    });
  }

  const best = bars.reduce((most, bar) => Math.max(most, bar.minor), 0);
  return bars.map((bar) => ({ ...bar, height: best > 0 ? bar.minor / best : 0 }));
}

function utcMonthStart(seconds: number): number {
  const day = new Date(seconds * 1000);
  return Date.UTC(day.getUTCFullYear(), day.getUTCMonth(), 1) / 1000;
}

/** One of the customer's billing terms, named the way the customer page names it. */
export type BillingTerm = { key: string; label: string; value: string };

export function billingTerms(billing: DeskCustomerBilling, t: TranslateFn): BillingTerm[] {
  return [
    {
      key: "term",
      label: t("Payment terms"),
      value: choiceLabel(customerPaymentTermChoices, billing.paymentTerm),
    },
    {
      key: "cycle",
      label: t("Billing cycle"),
      value: choiceLabel(billingCycleChoices, billing.billingCycle),
    },
    {
      key: "delivery",
      label: t("Invoices"),
      value: [
        choiceLabel(invoiceDeliveryChoices, billing.invoiceDelivery),
        invoiceChannel(billing, t),
      ].join(" · "),
    },
    {
      key: "fuel",
      label: t("Fuel"),
      value: choiceLabel(customerFuelSurchargeModeChoices, billing.fuelSurchargeMode),
    },
  ];
}

function invoiceChannel(billing: DeskCustomerBilling, t: TranslateFn): string {
  if (billing.emailInvoiceEnabled && billing.ediInvoiceEnabled) return t("Email and EDI");
  if (billing.ediInvoiceEnabled) return t("EDI");
  if (billing.emailInvoiceEnabled) return t("Email");
  return t("Not sent");
}

/** What every one of the customer's shipments must carry before it can be billed. */
export function invoiceNeeds(billing: DeskCustomerBilling, t: TranslateFn): string[] {
  const numbers = [
    billing.requirePONumber ? t("PO number") : "",
    billing.requireBOLNumber ? t("BOL number") : "",
    billing.requireDeliveryNumber ? t("Delivery number") : "",
  ];
  const documents = (billing.documentTypes ?? []).map((type) => type.name);
  return [...numbers, ...documents].filter((need) => need !== "");
}

/** The rules that change what an invoice comes to or how it goes out. */
export function billingRules(billing: DeskCustomerBilling, t: TranslateFn): string[] {
  const rate = Number(billing.lateChargeRate);
  return [
    billing.autoBill ? t("Bills automatically") : t("Billed by hand"),
    billing.taxExempt ? t("Tax exempt") : "",
    billing.applyLateCharges && Number.isFinite(rate) && rate > 0
      ? t(
          "{0}% late charge after {1, plural, one {# day} other {# days}}",
          rate,
          billing.gracePeriodDays,
        )
      : "",
    billing.enforceCreditLimit ? t("Credit limit enforced") : "",
    billing.billingCurrency && billing.billingCurrency !== "USD"
      ? t("Billed in {0}", billing.billingCurrency)
      : "",
  ].filter((rule) => rule !== "");
}
