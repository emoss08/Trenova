import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type {
  BillingCycle,
  InvoiceDelivery,
  InvoiceDetail,
  InvoiceSectionKey,
  InvoiceSplitKey,
} from "@trenova/shared/types/customer";

export type BillingSchedule = {
  invoiceDelivery: InvoiceDelivery;
  billingCycle: BillingCycle;
  billingCycleAnchorDay: number;
  splitBy: InvoiceSplitKey;
  sectionBy: InvoiceSectionKey;
  invoiceDetail: InvoiceDetail;
  maxShipmentsPerInvoice: number;
};

const CADENCE_LABELS: Record<BillingCycle, string> = {
  Immediate: "Per shipment",
  Daily: "Daily",
  Weekly: "Weekly",
  BiWeekly: "Bi-weekly",
  SemiMonthly: "Semi-monthly",
  Monthly: "Monthly",
  Quarterly: "Quarterly",
};

/** The cadence as a chip label — two words at most, for a card or a badge. */
export function cadenceLabel(cycle: BillingCycle): string {
  return CADENCE_LABELS[cycle] ?? "Per shipment";
}

const SPLIT_LABELS: Record<InvoiceSplitKey, string> = {
  Customer: "One invoice",
  CustomerAndPONumber: "One per PO",
  CustomerAndShipmentBOL: "One per BOL",
  CustomerAndOrder: "One per order",
  CustomerAndOrigin: "One per pickup",
  CustomerAndDestination: "One per delivery",
  CustomerAndServiceType: "One per service type",
};

/** How the period splits, short enough to sit next to a number. */
export function splitLabel(splitBy: InvoiceSplitKey): string {
  return SPLIT_LABELS[splitBy] ?? SPLIT_LABELS.Customer;
}

const ORDINAL_SUFFIXES = ["th", "st", "nd", "rd"];

function ordinal(day: number): string {
  const remainder = day % 100;
  const suffix =
    ORDINAL_SUFFIXES[(remainder - 20) % 10] ?? ORDINAL_SUFFIXES[remainder] ?? ORDINAL_SUFFIXES[0];
  return `${day}${suffix}`;
}

const WEEKDAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function cadenceClause(cycle: BillingCycle, anchorDay: number): string {
  switch (cycle) {
    case "Daily":
      return "Bills every day.";
    case "Weekly":
      return `Bills every week on ${WEEKDAYS[anchorDay] ?? "Monday"}.`;
    case "BiWeekly":
      return `Bills every other week on ${WEEKDAYS[anchorDay] ?? "Monday"}.`;
    case "SemiMonthly":
      return `Bills twice a month, on the 1st and the ${ordinal(anchorDay)}.`;
    case "Monthly":
      return `Bills every month on the ${ordinal(anchorDay)}.`;
    case "Quarterly":
      return `Bills every quarter on the ${ordinal(anchorDay)}.`;
    default:
      return "Bills as soon as a shipment is approved.";
  }
}

const SPLIT_CLAUSES: Record<InvoiceSplitKey, string> = {
  Customer: "are combined into a single invoice",
  CustomerAndPONumber: "are combined into one invoice per PO number",
  CustomerAndShipmentBOL: "are combined into one invoice per BOL",
  CustomerAndOrder: "are combined into one invoice per order",
  CustomerAndOrigin: "are combined into one invoice per pickup location",
  CustomerAndDestination: "are combined into one invoice per delivery location",
  CustomerAndServiceType: "are combined into one invoice per service type",
};

const SECTION_CLAUSES: Record<InvoiceSectionKey, string> = {
  Shipment: "grouped by shipment",
  PONumber: "grouped by PO number",
  Origin: "grouped by pickup location",
  Destination: "grouped by delivery location",
};

/**
 * Renders a billing schedule as the sentence an operator would say out loud.
 *
 * The old configuration spread this across three unrelated form sections and
 * still could not express it, so the sentence is the point: if it cannot be
 * said plainly, the settings disagree with each other.
 */
export function describeBillingSchedule(schedule: BillingSchedule): string {
  if (schedule.invoiceDelivery === "PerShipment") {
    return "Bills each shipment on its own invoice as soon as it is approved in the billing queue.";
  }

  if (schedule.invoiceDelivery === "PerOrder") {
    return "Bills each order on one invoice covering every billable leg, as soon as the order is approved.";
  }

  const parts = [cadenceClause(schedule.billingCycle, schedule.billingCycleAnchorDay)];

  const detail =
    schedule.invoiceDetail === "Summary"
      ? "one line per shipment"
      : "each shipment's charges itemised";

  parts.push(
    `All of this customer's delivered shipments in the period ` +
      `${SPLIT_CLAUSES[schedule.splitBy]}, with lines ` +
      `${SECTION_CLAUSES[schedule.sectionBy]} and ${detail}.`,
  );

  if (schedule.maxShipmentsPerInvoice > 0) {
    parts.push(`Up to ${schedule.maxShipmentsPerInvoice} shipments per invoice.`);
  }

  return parts.join(" ");
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
  if (seconds <= 0) return "Due now";

  const days = Math.floor(seconds / 86_400);
  if (days >= 2) return `Bills in ${days} days`;
  if (days === 1) return "Bills tomorrow";

  const hours = Math.floor(seconds / 3_600);
  if (hours >= 2) return `Bills in ${hours} hours`;
  return "Bills within the hour";
}
