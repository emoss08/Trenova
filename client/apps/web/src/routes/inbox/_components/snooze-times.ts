import type {
  InboundClassification,
  InboundMessageRow,
  InboundSnoozeEvent,
} from "@/lib/graphql/inbox";
import { fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";

/**
 * When a snoozed message comes back.
 *
 * The three times are wall-clock times in the person's own time zone — five
 * in the afternoon, eight tomorrow morning, eight on Monday — so a dispatcher
 * in Denver who snoozes until tomorrow gets the message back at eight in
 * Denver, whatever zone the browser or the server is in. A clock change
 * between now and then is the zone library's to resolve, not an hour added.
 */
export type SnoozePresetKey = "laterToday" | "tomorrow" | "monday";

export type SnoozePreset = { key: SnoozePresetKey; until: number };

const LATER_TODAY_HOUR = 17;
const MORNING_HOUR = 8;
const MONDAY = 1;
const DAYS_PER_WEEK = 7;

function wallClockAt(base: Date, dayOffset: number, hour: number, timeZone: string): number {
  const wall = new Date(
    base.getFullYear(),
    base.getMonth(),
    base.getDate() + dayOffset,
    hour,
    0,
    0,
  );

  return fromUserWallClock(wall, timeZone) ?? 0;
}

/**
 * The preset times, in order. "Later today" is left out once five o'clock has
 * passed, and Monday is left out when tomorrow already is Monday morning.
 */
export function snoozePresets(now: number, timeZone: string): SnoozePreset[] {
  const today = toUserWallClock(now, timeZone);
  if (today === undefined) {
    return [];
  }

  const presets: SnoozePreset[] = [];
  const laterToday = wallClockAt(today, 0, LATER_TODAY_HOUR, timeZone);
  if (laterToday > now) {
    presets.push({ key: "laterToday", until: laterToday });
  }

  const tomorrow = wallClockAt(today, 1, MORNING_HOUR, timeZone);
  presets.push({ key: "tomorrow", until: tomorrow });

  const daysToMonday = (MONDAY - today.getDay() + DAYS_PER_WEEK) % DAYS_PER_WEEK || DAYS_PER_WEEK;
  const monday = wallClockAt(today, daysToMonday, MORNING_HOUR, timeZone);
  if (monday !== tomorrow) {
    presets.push({ key: "monday", until: monday });
  }

  return presets;
}

/** Tomorrow at eight: what S and the bulk bar's snooze use. */
export function tomorrowMorning(now: number, timeZone: string): number {
  const preset = snoozePresets(now, timeZone).find((row) => row.key === "tomorrow");

  return preset?.until ?? now + 86_400;
}

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
