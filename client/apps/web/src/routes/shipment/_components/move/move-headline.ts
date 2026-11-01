import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import type { MoveStatus, Stop, StopEta } from "@trenova/shared/types/shipment";

/** What a move's headline says first, and the tone it is drawn in. */
export type MoveHeadline = {
  title: string;
  tone: "default" | "success" | "muted";
  /** The stop the headline is about, whose projection the line beneath reads. */
  focus: Stop | null;
};

/** The stop the move is working towards: the first open one that has not been left. */
export function focusStop(stops: readonly Stop[]): Stop | null {
  return stops.find((stop) => stop.status !== "Canceled" && !stop.actualDeparture) ?? null;
}

/**
 * The move in one sentence, as a dispatcher would say it: where it picks up and when, where it
 * is arriving and when (the projection when there is one, else the appointment), where the truck
 * is now, where it delivered, or that it was canceled. `placeOf` names a stop by its city, or
 * whatever the stop knows of its place.
 */
export function moveHeadline({
  status,
  stops,
  etas,
  placeOf,
  t,
}: {
  status: MoveStatus;
  stops: readonly Stop[];
  etas: ReadonlyMap<string, StopEta>;
  placeOf: (stop: Stop) => string;
  t: TranslateFn;
}): MoveHeadline {
  if (status === "Canceled") {
    return { title: t("Canceled"), tone: "muted", focus: null };
  }
  if (stops.every((stop) => !stop.locationId && !stop.addressLine)) {
    return { title: t("Add the stops to plan this move"), tone: "muted", focus: null };
  }

  const focus = focusStop(stops);
  if (!focus || status === "Completed") {
    const last = [...stops].reverse().find((stop) => stop.status !== "Canceled") ?? stops.at(-1)!;
    return { title: t("Delivered to {0}", placeOf(last)), tone: "success", focus: null };
  }

  const place = placeOf(focus);
  if (focus.actualArrival) {
    return {
      title: t("At {0} since {1}", place, formatUnixDateTimeShort(focus.actualArrival)),
      tone: "default",
      focus,
    };
  }

  const eta = focus.id ? etas.get(focus.id) : undefined;
  const when = eta?.estimatedArrival || focus.scheduledWindowStart || null;
  const started = stops.some((stop) => !!stop.actualDeparture);
  if (started) {
    return {
      title: when
        ? t("Arriving at {0}, {1}", place, formatUnixDateTimeShort(when))
        : t("On the way to {0}", place),
      tone: "default",
      focus,
    };
  }
  const scheduled = focus.scheduledWindowStart
    ? formatUnixDateTimeShort(focus.scheduledWindowStart)
    : null;
  const pickup = focus.type === "Pickup" || focus.type === "SplitPickup";
  const title = pickup
    ? scheduled
      ? t("Picks up at {0}, {1}", place, scheduled)
      : t("Picks up at {0}", place)
    : scheduled
      ? t("Delivers to {0}, {1}", place, scheduled)
      : t("Delivers to {0}", place);
  return { title, tone: "default", focus };
}
