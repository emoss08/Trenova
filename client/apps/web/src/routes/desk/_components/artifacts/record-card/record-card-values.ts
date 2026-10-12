import { formatUnixDateMedium } from "@trenova/shared/lib/date";
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

/** A unix second as a card writes a month: the month and the year. */
export function cardMonth(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    year: "numeric",
  });
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
