import type { DeskTrailer } from "@/lib/graphql/trailer-card";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { worstFirst, type DueCheck } from "../../record-card/card-due";
import {
  equipmentFacts,
  equipmentStatusCheck,
  leaseCheck,
  registrationCheck,
} from "../../record-card/equipment";
import type { CardCheckState, CardFact } from "../../record-card/record-card";
import { cardDate, dueReading } from "../../record-card/record-card-values";

const DAY_SECONDS = 86_400;
/** A trailer is inspected at least once every twelve months (49 CFR 396.17). */
export const ANNUAL_INSPECTION_DAYS = 365;
/** An annual inspection falling due inside this many days is worth a look. */
export const INSPECTION_NOTICE_DAYS = 30;

/** When the trailer's next annual inspection falls due; null with none on file. */
export function nextInspectionDue(lastInspectionDate: number | null | undefined): number | null {
  return lastInspectionDate ? lastInspectionDate + ANNUAL_INSPECTION_DAYS * DAY_SECONDS : null;
}

/** The trailer's annual inspection against now: none on file, overdue, close, or good. */
export function annualInspectionCheck(
  lastInspectionDate: number | null | undefined,
  now: number,
  t: TranslateFn,
): DueCheck {
  const due = nextInspectionDue(lastInspectionDate);
  if (due === null || !lastInspectionDate) {
    return {
      key: "inspection",
      state: "warn",
      title: t("Annual inspection"),
      detail: t("No inspection on file"),
    };
  }
  const reading = dueReading(due, now, INSPECTION_NOTICE_DAYS);
  if (reading.state === "bad") {
    return {
      key: "inspection",
      state: "bad",
      title: t("Annual inspection overdue"),
      detail: t("Was due {0} · last {1}", cardDate(due), cardDate(lastInspectionDate)),
    };
  }
  return {
    key: "inspection",
    state: reading.state,
    title:
      reading.state === "warn"
        ? t("Annual inspection due in {0, plural, one {# day} other {# days}}", reading.days)
        : t("Annual inspection"),
    detail: t("Due {0} · last {1}", cardDate(due), cardDate(lastInspectionDate)),
  };
}

/**
 * What stands between the trailer and its next load, worst first: whether it
 * is in service, its annual inspection, its registration and the lease on a
 * leased trailer.
 */
export function trailerChecks(trailer: DeskTrailer, now: number, t: TranslateFn): DueCheck[] {
  const checks: DueCheck[] = [];
  const status = equipmentStatusCheck(trailer.status, t);
  if (status) checks.push(status);
  checks.push(annualInspectionCheck(trailer.lastInspectionDate, now, t));
  checks.push(registrationCheck(trailer.registrationExpiry, now, t));
  const lease = leaseCheck(trailer, now, t);
  if (lease) checks.push(lease);
  return worstFirst(checks);
}

/** The most the trailer may carry, as the card writes it; null with none on file. */
export function maxLoadText(
  maxLoadWeight: number | null | undefined,
  t: TranslateFn,
): string | null {
  return maxLoadWeight && maxLoadWeight > 0 ? t("{0} lb", formatNumber(maxLoadWeight)) : null;
}

/** The facts the card lists about the trailer itself, in the order a dispatcher reads them. */
export function trailerFacts(trailer: DeskTrailer, t: TranslateFn): CardFact[] {
  return equipmentFacts(
    trailer,
    trailer.registrationState?.abbreviation,
    [
      {
        key: "load",
        label: t("Max load"),
        value: maxLoadText(trailer.maxLoadWeight, t) ?? t("None on file"),
      },
    ],
    t,
  );
}

/** One date on the trailer's service calendar, where it falls and how it stands. */
export type CalendarMark = {
  key: "last" | "inspection" | "registration" | "lease";
  label: string;
  at: number;
  state: CardCheckState | "past";
};

/** The span the calendar draws: a season back and a year ahead of now. */
export const CALENDAR_BACK_DAYS = 90;
export const CALENDAR_AHEAD_DAYS = 365;

/**
 * The dates the trailer's papers hang on, as marks on its service calendar:
 * the last annual inspection, when the next falls due, when its registration
 * runs out, and when the lease on a leased trailer ends, each in the state its
 * check reads.
 */
export function calendarMarks(
  trailer: Pick<
    DeskTrailer,
    "lastInspectionDate" | "registrationExpiry" | "ownershipType" | "leaseEndDate" | "lessorName"
  >,
  now: number,
  t: TranslateFn,
): CalendarMark[] {
  const marks: CalendarMark[] = [];
  if (trailer.lastInspectionDate) {
    marks.push({
      key: "last",
      label: t("Last inspected"),
      at: trailer.lastInspectionDate,
      state: "past",
    });
    const due = nextInspectionDue(trailer.lastInspectionDate);
    if (due !== null) {
      marks.push({
        key: "inspection",
        label: t("Inspection due"),
        at: due,
        state: annualInspectionCheck(trailer.lastInspectionDate, now, t).state,
      });
    }
  }
  if (trailer.registrationExpiry) {
    marks.push({
      key: "registration",
      label: t("Registration ends"),
      at: trailer.registrationExpiry,
      state: registrationCheck(trailer.registrationExpiry, now, t).state,
    });
  }
  const lease = leaseCheck(trailer, now, t);
  if (lease && trailer.leaseEndDate) {
    marks.push({
      key: "lease",
      label: t("Lease ends"),
      at: trailer.leaseEndDate,
      state: lease.state,
    });
  }
  return marks.sort((a, b) => a.at - b.at);
}

/**
 * Where a moment falls along the calendar, from 0 at its start to 1 at its
 * end, held to the ends for one outside it; `outside` says which way it ran off.
 */
export function calendarPlace(
  at: number,
  now: number,
): { share: number; outside: "before" | "after" | null } {
  const start = now - CALENDAR_BACK_DAYS * DAY_SECONDS;
  const end = now + CALENDAR_AHEAD_DAYS * DAY_SECONDS;
  const share = (at - start) / (end - start);
  if (share < 0) return { share: 0, outside: "before" };
  if (share > 1) return { share: 1, outside: "after" };
  return { share, outside: null };
}
