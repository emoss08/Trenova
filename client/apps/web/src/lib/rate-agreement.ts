import { choiceLabel, rateAgreementStatusChoices, rateAgreementTypeChoices } from "@/lib/choices";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

const DAY_SECONDS = 86_400;

/** What reading an agreement's term needs, whichever query it came from. */
export type AgreementTerm = {
  status: string;
  effectiveFrom: number;
  effectiveTo?: number | null;
  autoRenew: boolean;
  renewalNoticeDays: number;
};

/** Whether an agreement's term is fine, needs a look, or has run out. */
export type AgreementTermState = "ok" | "warn" | "bad";

/** How long an agreement runs, and whether its end needs someone's attention. */
export type AgreementWindow = { text: string; state: AgreementTermState | null };

function day(seconds: number): string {
  return formatUnixDateMedium(seconds, { fallback: "" });
}

/** Whole days from `now` until `end`, both unix seconds; negative once `end` has passed. */
export function daysUntil(end: number, now: number): number {
  return Math.ceil((end - now) / DAY_SECONDS);
}

/**
 * An agreement's window as of `now` (unix seconds). An active agreement with no
 * end runs on; one inside its renewal notice is flagged unless it renews by
 * itself; one past its end, or not active at all, says what it is.
 */
export function agreementWindow(
  agreement: AgreementTerm,
  now: number,
  t: TranslateFn,
): AgreementWindow {
  if (agreement.status !== "Active") {
    return { text: choiceLabel(rateAgreementStatusChoices, agreement.status), state: null };
  }
  if (agreement.effectiveFrom > now) {
    return { text: t("Starts {0}", day(agreement.effectiveFrom)), state: null };
  }
  const end = agreement.effectiveTo;
  if (end === null || end === undefined) {
    return { text: t("No end date"), state: "ok" };
  }
  if (end < now) {
    return { text: t("Ended {0}", day(end)), state: "bad" };
  }
  const daysLeft = daysUntil(end, now);
  if (daysLeft <= agreement.renewalNoticeDays) {
    return agreement.autoRenew
      ? { text: t("Renews {0}", day(end)), state: "ok" }
      : {
          text: t("Ends in {0, plural, one {# day} other {# days}}", daysLeft),
          state: "warn",
        };
  }
  return { text: t("Through {0}", day(end)), state: "ok" };
}

/** An agreement's kind, as the rate agreement pages name it. */
export function agreementTypeLabel(type: string): string {
  return choiceLabel(rateAgreementTypeChoices, type);
}

/** Agreements that price freight now first, then the rest, each group newest first. */
export function orderedAgreements<T extends Pick<AgreementTerm, "status" | "effectiveFrom">>(
  agreements: readonly T[],
): T[] {
  return [...agreements].sort(
    (a, b) =>
      Number(b.status === "Active") - Number(a.status === "Active") ||
      b.effectiveFrom - a.effectiveFrom,
  );
}

/** What paging an agreement's lanes reads from each page. */
export type LanePageCursor = {
  offset: number;
  limit: number;
  count: number;
  /** False when count is the server's estimate of more lanes than it counts. */
  countExact: boolean;
  results: readonly unknown[];
};

/**
 * Where the next page of an agreement's lanes starts, or undefined once every
 * live lane is read. An empty page ends the reading too, so a count that moved
 * while paging never asks forever. An estimated count says nothing about where
 * the lanes end, so over one a full page is followed by the next.
 */
export function nextLaneOffset(page: LanePageCursor): number | undefined {
  const next = page.offset + page.results.length;
  if (page.results.length === 0) {
    return undefined;
  }
  if (!page.countExact) {
    return page.results.length >= page.limit ? next : undefined;
  }
  return next < page.count ? next : undefined;
}
