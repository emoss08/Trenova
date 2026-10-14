import type {
  CustomerStatusNotificationRow,
  CustomerTrackingField,
  CustomerTrackingLinkRow,
} from "@/lib/graphql/customer-tracking";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type LinkState = "live" | "expired" | "revoked";

/** Whether a link still opens, judged at the given instant. */
export function linkState(link: CustomerTrackingLinkRow, now: number): LinkState {
  if (link.revokedAt != null) {
    return "revoked";
  }
  return link.expiresAt > now ? "live" : "expired";
}

export function liveLinkCount(links: readonly CustomerTrackingLinkRow[], now: number): number {
  return links.filter((link) => linkState(link, now) === "live").length;
}

export function statusEventLabel(t: TranslateFn, event: CustomerStatusNotificationRow["event"]) {
  switch (event) {
    case "PickedUp":
      return t("Picked up");
    case "Delayed":
      return t("Delayed");
    case "Delivered":
      return t("Delivered");
    default:
      return t("Proof of delivery ready");
  }
}

export function skipReasonLabel(
  t: TranslateFn,
  reason: CustomerStatusNotificationRow["skipReason"],
): string | null {
  switch (reason) {
    case "NoRecipients":
      return t("Nobody to send it to");
    case "RuleOff":
      return t("The rule was turned off");
    case "ShipmentGone":
      return t("The shipment was canceled");
    default:
      return null;
  }
}

export const TRACKING_FIELDS: readonly CustomerTrackingField[] = [
  "Status",
  "LocationNames",
  "Windows",
  "Actuals",
  "Eta",
  "References",
  "Carrier",
  "Equipment",
  "Freight",
  "LastLocation",
  "Holds",
  "Notes",
  "ProofOfDelivery",
];

export function trackingFieldLabel(t: TranslateFn, field: CustomerTrackingField): string {
  switch (field) {
    case "Status":
      return t("Status");
    case "LocationNames":
      return t("Location names and addresses");
    case "Windows":
      return t("Appointment windows");
    case "Actuals":
      return t("Arrival and departure times");
    case "Eta":
      return t("Projected arrivals");
    case "References":
      return t("BOL and customer reference");
    case "Carrier":
      return t("Carrier");
    case "Equipment":
      return t("Equipment");
    case "Freight":
      return t("Pieces and weight");
    case "LastLocation":
      return t("Last reported location");
    case "Holds":
      return t("Holds marked visible to the customer");
    case "Notes":
      return t("Comments marked for the customer");
    default:
      return t("Proof of delivery");
  }
}

export const STATUS_EVENTS: readonly CustomerStatusNotificationRow["event"][] = [
  "PickedUp",
  "Delayed",
  "Delivered",
  "PodReady",
];

/** Where one status email stands on this shipment. */
export type StatusEmailStep = {
  event: CustomerStatusNotificationRow["event"];
  /** Whether the customer's rules send this email at all. */
  enabled: boolean;
  /** The latest email owed for the event, if one has been. */
  latest: CustomerStatusNotificationRow | null;
  /** How many have been owed, e.g. a delay email per late stop. */
  count: number;
};

/**
 * The four status emails in the order a shipment earns them, each with
 * whether the rules send it and the latest one owed. A rule turned off after
 * an email went out still shows that email.
 */
export function statusEmailSteps(
  rules: Readonly<Record<CustomerStatusNotificationRow["event"], boolean>>,
  rows: readonly CustomerStatusNotificationRow[],
): StatusEmailStep[] {
  return STATUS_EVENTS.map((event) => {
    const owed = rows.filter((row) => row.event === event);
    const latest = owed.reduce<CustomerStatusNotificationRow | null>(
      (best, row) => (best == null || row.occurredAt > best.occurredAt ? row : best),
      null,
    );
    return { event, enabled: rules[event], latest, count: owed.length };
  });
}
