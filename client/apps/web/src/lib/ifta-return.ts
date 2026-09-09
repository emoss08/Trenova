import type { ExportColumn } from "@/lib/data-table-export";
import type { IftaPeriod, IftaReturn, IftaReturnLine } from "@/lib/graphql/ifta-return";
import type { IftaProblemCode } from "@trenova/graphql/generated/graphql";
import { toUserWallClock } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import {
  addDecimalStrings,
  compareDecimalStrings,
  formatDecimalString,
} from "@trenova/shared/types/decimal";
import type { IftaFuelType, IftaReturnStatus } from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_MILES_SCALE } from "@trenova/shared/types/ifta-jurisdiction-mileage";
import { IFTA_MAX_YEAR, IFTA_MIN_YEAR } from "@trenova/shared/types/ifta-tax-rate";

export const IFTA_GALLONS_SCALE = 0;
// Miles are entered to two places but a return reports them whole, the way the
// form prints them.
export const IFTA_MILES_DISPLAY_SCALE = 0;
export const IFTA_MONEY_SCALE = 2;
export const IFTA_MPG_SCALE = 2;

export type IftaPeriodKey = { year: number; quarter: number };

/**
 * A return with its masked fragments opened. The document masks `period` and
 * `lines` behind fragment references; every derivation below reads through
 * them, so the shape they are read at is named once here.
 */
export type IftaReturnView = Omit<IftaReturn, "lines" | "period"> & {
  period: IftaPeriod;
  lines: IftaReturnLine[];
};

export function defaultIftaPeriod(nowUnix: number, timezone?: string): IftaPeriodKey {
  const local = toUserWallClock(nowUnix, timezone) ?? new Date(nowUnix * 1000);
  const current = Math.floor(local.getMonth() / 3) + 1;
  return current === 1
    ? { year: local.getFullYear() - 1, quarter: 4 }
    : { year: local.getFullYear(), quarter: current - 1 };
}

export function quarterLabel(period: IftaPeriodKey): string {
  return `Q${period.quarter} ${period.year}`;
}

const INTEGER = /^\d+$/;

function parseBounded(value: string | null, min: number, max: number): number | null {
  if (value === null || !INTEGER.test(value)) return null;
  const parsed = Number(value);
  return parsed >= min && parsed <= max ? parsed : null;
}

export function periodFromSearch(
  year: string | null,
  quarter: string | null,
  fallback: IftaPeriodKey,
): IftaPeriodKey {
  const parsedYear = parseBounded(year, IFTA_MIN_YEAR, IFTA_MAX_YEAR);
  const parsedQuarter = parseBounded(quarter, 1, 4);
  if (parsedYear === null || parsedQuarter === null) return fallback;
  return { year: parsedYear, quarter: parsedQuarter };
}

export type IftaLineTotals = {
  totalMiles: string;
  taxableMiles: string;
  taxPaidGallons: string;
  taxableGallons: string;
  netTaxableGallons: string;
  taxDue: string;
  surchargeDue: string;
  lineTotal: string;
};

type SummableLine = Pick<IftaReturnLine, keyof IftaLineTotals>;

function sumField(lines: readonly SummableLine[], field: keyof IftaLineTotals, scale: number) {
  return addDecimalStrings(
    lines.map((line) => line[field]),
    scale,
  );
}

export function sumLines(lines: readonly SummableLine[]): IftaLineTotals {
  return {
    totalMiles: sumField(lines, "totalMiles", IFTA_MILES_SCALE),
    taxableMiles: sumField(lines, "taxableMiles", IFTA_MILES_SCALE),
    taxPaidGallons: sumField(lines, "taxPaidGallons", IFTA_GALLONS_SCALE),
    taxableGallons: sumField(lines, "taxableGallons", IFTA_GALLONS_SCALE),
    netTaxableGallons: sumField(lines, "netTaxableGallons", IFTA_GALLONS_SCALE),
    taxDue: sumField(lines, "taxDue", IFTA_MONEY_SCALE),
    surchargeDue: sumField(lines, "surchargeDue", IFTA_MONEY_SCALE),
    lineTotal: sumField(lines, "lineTotal", IFTA_MONEY_SCALE),
  };
}

export function isNegativeDecimal(value: string): boolean {
  return compareDecimalStrings(value, "0") < 0;
}

export function isCreditLine(line: Pick<IftaReturnLine, "lineTotal">): boolean {
  return isNegativeDecimal(line.lineTotal);
}

export type IftaLineGroup<TLine extends SummableLine> = {
  fuelType: IftaFuelType;
  lines: TLine[];
  subtotal: IftaLineTotals;
};

export function groupLinesByFuelType<TLine extends SummableLine & { fuelType: IftaFuelType }>(
  lines: readonly TLine[],
): IftaLineGroup<TLine>[] {
  const groups = new Map<IftaFuelType, TLine[]>();
  for (const line of lines) {
    const bucket = groups.get(line.fuelType);
    if (bucket) {
      bucket.push(line);
    } else {
      groups.set(line.fuelType, [line]);
    }
  }
  return [...groups.entries()].map(([fuelType, groupLines]) => ({
    fuelType,
    lines: groupLines,
    subtotal: sumLines(groupLines),
  }));
}

export function linesMissingRates<
  TLine extends Pick<IftaReturnLine, "rateMissing" | "isIftaMember">,
>(lines: readonly TLine[]): TLine[] {
  return lines.filter((line) => line.rateMissing && line.isIftaMember);
}

export function canFinalize(
  ret: Pick<IftaReturn, "status"> & {
    lines: readonly Pick<IftaReturnLine, "rateMissing" | "isIftaMember">[];
  },
): boolean {
  return ret.status === "Draft" && linesMissingRates(ret.lines).length === 0;
}

export type IftaReturnPermissions = {
  create: boolean;
  update: boolean;
  approve: boolean;
  reopen: boolean;
  submit: boolean;
  delete: boolean;
  export: boolean;
};

export type IftaReturnTransition =
  | "generate"
  | "recompute"
  | "finalize"
  | "reopen"
  | "markFiled"
  | "amend"
  | "delete"
  | "export";

const TRANSITIONS_BY_STATUS: Record<
  IftaReturnStatus,
  ReadonlyArray<[IftaReturnTransition, keyof IftaReturnPermissions]>
> = {
  Draft: [
    ["recompute", "update"],
    ["finalize", "approve"],
    ["delete", "delete"],
    ["export", "export"],
  ],
  Finalized: [
    ["reopen", "reopen"],
    ["markFiled", "submit"],
    ["export", "export"],
  ],
  Filed: [
    ["amend", "create"],
    ["export", "export"],
  ],
};

export function returnTransitions(
  status: IftaReturnStatus | null,
  perms: IftaReturnPermissions,
): IftaReturnTransition[] {
  if (status === null) return perms.create ? ["generate"] : [];
  return TRANSITIONS_BY_STATUS[status]
    .filter(([, permission]) => perms[permission])
    .map(([transition]) => transition);
}

export type IftaReturnCsvRow = {
  kind: "line" | "subtotal" | "total";
  fuelType: string;
  jurisdiction: string;
  member: string;
  totalMiles: string;
  taxableMiles: string;
  taxPaidGallons: string;
  taxableGallons: string;
  netTaxableGallons: string;
  ratePerGallon: string;
  taxDue: string;
  surchargeDue: string;
  lineTotal: string;
};

function csvColumn(id: keyof IftaReturnCsvRow, header: string): ExportColumn<IftaReturnCsvRow> {
  return { id, header, getValue: (row) => row[id] };
}

export const IFTA_RETURN_CSV_COLUMNS: ExportColumn<IftaReturnCsvRow>[] = [
  csvColumn("fuelType", "Fuel type"),
  csvColumn("jurisdiction", "Jurisdiction"),
  csvColumn("member", "IFTA member"),
  csvColumn("totalMiles", "Total miles"),
  csvColumn("taxableMiles", "Taxable miles"),
  csvColumn("taxPaidGallons", "Tax-paid gallons"),
  csvColumn("taxableGallons", "Taxable gallons"),
  csvColumn("netTaxableGallons", "Net taxable gallons"),
  csvColumn("ratePerGallon", "Rate per gallon"),
  csvColumn("taxDue", "Tax due"),
  csvColumn("surchargeDue", "Surcharge due"),
  csvColumn("lineTotal", "Line total"),
];

function totalsRow(
  kind: "subtotal" | "total",
  fuelType: string,
  label: string,
  totals: IftaLineTotals,
): IftaReturnCsvRow {
  return {
    kind,
    fuelType,
    jurisdiction: label,
    member: "",
    totalMiles: totals.totalMiles,
    taxableMiles: totals.taxableMiles,
    taxPaidGallons: totals.taxPaidGallons,
    taxableGallons: totals.taxableGallons,
    netTaxableGallons: totals.netTaxableGallons,
    ratePerGallon: "",
    taxDue: totals.taxDue,
    surchargeDue: totals.surchargeDue,
    lineTotal: totals.lineTotal,
  };
}

export function iftaReturnCsvRows(ret: Pick<IftaReturnView, "lines">): IftaReturnCsvRow[] {
  const groups = groupLinesByFuelType(ret.lines);
  const rows: IftaReturnCsvRow[] = [];
  for (const group of groups) {
    const fuelLabel = IFTA_FUEL_TYPE_LABELS[group.fuelType];
    for (const line of group.lines) {
      rows.push({
        kind: "line",
        fuelType: fuelLabel,
        jurisdiction: line.jurisdiction.code,
        member: line.isIftaMember ? "Yes" : "No",
        totalMiles: line.totalMiles,
        taxableMiles: line.taxableMiles,
        taxPaidGallons: line.taxPaidGallons,
        taxableGallons: line.taxableGallons,
        netTaxableGallons: line.netTaxableGallons,
        ratePerGallon: line.rateMissing ? "" : (line.ratePerGallon ?? ""),
        taxDue: line.taxDue,
        surchargeDue: line.surchargeDue,
        lineTotal: line.lineTotal,
      });
    }
    rows.push(totalsRow("subtotal", fuelLabel, `${fuelLabel} subtotal`, group.subtotal));
  }
  rows.push(totalsRow("total", "", "Total", sumLines(ret.lines)));
  return rows;
}

export function iftaReturnCsvFilename(
  ret: Pick<IftaReturn, "year" | "quarter" | "amendmentNumber">,
): string {
  const base = `ifta-return-${ret.year}q${ret.quarter}`;
  return ret.amendmentNumber > 0 ? `${base}-amendment-${ret.amendmentNumber}.csv` : `${base}.csv`;
}

export type IftaProblemTone = "danger" | "warn" | "info";

const PROBLEM_TONES: Record<IftaProblemCode, IftaProblemTone> = {
  MissingRate: "danger",
  NoFuelForType: "warn",
  UnattributedMiles: "warn",
  NoTractorMiles: "warn",
  MileageMismatch: "warn",
  NonMemberActivity: "info",
  NonQualifiedActivity: "info",
};

export function problemTone(code: IftaProblemCode): IftaProblemTone {
  return PROBLEM_TONES[code];
}

export function formatIftaMoney(value: string, currencyCode: string): string {
  return formatCurrency(Number(formatDecimalString(value, IFTA_MONEY_SCALE)), currencyCode);
}

export function formatIftaMeasure(value: string, scale: number): string {
  return Number(formatDecimalString(value, scale)).toLocaleString("en-US", {
    minimumFractionDigits: scale,
    maximumFractionDigits: scale,
  });
}

export type IftaNetPosition = {
  label: string;
  magnitude: string;
  isCredit: boolean;
};

export function netPosition(netDue: string): IftaNetPosition {
  const isCredit = isNegativeDecimal(netDue);
  const rounded = formatDecimalString(netDue, IFTA_MONEY_SCALE);
  return {
    label: isCredit ? "Net credit" : "Net tax due",
    magnitude: isCredit ? rounded.slice(1) : rounded,
    isCredit,
  };
}

const YEAR_WINDOW = 5;

export function iftaYearOptions(currentYear: number, selectedYear: number): number[] {
  const years: number[] = [];
  for (let year = currentYear; year > currentYear - YEAR_WINDOW; year--) {
    if (year >= IFTA_MIN_YEAR && year <= IFTA_MAX_YEAR) years.push(year);
  }
  if (
    !years.includes(selectedYear) &&
    selectedYear >= IFTA_MIN_YEAR &&
    selectedYear <= IFTA_MAX_YEAR
  ) {
    years.push(selectedYear);
  }
  return years;
}

const SECONDS_PER_DAY = 86_400;

export function periodInclusiveEnd(end: number): number {
  return end - SECONDS_PER_DAY;
}

type IftaReturnProblem = IftaReturn["problems"][number];

export type IftaProblemTotals = { count: number; amount: string };

/**
 * How many problems a code raised and the figure they add up to — miles for
 * the mileage codes, gallons for the fuel ones. A code that reports no figure
 * is still counted; its total stays zero rather than being invented.
 */
export function problemTotals(
  problems: readonly Pick<IftaReturnProblem, "code" | "amount">[],
  code: IftaProblemCode,
): IftaProblemTotals {
  const matching = problems.filter((problem) => problem.code === code);
  return {
    count: matching.length,
    amount: addDecimalStrings(
      matching.map((problem) => problem.amount ?? "0"),
      IFTA_MILES_SCALE,
    ),
  };
}
