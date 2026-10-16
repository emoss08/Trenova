import type {
  InboundClassification,
  InboundMessageRow,
  InboundSnoozeEvent,
} from "@/lib/graphql/inbox";

/** A snooze until something happens, as a menu offers it for one message. */
export type EventSnoozeOption =
  | { event: Extract<InboundSnoozeEvent, "ShipmentDelivered">; proNumber: string }
  | { event: Extract<InboundSnoozeEvent, "DocumentAttached"> }
  | { event: Extract<InboundSnoozeEvent, "InvoicePaid"> };

/** Kinds that arrive after a load has delivered; waiting on its delivery means nothing. */
const AFTER_DELIVERY: ReadonlySet<InboundClassification> = new Set([
  "ProofOfDelivery",
  "Invoice",
  "DetentionDispute",
]);

/** Kinds whose next move is money changing hands. */
const WAITS_ON_PAYMENT: ReadonlySet<InboundClassification> = new Set([
  "ProofOfDelivery",
  "Invoice",
  "DetentionDispute",
]);

/**
 * The events a message can wait on. Every one is watched on the message's
 * shipment, so a message with no shipment linked can only wait on a time.
 */
export function eventSnoozeOptions(
  message: Pick<InboundMessageRow, "matchedShipment" | "classification">,
): EventSnoozeOption[] {
  const shipment = message.matchedShipment;
  if (shipment === null) {
    return [];
  }

  const options: EventSnoozeOption[] = [];
  const kind = message.classification ?? null;
  if (kind === null || !AFTER_DELIVERY.has(kind)) {
    options.push({ event: "ShipmentDelivered", proNumber: shipment.proNumber });
  }
  options.push({ event: "DocumentAttached" });
  if (kind !== null && WAITS_ON_PAYMENT.has(kind)) {
    options.push({ event: "InvoicePaid" });
  }

  return options;
}
