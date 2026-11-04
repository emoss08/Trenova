import type { DetentionIntelligence } from "@/lib/graphql/detention-intelligence";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { CAUSES, CAUSE_LABELS, amount, causeMinutes, shareOf } from "./intelligence-model";

/**
 * One CSV cell. A facility name is typed by people, and a spreadsheet runs a cell that starts
 * with =, +, -, @ or a control character as a formula, so such a cell is opened with a quote.
 */
export function csvCell(value: string | number): string {
  const raw = String(value);
  const text = /^[=+\-@\t\r]/.test(raw) ? `'${raw}` : raw;
  return /[",\n\r]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}

export function toCsv(rows: readonly (readonly (string | number)[])[]): string {
  return rows.map((row) => row.map(csvCell).join(",")).join("\n");
}

const roundTo = (value: number, places: number) => Number(value.toFixed(places));
const hoursOf = (minutes: number) => roundTo(minutes / 60, 1);

/**
 * Every facility on the page, one row each, as figures a spreadsheet can sum: hours, shares and
 * amounts as plain numbers. Driver pay and net margin are columns only for a reader who can see
 * margins.
 */
export function intelligenceCsv(data: DetentionIntelligence, t: TranslateFn): string {
  const margins = data.totals.netMargin !== null;
  const header = [
    t("Facility"),
    t("Stops"),
    t("Stops detained"),
    t("Hours past free time"),
    ...CAUSES.map((cause) => CAUSE_LABELS[cause.key]),
    t("Median stop"),
    t("Slowest one in ten"),
    t("Free time granted"),
    t("Billed"),
    ...(margins ? [t("Driver pay"), t("Net")] : []),
    t("Forgiven"),
    t("Notice on time"),
  ];
  const rows = data.facilities.map((facility) => [
    facility.locationName,
    facility.stopCount,
    roundTo(shareOf(facility.detainedCount, facility.stopCount), 3),
    hoursOf(facility.pastFreeMinutes),
    ...CAUSES.map((cause) => hoursOf(causeMinutes(facility, cause.key))),
    Math.round(facility.medianDwellMinutes),
    Math.round(facility.p90DwellMinutes),
    Math.round(facility.medianFreeMinutes),
    roundTo(amount(facility.billedAmount), 2),
    ...(margins
      ? [roundTo(amount(facility.driverPayAmount), 2), roundTo(amount(facility.netMargin), 2)]
      : []),
    roundTo(amount(facility.waivedAmount), 2),
    facility.noticeDueCount > 0
      ? roundTo(shareOf(facility.noticeOnTimeCount, facility.noticeDueCount), 3)
      : "",
  ]);

  return toCsv([header, ...rows]);
}
