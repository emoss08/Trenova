import { formatOrdinal } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { OpenStatement } from "@trenova/shared/types/statement";
import type {
  BillingCycle,
  InvoiceDelivery,
  InvoiceDetail,
  InvoiceSectionKey,
  InvoiceSplitKey,
} from "@trenova/shared/types/customer";
import { weekdayName } from "@/lib/cron";
import { defineLabels } from "@trenova/shared/i18n/labels";

export type BillingSchedule = {
  invoiceDelivery: InvoiceDelivery;
  billingCycle: BillingCycle;
  billingCycleAnchorDay: number;
  splitBy: InvoiceSplitKey;
  sectionBy: InvoiceSectionKey;
  invoiceDetail: InvoiceDetail;
  maxShipmentsPerInvoice: number;
};

const CADENCE_LABELS: Record<BillingCycle, string> = defineLabels({
  Immediate: "Per shipment",
  Daily: "Daily",
  Weekly: "Weekly",
  BiWeekly: "Bi-weekly",
  SemiMonthly: "Semi-monthly",
  Monthly: "Monthly",
  Quarterly: "Quarterly",
});

/** The cadence as a chip label — two words at most, for a card or a badge. */
export function cadenceLabel(cycle: BillingCycle): string {
  return CADENCE_LABELS[cycle] ?? "Per shipment";
}

const SPLIT_LABELS: Record<InvoiceSplitKey, string> = defineLabels({
  Customer: "One invoice",
  CustomerAndPONumber: "One per PO",
  CustomerAndShipmentBOL: "One per BOL",
  CustomerAndOrder: "One per order",
  CustomerAndOrigin: "One per pickup",
  CustomerAndDestination: "One per delivery",
  CustomerAndServiceType: "One per service type",
});

/** How the period splits, short enough to sit next to a number. */
export function splitLabel(splitBy: InvoiceSplitKey): string {
  return SPLIT_LABELS[splitBy] ?? SPLIT_LABELS.Customer;
}

function anchorWeekday(anchorDay: number): string {
  return weekdayName(
    Number.isInteger(anchorDay) && anchorDay >= 0 && anchorDay <= 6 ? anchorDay : 1,
  );
}

function cadenceClause(cycle: BillingCycle, anchorDay: number): string {
  switch (cycle) {
    case "Daily":
      return translate("Bills every day.");
    case "Weekly":
      return translate("Bills every week on {0}.", anchorWeekday(anchorDay));
    case "BiWeekly":
      return translate("Bills every other week on {0}.", anchorWeekday(anchorDay));
    case "SemiMonthly":
      return translate("Bills twice a month, on the 1st and the {0}.", formatOrdinal(anchorDay));
    case "Monthly":
      return translate("Bills every month on the {0}.", formatOrdinal(anchorDay));
    case "Quarterly":
      return translate("Bills every quarter on the {0}.", formatOrdinal(anchorDay));
    default:
      return translate("Bills as soon as a shipment is approved.");
  }
}

const SPLIT_SENTENCES: Record<InvoiceSplitKey, string> = defineLabels({
  Customer:
    "All of this customer's delivered shipments in the period are combined into a single invoice.",
  CustomerAndPONumber:
    "All of this customer's delivered shipments in the period are combined into one invoice per PO number.",
  CustomerAndShipmentBOL:
    "All of this customer's delivered shipments in the period are combined into one invoice per BOL.",
  CustomerAndOrder:
    "All of this customer's delivered shipments in the period are combined into one invoice per order (a shipment booked without an order is invoiced on its own).",
  CustomerAndOrigin:
    "All of this customer's delivered shipments in the period are combined into one invoice per pickup location.",
  CustomerAndDestination:
    "All of this customer's delivered shipments in the period are combined into one invoice per delivery location.",
  CustomerAndServiceType:
    "All of this customer's delivered shipments in the period are combined into one invoice per service type.",
});

const SECTION_SENTENCES: Record<InvoiceSectionKey, string> = defineLabels({
  Shipment: "Lines are grouped by shipment.",
  PONumber: "Lines are grouped by PO number.",
  Origin: "Lines are grouped by pickup location.",
  Destination: "Lines are grouped by delivery location.",
});

const DETAIL_SENTENCES: Record<InvoiceDetail, string> = defineLabels({
  Summary: "Each invoice shows one line per shipment.",
  Detailed: "Each invoice itemises every shipment's charges.",
});

/**
 * Renders a billing schedule as the sentence an operator would say out loud.
 *
 * The old configuration spread this across three unrelated form sections and
 * still could not express it, so the sentence is the point: if it cannot be
 * said plainly, the settings disagree with each other.
 */
export function describeBillingSchedule(schedule: BillingSchedule): string {
  if (schedule.invoiceDelivery === "PerShipment") {
    return translate(
      "Bills each shipment on its own invoice as soon as it is approved in the billing queue.",
    );
  }

  if (schedule.invoiceDelivery === "PerOrder") {
    return translate(
      "Bills each order on one invoice covering every billable leg, as soon as the order is approved.",
    );
  }

  const parts = [
    cadenceClause(schedule.billingCycle, schedule.billingCycleAnchorDay),
    SPLIT_SENTENCES[schedule.splitBy],
    SECTION_SENTENCES[schedule.sectionBy],
    DETAIL_SENTENCES[schedule.invoiceDetail],
  ];

  if (schedule.maxShipmentsPerInvoice > 0) {
    parts.push(
      translate(
        "{0, plural, one {Up to # shipment per invoice.} other {Up to # shipments per invoice.}}",
        schedule.maxShipmentsPerInvoice,
      ),
    );
  }

  return parts.join(" ");
}

/**
 * How many shipments on a statement are billing on their own invoice because the
 * customer is split by order and those shipments were not booked as orders.
 *
 * This measures the configuration's effect; it never overrides it. Choosing how a
 * customer is invoiced belongs to the people who run the account, and the system
 * bills exactly what the profile says. The count exists so the statement can
 * point out a result that may not be what they expected.
 *
 * Zero whenever members were not loaded — the list read omits them — so a caller
 * never shows a note built from a guess.
 */
export function standaloneShipmentCount(
  statement: Pick<OpenStatement, "splitBy" | "groups">,
): number {
  if (statement.splitBy !== "CustomerAndOrder") return 0;

  let count = 0;
  for (const group of statement.groups ?? []) {
    for (const shipment of group.shipments ?? []) {
      if (!shipment.orderId) count += 1;
    }
  }

  return count;
}

/**
 * "Mar 1 – Apr 1" for a half-open period.
 *
 * The end is exclusive on the wire, so it is shown as the boundary the freight
 * stops at rather than being decremented by a day — a biller reading "Mar 1 –
 * Mar 31" would reasonably expect a 31 March delivery to be on it, and it is
 * not.
 */
export function periodRange(periodStart: number, periodEnd: number): string {
  return `${formatUnixDateMedium(periodStart)} – ${formatUnixDateMedium(periodEnd)}`;
}

/**
 * How long until a statement bills, in the words a biller uses.
 *
 * A boundary already in the past is said out loud rather than clamped: it means
 * the scheduled sweep has not caught up, which is worth seeing.
 */
export function billsInLabel(periodEnd: number, nowSeconds: number): string {
  const seconds = periodEnd - nowSeconds;
  if (seconds <= 0) return translate("Due now");

  const days = Math.floor(seconds / 86_400);
  if (days >= 2)
    return translate("{0, plural, one {Bills in # day} other {Bills in # days}}", days);
  if (days === 1) return translate("Bills tomorrow");

  const hours = Math.floor(seconds / 3_600);
  if (hours >= 2) {
    return translate("{0, plural, one {Bills in # hour} other {Bills in # hours}}", hours);
  }
  return translate("Bills within the hour");
}
