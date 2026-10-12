import { choiceLabel, orderStatusChoices } from "@/lib/choices";
import type { DeskOrder, DeskOrderCharge } from "@/lib/graphql/order-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardFact, CardStat, CardTone } from "../../record-card/record-card";
import { cardDate, money } from "../../record-card/record-card-values";

type OrderStatus = DeskOrder["status"];

/** The way an order moves, from taken to closed; a canceled order leaves the track. */
export const ORDER_LIFECYCLE = [
  "Draft",
  "Confirmed",
  "InProgress",
  "Completed",
  "Billed",
  "Closed",
] as const satisfies readonly OrderStatus[];

/** One step of the order's way, behind it, where it is, or ahead. */
export type LifecycleStep = {
  key: OrderStatus;
  label: string;
  state: "done" | "now" | "next" | "off";
};

/**
 * The order's way from draft to closed, each step marked done, now or next,
 * so the card shows how far its legs have carried it. A canceled order shows
 * the whole way off.
 */
export function lifecycleSteps(status: OrderStatus): LifecycleStep[] {
  const at = (ORDER_LIFECYCLE as readonly OrderStatus[]).indexOf(status);
  return ORDER_LIFECYCLE.map((key, index) => ({
    key,
    label: choiceLabel(orderStatusChoices, key),
    state: status === "Canceled" ? "off" : index < at ? "done" : index === at ? "now" : "next",
  }));
}

/** Where the order stands, as the pill at the card's head says it. */
export function orderStanding(status: OrderStatus): { label: string; tone: CardTone } {
  const label = choiceLabel(orderStatusChoices, status);
  switch (status) {
    case "Draft":
      return { label, tone: "draft" };
    case "Confirmed":
    case "InProgress":
      return { label, tone: "progress" };
    case "Completed":
    case "Billed":
      return { label, tone: "success" };
    case "Closed":
    case "Canceled":
      return { label, tone: "muted" };
  }
}

/** What the order is worth: its total, else what was quoted, with which it is. */
export function orderWorth(
  order: Pick<DeskOrder, "totalAmount" | "quotedAmount" | "currencyCode">,
  t: TranslateFn,
): { amount: string | null; caption: string } {
  if (order.totalAmount !== null && order.totalAmount !== undefined) {
    return { amount: money(order.totalAmount, order.currencyCode), caption: t("order total") };
  }
  if (order.quotedAmount !== null && order.quotedAmount !== undefined) {
    return { amount: money(order.quotedAmount, order.currencyCode), caption: t("quoted") };
  }
  return { amount: null, caption: t("Not priced yet") };
}

/** A decimal amount in cents, so charges add up without drifting. */
function cents(amount: string | null | undefined): number {
  const value = Number(amount ?? 0);
  return Number.isFinite(value) ? Math.round(value * 100) : 0;
}

/** What the order's charges come to: in all, already invoiced, and still to invoice. */
export function chargeStats(
  charges: readonly DeskOrderCharge[],
  currency: string,
  t: TranslateFn,
): CardStat[] {
  const total = charges.reduce((sum, charge) => sum + cents(charge.amount), 0);
  const invoiced = charges
    .filter((charge) => charge.invoicedAt)
    .reduce((sum, charge) => sum + cents(charge.amount), 0);
  return [
    { key: "total", label: t("Charges"), value: money(total / 100, currency) },
    { key: "invoiced", label: t("Invoiced"), value: money(invoiced / 100, currency) },
    { key: "open", label: t("Still to invoice"), value: money((total - invoiced) / 100, currency) },
  ];
}

/** A charge as its row reads it: whether it has been invoiced, and how it is split. */
export function chargeLine(charge: DeskOrderCharge, t: TranslateFn): string {
  return [
    charge.invoicedAt
      ? t("Invoiced {0}", cardDate(charge.invoicedAt))
      : charge.invoiceId
        ? t("Partly invoiced")
        : t("Not yet invoiced"),
    charge.allocations.length > 1
      ? t("Split {0, plural, one {# way} other {# ways}}", charge.allocations.length)
      : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** The facts the card lists about the order, in the order the office reads them. */
export function orderFacts(order: DeskOrder, t: TranslateFn): CardFact[] {
  return [
    { key: "number", label: t("Order number"), value: order.orderNumber },
    { key: "po", label: t("PO number"), value: order.poNumber?.trim() || t("None") },
    { key: "bol", label: t("BOL"), value: order.bol?.trim() || t("None") },
    { key: "currency", label: t("Currency"), value: order.currencyCode },
    {
      key: "quoted",
      label: t("Quoted"),
      value:
        order.quotedAmount !== null && order.quotedAmount !== undefined
          ? money(order.quotedAmount, order.currencyCode)
          : t("Not quoted"),
    },
    { key: "created", label: t("Taken"), value: cardDate(order.createdAt) },
  ];
}
