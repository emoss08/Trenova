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
