import { formatDurationFromSeconds, formatUnixDateMedium } from "@trenova/shared/lib/date";
/** An amount as a record card writes it, or a dash for one that is not there. */
export function money(amount: string | number | null | undefined, currency = "USD"): string {
  const figure = Number(amount);
  if (amount === null || amount === undefined || amount === "" || !Number.isFinite(figure)) {
    return "—";
  }
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

/** An amount the server counts in minor units, such as cents, as a record card writes it. */
export function minorMoney(minor: number | null | undefined, currency = "USD"): string {
  return minor === null || minor === undefined ? money(minor) : money(minor / 100, currency);
}

/** A unix second as a card's timeline says it: the day and the time. */
export function cardMoment(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

/** A unix second as a card says a time on a day already given: the time alone. */
export function cardTime(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}

/**
 * The second of two moments, as a card writes it beside the first: the time
 * alone when both fall on the same day, else the day and the time.
 */
export function cardMomentAfter(first: number, second: number): string {
  return cardDay(first) === cardDay(second) ? cardTime(second) : cardMoment(second);
}

/** A unix second as a card's header says it: the day alone. */
export function cardDay(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

/** A unix second as a card writes a date that may be in another year: the day and the year. */
export function cardDate(seconds: number | null | undefined): string {
  return formatUnixDateMedium(seconds, { fallback: "" });
}

/** A unix second as a card names a day ahead or just gone: "Fri, Oct 17". */
export function cardWeekday(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}

/** A unix second as a card writes a month: the month and the year. */
export function cardMonth(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    year: "numeric",
  });
}

const DAY_SECONDS = 86_400;

/** When something falls due, against now: past, close (inside the notice), or comfortably off. */
export type DueReading = { state: "ok" | "warn" | "bad"; days: number };

/** Whole days until `at` from `now`, and whether that is past, inside the notice or beyond it. */
export function dueReading(at: number, now: number, noticeDays: number): DueReading {
  const days = Math.ceil((at - now) / DAY_SECONDS);
  if (at <= now) return { state: "bad", days };
  if (days <= noticeDays) return { state: "warn", days };
  return { state: "ok", days };
}

/** Where a record sits in the list it was opened from, and the ones either side. */
export type Neighbors = {
  prevId?: string | null;
  nextId?: string | null;
  /** One-based; zero when the record is not in the list. */
  position: number;
  total: number;
};

/** A record's neighbours in a table's own order, for stepping through it from the card. */
export function neighborsIn(order: readonly string[], id: string): Neighbors {
  const at = order.indexOf(id);
  if (at === -1) return { prevId: null, nextId: null, position: 0, total: order.length };
  return {
    prevId: order[at - 1] ?? null,
    nextId: order[at + 1] ?? null,
    position: at + 1,
    total: order.length,
  };
}

/** A person as a record names them: a first and a last name, either possibly blank. */
export type Person = { firstName: string; lastName: string } | null | undefined;

/** A person's name as a card writes it; empty for none. */
export function personName(person: Person): string {
  if (!person) return "";
  return [person.firstName, person.lastName]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(" ");
}

/** A span in minutes, as a card reads a wait or a delay: "1h 36m", or "0m" for none. */
export function waitText(minutes: number): string {
  return formatDurationFromSeconds(Math.round(Math.max(0, minutes)) * 60);
}

/** A place's city, state and postal code on one line: "Dallas, TX 75201". */
export function cityLine(place: {
  city: string;
  postalCode?: string | null;
  state?: { abbreviation: string } | null;
}): string {
  const city = place.city.trim();
  const region = [place.state?.abbreviation ?? "", place.postalCode?.trim() ?? ""]
    .filter(Boolean)
    .join(" ");
  return [city, region].filter(Boolean).join(", ");
}
