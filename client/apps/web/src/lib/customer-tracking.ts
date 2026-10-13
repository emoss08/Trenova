import type {
  CustomerStatusNotificationRow,
  CustomerTrackingField,
  CustomerTrackingLinkRow,
} from "@/lib/graphql/customer-tracking";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";

export type LinkState = "live" | "expired" | "revoked";

/** Whether a link still opens, judged at the given instant. */
export function linkState(link: CustomerTrackingLinkRow, now: number): LinkState {
  if (link.revokedAt != null) {
    return "revoked";
  }
  return link.expiresAt > now ? "live" : "expired";
}

export function linkStateAttrs(t: TranslateFn): Record<LinkState, BadgeAttrProps> {
  return {
    live: { phase: "active", text: t("Live") },
    expired: { phase: "closed", text: t("Expired") },
    revoked: { phase: "closed", text: t("Revoked") },
  };
}

export function liveLinkCount(links: readonly CustomerTrackingLinkRow[], now: number): number {
  return links.filter((link) => linkState(link, now) === "live").length;
}

export function notificationStatusAttrs(
  t: TranslateFn,
): Record<CustomerStatusNotificationRow["status"], BadgeAttrProps> {
  return {
    Pending: {
      phase: "queued",
      text: t("Pending"),
      description: t("Owed and not yet sent; retried after a failure"),
    },
    Sent: { phase: "complete", text: t("Sent") },
    Skipped: { phase: "closed", text: t("Skipped") },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("Gave up after repeated failures"),
    },
  };
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
