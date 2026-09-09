import type { IftaTaxRateInput } from "@trenova/graphql/generated/graphql";
import { compareDecimalStrings, decimalPattern } from "@trenova/shared/types/decimal";
import { iftaFuelTypeSchema, type IftaFuelType } from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_RATE_SCALE } from "@trenova/shared/types/ifta-tax-rate";

export type IftaTaxRateImportJurisdiction = {
  id: string;
  countryCode: string;
  code: string;
};

export type IftaTaxRateImportRow = {
  line: number;
  jurisdictionCode: string;
  fuelType: string;
  ratePerGallon: string;
  surchargeRatePerGallon: string | null;
  error: string | null;
  input: IftaTaxRateInput | null;
};

export type IftaTaxRateImportResult = {
  fileErrors: string[];
  rows: IftaTaxRateImportRow[];
  valid: IftaTaxRateInput[];
};

type ImportOptions = {
  jurisdictions: readonly IftaTaxRateImportJurisdiction[];
  year: number;
  quarter: number;
};

type ColumnKey = "jurisdiction" | "fuelType" | "rate" | "surcharge";

const COLUMN_ALIASES: Record<ColumnKey, readonly string[]> = {
  jurisdiction: ["jurisdiction", "jurisdiction code", "code", "state", "province", "juris"],
  fuelType: ["fuel type", "fuel", "product", "fueltype"],
  rate: ["rate per gallon", "rate", "tax rate", "rate/gal", "rate per gal", "tax"],
  surcharge: [
    "surcharge per gallon",
    "surcharge",
    "surcharge rate",
    "surcharge rate per gallon",
    "surcharge/gal",
  ],
};

const REQUIRED_COLUMNS: readonly ColumnKey[] = ["jurisdiction", "fuelType", "rate"];

const COLUMN_NAMES: Record<ColumnKey, string> = {
  jurisdiction: "jurisdiction",
  fuelType: "fuel type",
  rate: "rate per gallon",
  surcharge: "surcharge",
};

export const IFTA_TAX_RATE_TEMPLATE_CSV = [
  "Jurisdiction,Fuel Type,Rate Per Gallon,Surcharge Per Gallon",
  "TX,Diesel,0.2000,",
  "IN,Diesel,0.5700,0.1100",
  "CA-BC,Diesel,0.2500,",
  "",
].join("\n");

export function iftaTaxRateTemplateFileName(year: number, quarter: number): string {
  return `ifta-tax-rates-${year}Q${quarter}.csv`;
}

const RATE_PATTERN = decimalPattern(IFTA_RATE_SCALE);

function normalizeHeader(value: string): string {
  return value.trim().toLowerCase().replace(/[_-]+/g, " ").replace(/\s+/g, " ");
}

function splitCsvLine(line: string): string[] {
  const cells: string[] = [];
  let current = "";
  let quoted = false;
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (quoted) {
      if (char === '"') {
        if (line[i + 1] === '"') {
          current += '"';
          i++;
        } else {
          quoted = false;
        }
      } else {
        current += char;
      }
    } else if (char === '"') {
      quoted = true;
    } else if (char === ",") {
      cells.push(current);
      current = "";
    } else {
      current += char;
    }
  }
  cells.push(current);
  return cells.map((cell) => cell.trim());
}

function resolveColumns(header: string[]): Partial<Record<ColumnKey, number>> {
  const columns: Partial<Record<ColumnKey, number>> = {};
  const normalized = header.map(normalizeHeader);
  for (const key of Object.keys(COLUMN_ALIASES) as ColumnKey[]) {
    for (const alias of COLUMN_ALIASES[key]) {
      const index = normalized.indexOf(alias);
      if (index !== -1 && !Object.values(columns).includes(index)) {
        columns[key] = index;
        break;
      }
    }
  }
  return columns;
}

const FUEL_TYPE_BY_KEY = new Map<string, IftaFuelType>(
  iftaFuelTypeSchema.options.map((value) => [value.toLowerCase(), value]),
);

function resolveFuelType(raw: string): IftaFuelType | null {
  return FUEL_TYPE_BY_KEY.get(raw.trim().toLowerCase().replace(/\s+/g, "")) ?? null;
}

type JurisdictionLookup = {
  byQualified: Map<string, IftaTaxRateImportJurisdiction>;
  byCode: Map<string, IftaTaxRateImportJurisdiction[]>;
};

function buildLookup(jurisdictions: readonly IftaTaxRateImportJurisdiction[]): JurisdictionLookup {
  const byQualified = new Map<string, IftaTaxRateImportJurisdiction>();
  const byCode = new Map<string, IftaTaxRateImportJurisdiction[]>();
  for (const jurisdiction of jurisdictions) {
    const code = jurisdiction.code.toUpperCase();
    byQualified.set(`${jurisdiction.countryCode.toUpperCase()}-${code}`, jurisdiction);
    const bucket = byCode.get(code);
    if (bucket) bucket.push(jurisdiction);
    else byCode.set(code, [jurisdiction]);
  }
  return { byQualified, byCode };
}

function resolveJurisdiction(
  raw: string,
  lookup: JurisdictionLookup,
): { jurisdiction: IftaTaxRateImportJurisdiction | null; error: string | null } {
  const code = raw.trim().toUpperCase();
  const qualified = lookup.byQualified.get(code);
  if (qualified) return { jurisdiction: qualified, error: null };
  const candidates = lookup.byCode.get(code) ?? [];
  if (candidates.length === 1) return { jurisdiction: candidates[0], error: null };
  if (candidates.length > 1) {
    const forms = candidates
      .map((candidate) => `${candidate.countryCode.toUpperCase()}-${code}`)
      .join(" or ");
    return {
      jurisdiction: null,
      error: `Jurisdiction "${raw}" exists in more than one country; write it as ${forms}.`,
    };
  }
  return { jurisdiction: null, error: `Unknown jurisdiction "${raw}".` };
}

function validateRate(raw: string, label: "Rate" | "Surcharge"): string | null {
  if (!RATE_PATTERN.test(raw)) return `${label} must be a number with up to four decimals.`;
  if (compareDecimalStrings(raw, "0") < 0) return `${label} cannot be negative.`;
  return null;
}

export function parseIftaTaxRateCsv(
  text: string,
  { jurisdictions, year, quarter }: ImportOptions,
): IftaTaxRateImportResult {
  const lines = text.replace(/^﻿/, "").split(/\r\n|\n|\r/);
  const headerIndex = lines.findIndex((line) => line.trim() !== "");
  if (headerIndex === -1) {
    return { fileErrors: ["The file is empty."], rows: [], valid: [] };
  }

  const columns = resolveColumns(splitCsvLine(lines[headerIndex]));
  const missing = REQUIRED_COLUMNS.filter((key) => columns[key] === undefined);
  if (missing.length > 0) {
    return {
      fileErrors: missing.map((key) => `The file has no "${COLUMN_NAMES[key]}" column.`),
      rows: [],
      valid: [],
    };
  }

  const lookup = buildLookup(jurisdictions);
  const seen = new Map<string, number>();
  const rows: IftaTaxRateImportRow[] = [];
  const valid: IftaTaxRateInput[] = [];

  for (let index = headerIndex + 1; index < lines.length; index++) {
    const line = lines[index];
    if (line.trim() === "") continue;
    const cells = splitCsvLine(line);
    const cell = (key: ColumnKey) => {
      const column = columns[key];
      return column === undefined ? "" : (cells[column] ?? "");
    };
    const row: IftaTaxRateImportRow = {
      line: index + 1,
      jurisdictionCode: cell("jurisdiction"),
      fuelType: cell("fuelType"),
      ratePerGallon: cell("rate"),
      surchargeRatePerGallon: cell("surcharge") === "" ? null : cell("surcharge"),
      error: null,
      input: null,
    };
    rows.push(row);

    if (row.jurisdictionCode === "") {
      row.error = "Jurisdiction is required.";
      continue;
    }
    const { jurisdiction, error: jurisdictionError } = resolveJurisdiction(
      row.jurisdictionCode,
      lookup,
    );
    if (!jurisdiction) {
      row.error = jurisdictionError;
      continue;
    }
    if (row.fuelType === "") {
      row.error = "Fuel type is required.";
      continue;
    }
    const fuelType = resolveFuelType(row.fuelType);
    if (!fuelType) {
      row.error = `Unknown fuel type "${row.fuelType}".`;
      continue;
    }
    row.fuelType = fuelType;
    if (row.ratePerGallon === "") {
      row.error = "Rate is required.";
      continue;
    }
    const rateError = validateRate(row.ratePerGallon, "Rate");
    if (rateError) {
      row.error = rateError;
      continue;
    }
    if (row.surchargeRatePerGallon !== null) {
      const surchargeError = validateRate(row.surchargeRatePerGallon, "Surcharge");
      if (surchargeError) {
        row.error = surchargeError;
        continue;
      }
    }
    const key = `${jurisdiction.id}|${fuelType}`;
    const firstLine = seen.get(key);
    if (firstLine !== undefined) {
      row.error = `Duplicate of line ${firstLine} (${jurisdiction.code.toUpperCase()}, ${fuelType}).`;
      continue;
    }
    seen.set(key, row.line);

    row.input = {
      jurisdictionId: jurisdiction.id,
      year,
      quarter,
      fuelType,
      ratePerGallon: row.ratePerGallon,
      surchargeRatePerGallon: row.surchargeRatePerGallon,
    };
    valid.push(row.input);
  }

  if (rows.length === 0) {
    return { fileErrors: ["The file has a header row but no rates."], rows: [], valid: [] };
  }

  return { fileErrors: [], rows, valid };
}
