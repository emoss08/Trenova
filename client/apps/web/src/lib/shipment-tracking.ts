import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type {
  CustomerTrackingFieldValue,
  PublicShipmentStatus,
  PublicShipmentStop,
  PublicShipmentView,
} from "@trenova/shared/types/shipment-tracking";

/** Where the load is, as the customer reads it. */
export function publicStatusAttrs(t: TranslateFn): Record<PublicShipmentStatus, BadgeAttrProps> {
  return {
    Planned: { phase: "queued", text: t("Planned"), description: t("Not picked up yet") },
    InTransit: { phase: "active", text: t("In transit"), description: t("On its way") },
    Delayed: {
      phase: "attention",
      text: t("Delayed"),
      description: t("Expected to reach a stop later than planned"),
    },
    Delivered: { phase: "complete", text: t("Delivered"), description: t("Delivered") },
    Canceled: { phase: "closed", text: t("Canceled"), description: t("The shipment was canceled") },
  };
}

export function shows(view: PublicShipmentView, field: CustomerTrackingFieldValue): boolean {
  return view.fields.includes(field);
}

export function stopPlace(stop: PublicShipmentStop): string {
  const town = [stop.city, stop.stateCode].filter(Boolean).join(", ");
  return [stop.locationName, town].filter(Boolean).join(" · ");
}

/**
 * The one line that says where a stop stands: left, reached, expected, or due.
 * Each part appears only when the view carries it, so a customer shown only the
 * route sees no times at all.
 */
export function stopStanding(t: TranslateFn, stop: PublicShipmentStop): string | null {
  if (stop.departedAt != null) {
    return t("Left {0}", formatUnixDateTime(stop.departedAt));
  }
  if (stop.arrivedAt != null) {
    return t("Arrived {0}", formatUnixDateTime(stop.arrivedAt));
  }
  if (stop.estimatedArrival != null) {
    return stop.running === "Late" || stop.running === "AtRisk"
      ? t("Now expected {0}", formatUnixDateTime(stop.estimatedArrival))
      : t("Expected {0}", formatUnixDateTime(stop.estimatedArrival));
  }
  if (stop.windowStart != null) {
    return t("Due {0}", formatUnixDateTime(stop.windowStart));
  }
  return null;
}

export function stopWindow(t: TranslateFn, stop: PublicShipmentStop): string | null {
  if (stop.windowStart == null) {
    return null;
  }
  if (stop.windowEnd == null) {
    return t("Appointment {0}", formatUnixDateTime(stop.windowStart));
  }
  return t(
    "Appointment {0} to {1}",
    formatUnixDateTime(stop.windowStart),
    formatUnixDateTime(stop.windowEnd),
  );
}

export function holdLabel(t: TranslateFn, hold: { type: string; label?: string }): string {
  if (hold.label) {
    return hold.label;
  }
  switch (hold.type) {
    case "CustomerHold":
      return t("Waiting on you");
    case "ComplianceHold":
      return t("Compliance check");
    case "FinanceHold":
      return t("Billing review");
    default:
      return t("Operational hold");
  }
}
