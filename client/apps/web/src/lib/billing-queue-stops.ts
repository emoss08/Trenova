import type { TranslateFn } from "@trenova/shared/i18n/use-t";

type MoveStops = {
  sequence: number;
  loaded?: boolean | null;
  distance?: number | null;
  stops: ReadonlyArray<{
    id?: string | null;
    type: string;
    sequence: number;
    scheduledWindowStart: number;
    scheduledWindowEnd?: number | null;
    actualArrival?: number | null;
    actualDeparture?: number | null;
    addressLine: string;
    location?: {
      name: string;
      city: string;
      state?: { abbreviation: string } | null;
    } | null;
  }>;
};

/** One stop as the Shipment tab lists it, in the order the load ran. */
export type StopRow = {
  key: string;
  type: string;
  name: string;
  place: string;
  scheduledStart: number;
  scheduledEnd: number | null;
  arrival: number | null;
  departure: number | null;
  /** Seconds between arriving and leaving; null until both are known. */
  onSite: number | null;
};

/** Every stop of every move, move by move and stop by stop. */
export function stopRows(moves: ReadonlyArray<MoveStops>): StopRow[] {
  return [...moves]
    .sort((a, b) => a.sequence - b.sequence)
    .flatMap((move) =>
      [...move.stops]
        .sort((a, b) => a.sequence - b.sequence)
        .map((stop, index) => {
          const location = stop.location;
          const place = location
            ? [location.city, location.state?.abbreviation].filter(Boolean).join(", ")
            : "";
          const arrival = stop.actualArrival ?? null;
          const departure = stop.actualDeparture ?? null;
          return {
            key: stop.id ?? `${move.sequence}-${index}`,
            type: stop.type,
            name: location?.name || stop.addressLine,
            place: place || stop.addressLine,
            scheduledStart: stop.scheduledWindowStart,
            scheduledEnd: stop.scheduledWindowEnd ?? null,
            arrival,
            departure,
            onSite:
              arrival != null && departure != null && departure >= arrival
                ? departure - arrival
                : null,
          };
        }),
    );
}

/** The miles the load ran with freight and without, across its moves. */
export function shipmentMiles(moves: ReadonlyArray<MoveStops>): { loaded: number; empty: number } {
  let loaded = 0;
  let empty = 0;
  for (const move of moves) {
    const distance = move.distance ?? 0;
    if (move.loaded === false) {
      empty += distance;
    } else {
      loaded += distance;
    }
  }

  return { loaded: Math.round(loaded), empty: Math.round(empty) };
}

/** Time on site, "4h 10m" or "53m". */
export function dwellText(seconds: number, t: TranslateFn): string {
  const minutes = Math.round(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;

  return hours > 0 ? t("{0}h {1}m", hours, rest) : t("{0}m", rest);
}
