import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type TemperatureRange = { min: number | null; max: number | null };

type TemperatureBounds = {
  temperatureMin?: number | null;
  temperatureMax?: number | null;
};

type TemperatureStop = TemperatureBounds & { status?: string | null };

type TemperatureMove = { status?: string | null; stops?: readonly TemperatureStop[] | null };

/** "34–38°F", "≥ 34°F", "≤ 38°F"; one figure when both ends meet; "" with neither. */
export function temperatureLabel(range: TemperatureRange, t: TranslateFn): string {
  const { min, max } = range;
  if (min !== null && max !== null) {
    return min === max ? t("{0}°F", min) : t("{0}–{1}°F", min, max);
  }
  if (min !== null) return t("≥ {0}°F", min);
  if (max !== null) return t("≤ {0}°F", max);
  return "";
}

/** The range a stop, commodity line or load holds, with an unset bound as null. */
export function temperatureOf(bounds: TemperatureBounds | null | undefined): TemperatureRange {
  return { min: bounds?.temperatureMin ?? null, max: bounds?.temperatureMax ?? null };
}

/** The ranges the shipment's live stops and its commodity lines set, in that order. */
export function lineTemperatures(
  moves: readonly TemperatureMove[] | null | undefined,
  commodities: readonly TemperatureBounds[] | null | undefined,
): TemperatureRange[] {
  const stops = (moves ?? [])
    .filter((move) => move.status !== "Canceled")
    .flatMap((move) => (move.stops ?? []).filter((stop) => stop.status !== "Canceled"));
  return [...stops, ...(commodities ?? [])]
    .map(temperatureOf)
    .filter((range) => range.min !== null || range.max !== null);
}

/**
 * The load's range, as the server derives it (`temperaturerange.Load`): the range every line
 * shares, or, when they share none (a frozen and a chilled drop), the span from the coldest to the
 * warmest bound any line gives. Null when no line sets one.
 */
export function loadTemperature(ranges: readonly TemperatureRange[]): TemperatureRange | null {
  if (ranges.length === 0) return null;
  const present = (values: (number | null)[]) => values.filter((value) => value !== null);
  const mins = present(ranges.map((range) => range.min));
  const maxes = present(ranges.map((range) => range.max));
  const overlap: TemperatureRange = {
    min: mins.length > 0 ? Math.max(...mins) : null,
    max: maxes.length > 0 ? Math.min(...maxes) : null,
  };
  if (overlap.min === null || overlap.max === null || overlap.min <= overlap.max) {
    return overlap;
  }
  const bounds = [...mins, ...maxes];
  return { min: Math.min(...bounds), max: Math.max(...bounds) };
}
