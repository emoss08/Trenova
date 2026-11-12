import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";

/** How long between two instants, in the short words the Desk's clocks use ("1d 19h"). */
export function spanBetween(from: number, to: number): string {
  return formatMinutesSpan(Math.max(1, Math.round(Math.abs(to - from) / 60)));
}

/** An instant against now, in words: "1d 19h ago", "in 6h". A time of 0 is not known. */
export function agoOrIn(t: TranslateFn, at: number, now: number): string {
  if (at <= 0) {
    return "";
  }
  if (Math.abs(at - now) < 60) {
    return t("Just now");
  }
  const span = spanBetween(at, now);

  return at < now ? t("{0} ago", span) : t("in {0}", span);
}

/**
 * Where an instant sits on a track that runs from `start` to `end`, as a
 * share of its width, kept a little inside both ends so a mark at an edge is
 * still whole.
 */
export function trackShare(at: number, start: number, end: number): number {
  if (end <= start) {
    return 50;
  }

  return Math.min(97, Math.max(3, ((at - start) / (end - start)) * 100));
}
