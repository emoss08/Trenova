import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  formatUnixMonthDay,
  formatUnixTime,
} from "@trenova/shared/lib/date";
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

/** Where the load is along its stops, as the route ribbon draws it. */
export type Journey = {
  /** 0 at the first stop, 1 at the last, stops spaced evenly. */
  progress: number;
  /** The stop the load is at, or heading to next. */
  activeIndex: number;
  /** Whether the load is standing at activeIndex rather than between stops. */
  atStop: boolean;
  started: boolean;
  finished: boolean;
};

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

/**
 * The load's place along its route at `now`. Between two stops it moves with
 * time from the departure toward the next stop's expected arrival, so a load
 * three quarters of the way through its drive sits three quarters along the
 * segment; without an expectation it sits halfway.
 */
export function journeyOf(view: PublicShipmentView, now: number): Journey {
  const stops = view.stops;
  const last = stops.length - 1;
  if (last < 1) {
    return { progress: 0, activeIndex: 0, atStop: true, started: false, finished: false };
  }
  if (view.status === "Delivered" || stops[last].arrivedAt != null) {
    return { progress: 1, activeIndex: last, atStop: true, started: true, finished: true };
  }

  let reached = -1;
  for (let index = 0; index <= last; index += 1) {
    if (stops[index].arrivedAt != null || stops[index].departedAt != null) {
      reached = index;
    }
  }
  if (reached === -1) {
    return { progress: 0, activeIndex: 0, atStop: false, started: false, finished: false };
  }

  const here = stops[reached];
  if (here.departedAt == null) {
    return {
      progress: reached / last,
      activeIndex: reached,
      atStop: true,
      started: true,
      finished: false,
    };
  }

  const next = stops[reached + 1];
  const target = next.estimatedArrival ?? next.windowStart ?? null;
  const span = target != null ? target - here.departedAt : 0;
  const fraction = span > 0 ? clamp((now - here.departedAt) / span, 0.06, 0.94) : 0.5;
  return {
    progress: (reached + fraction) / last,
    activeIndex: reached + 1,
    atStop: false,
    started: true,
    finished: false,
  };
}

/**
 * The stop the customer is waiting on: the first one the load has not reached,
 * else the last. A stop it has left counts as reached even when only the
 * departure was reported.
 */
export function nextStopOf(view: PublicShipmentView): PublicShipmentStop | undefined {
  return (
    view.stops.find((stop) => stop.arrivedAt == null && stop.departedAt == null) ??
    view.stops.at(-1)
  );
}

/** The page's headline, one sentence about the whole shipment. */
export function trackingHeadline(t: TranslateFn, view: PublicShipmentView): string {
  const first = view.stops[0];
  const final = view.stops.at(-1);
  const destination = final?.city ?? "";
  switch (view.status) {
    case "Delivered":
      return destination ? t("Delivered in {0}.", destination) : t("Delivered.");
    case "Canceled":
      return t("This shipment was canceled.");
    case "Delayed":
      return destination ? t("Running late to {0}.", destination) : t("Running late.");
    case "Planned":
      return first?.city ? t("Getting ready to leave {0}.", first.city) : t("Getting ready.");
    default:
      return destination ? t("On its way to {0}.", destination) : t("On its way.");
  }
}

/**
 * How an expected arrival sits against the stop's appointment, in minutes:
 * negative is before the window opens, positive after it closes, zero inside.
 */
export function minutesOutsideWindow(stop: PublicShipmentStop, at: number): number | null {
  if (stop.windowStart == null) {
    return null;
  }
  const end = stop.windowEnd ?? stop.windowStart;
  if (at < stop.windowStart) {
    return Math.round((at - stop.windowStart) / 60);
  }
  if (at > end) {
    return Math.round((at - end) / 60);
  }
  return 0;
}

function localDayIndex(unix: number): number {
  const date = new Date(unix * 1000);
  return Math.floor(
    Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()) / 86_400_000,
  );
}

/**
 * A moment as a person says it: "Today, 4:40 PM", "Tomorrow, 9:00 AM",
 * "Oct 16, 2:15 PM". Inline it reads inside a sentence: "today at 4:40 PM".
 */
export function dayAndTime(
  t: TranslateFn,
  unix: number,
  now: number,
  { inline = false }: { inline?: boolean } = {},
): string {
  const time = formatUnixTime(unix);
  const days = localDayIndex(unix) - localDayIndex(now);
  if (days === 0) {
    return inline ? t("today at {0}", time) : t("Today, {0}", time);
  }
  if (days === 1) {
    return inline ? t("tomorrow at {0}", time) : t("Tomorrow, {0}", time);
  }
  if (days === -1) {
    return inline ? t("yesterday at {0}", time) : t("Yesterday, {0}", time);
  }
  return inline
    ? t("{0} at {1}", formatUnixMonthDay(unix), time)
    : t("{0}, {1}", formatUnixMonthDay(unix), time);
}

/** An appointment window as one short range: "4:58 – 5:58 PM", or its start alone. */
export function windowRange(t: TranslateFn, stop: PublicShipmentStop): string | null {
  if (stop.windowStart == null) {
    return null;
  }
  if (stop.windowEnd == null || stop.windowEnd === stop.windowStart) {
    return formatUnixTime(stop.windowStart);
  }
  return t("{0} – {1}", formatUnixTime(stop.windowStart), formatUnixTime(stop.windowEnd));
}
