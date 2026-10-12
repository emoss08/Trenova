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
import type { CardFact } from "../../record-card/record-card";
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
