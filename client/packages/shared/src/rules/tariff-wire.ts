// tariff-wire.ts reads and writes Tariff's wire values (docs/engineering/formula-language.md,
// "Wire values") and the units they carry. A form holds what a person typed (a decimal text, a
// unit, a date); the module, the server and Postgres hold the tagged JSON. These helpers are the
// one crossing between them, so a rate never loses its unit or its precision on the way.
//
// The unit table mirrors the crate's (`services/crates/rules/src/tariff/units.rs`): the same
// symbols, dimensions and exact factors to the dimension's base unit. Decimals stay text; nothing
// here does arithmetic on a float.

import {
  TARIFF_DECIMAL_PATTERN,
  type TariffType,
  type TariffWireDate,
  type TariffWireNumber,
  type TariffWireQuantity,
  type TariffWireRecord,
  type TariffWireValue,
} from "@trenova/shared/types/tariff";

/** The base dimensions a unit measures; `Count` and `Percent` are dimensionless. */
export type TariffDimension =
  | "Money"
  | "Weight"
  | "Distance"
  | "Length"
  | "Volume"
  | "Duration"
  | "Temperature"
  | "Count"
  | "Percent";

export type TariffSimpleUnit = {
  symbol: string;
  dimension: TariffDimension;
  /** The exact factor to the dimension's base unit (kg, m, m3, s), as decimal text. */
  factor: string;
};

/** Every simple unit the language knows, in the crate's order. */
export const TARIFF_UNITS: readonly TariffSimpleUnit[] = [
  { symbol: "$", dimension: "Money", factor: "1" },
  { symbol: "lb", dimension: "Weight", factor: "0.45359237" },
  { symbol: "cwt", dimension: "Weight", factor: "45.359237" },
  { symbol: "kg", dimension: "Weight", factor: "1" },
  { symbol: "t", dimension: "Weight", factor: "1000" },
  { symbol: "ton", dimension: "Weight", factor: "907.18474" },
  { symbol: "mi", dimension: "Distance", factor: "1609.344" },
  { symbol: "km", dimension: "Distance", factor: "1000" },
  { symbol: "in", dimension: "Length", factor: "0.0254" },
  { symbol: "ft", dimension: "Length", factor: "0.3048" },
  { symbol: "m", dimension: "Length", factor: "1" },
  { symbol: "ft3", dimension: "Volume", factor: "0.028316846592" },
  { symbol: "m3", dimension: "Volume", factor: "1" },
  { symbol: "gal", dimension: "Volume", factor: "0.003785411784" },
  { symbol: "l", dimension: "Volume", factor: "0.001" },
  { symbol: "min", dimension: "Duration", factor: "60" },
  { symbol: "h", dimension: "Duration", factor: "3600" },
  { symbol: "d", dimension: "Duration", factor: "86400" },
  { symbol: "°F", dimension: "Temperature", factor: "1" },
  { symbol: "°C", dimension: "Temperature", factor: "1" },
  { symbol: "%", dimension: "Percent", factor: "0.01" },
  { symbol: "pallet", dimension: "Count", factor: "1" },
  { symbol: "piece", dimension: "Count", factor: "1" },
  { symbol: "stop", dimension: "Count", factor: "1" },
];

const UNIT_BY_SYMBOL = new Map(TARIFF_UNITS.map((unit) => [unit.symbol, unit]));

/** A unit as written on the wire: `lb`, `$`, `$/mi`, `lb/ft3`, `%`. */
export type TariffUnit = {
  /** The unit over the bar (`$` in `$/mi`), or null for `1/x`. */
  numerator: TariffSimpleUnit | null;
  /** The units under the bar, in order. */
  denominator: TariffSimpleUnit[];
};

/** Reads a wire unit, or null when a part of it names no unit. */
export function parseTariffUnit(text: string): TariffUnit | null {
  const parts = text.split("/").map((part) => part.trim());
  const [top, ...bottom] = parts;
  if (top === undefined || parts.some((part) => part === "")) return null;
  const numerator = top === "1" ? null : (UNIT_BY_SYMBOL.get(top) ?? undefined);
  if (numerator === undefined) return null;
  const denominator: TariffSimpleUnit[] = [];
  for (const part of bottom) {
    const unit = UNIT_BY_SYMBOL.get(part);
    if (!unit) return null;
    denominator.push(unit);
  }
  return { numerator, denominator };
}

/** Writes a unit the way the crate does: `$/mi`, `lb/ft3`. */
export function formatTariffUnit(unit: TariffUnit): string {
  const top = unit.numerator?.symbol ?? "1";
  return [top, ...unit.denominator.map((part) => part.symbol)].join("/");
}

/**
 * A unit's dimension as the checker names it: `Weight`, `Money/Distance`, `Weight/Volume`. A count
 * label (`pallet`, `piece`, `stop`) folds into the value as it does in the crate, so `$/pallet`
 * is `Money` and `pallet` alone is a `Number`.
 */
export function tariffUnitDimension(unit: TariffUnit): string {
  const counted = (part: TariffSimpleUnit | null) => part?.dimension === "Count";
  const top = unit.numerator && !counted(unit.numerator) ? unit.numerator.dimension : "Number";
  const under = unit.denominator.filter((part) => !counted(part));
  if (under.length === 0) return top;
  return [top, ...under.map((part) => part.dimension)].join("/");
}

/** Whether a unit is money or money per something: shown with the currency in front. */
export function isMoneyUnit(unit: TariffUnit): boolean {
  return unit.numerator?.symbol === "$";
}

/**
 * The units a value of `dimension` may be written in (`Weight` → lb, cwt, kg, t, ton;
 * `Money/Weight` → $/lb, $/cwt, …), for a unit select. A dimension with no units (an unknown
 * name) gives an empty list.
 */
export function tariffUnitsOf(dimension: string): string[] {
  const [top, ...bottom] = dimension.split("/");
  if (top === undefined) return [];
  const tops = top === "Money" ? ["$"] : symbolsOf(top);
  let combined = tops;
  for (const part of bottom) {
    const under = symbolsOf(part);
    combined = combined.flatMap((prefix) => under.map((symbol) => `${prefix}/${symbol}`));
  }
  return combined;
}

function symbolsOf(dimension: string): string[] {
  return TARIFF_UNITS.filter((unit) => unit.dimension === dimension).map((unit) => unit.symbol);
}

/** Decimal text the module reads exactly, or null when it is not one (`"1."`, `"abc"`, `""`). */
export function tariffDecimal(text: string | number | null | undefined): string | null {
  if (typeof text === "number") {
    return Number.isFinite(text) ? tariffDecimal(String(text)) : null;
  }
  if (typeof text !== "string") return null;
  const trimmed = text.trim();
  if (!TARIFF_DECIMAL_PATTERN.test(trimmed)) return null;
  return normalizeDecimal(trimmed);
}

/** The canonical form of decimal text: no leading zeros, no trailing fraction zeros, no `-0`. */
export function normalizeDecimal(text: string): string {
  const negative = text.startsWith("-");
  const unsigned = negative ? text.slice(1) : text;
  const [whole = "0", fraction = ""] = unsigned.split(".");
  const integer = whole.replace(/^0+(?=\d)/, "");
  const decimals = fraction.replace(/0+$/, "");
  const body = decimals ? `${integer}.${decimals}` : integer;
  return negative && /[1-9]/.test(body) ? `-${body}` : body;
}

/** `Int` or `Number`: `{"$n": "1200"}`. */
export function tariffNumber(text: string): TariffWireNumber {
  return { $n: normalizeDecimal(text) };
}

/** A quantity, money, a rate or a percent: `{"$q": "2.35", "u": "$/mi"}`. */
export function tariffQuantity(amount: string, unit: string): TariffWireQuantity {
  return { $q: normalizeDecimal(amount), u: unit };
}

/** Money in the rating currency: `{"$q": "450", "u": "$"}`. */
export function tariffMoney(amount: string): TariffWireQuantity {
  return tariffQuantity(amount, "$");
}

/** A percent, written as the number before `%`: `18` is 18%. */
export function tariffPercent(amount: string): TariffWireQuantity {
  return tariffQuantity(amount, "%");
}

/** A date (`2026-10-22`) or a date-time with its offset (`2026-10-22T08:00:00-05:00`). */
export function tariffDate(text: string): TariffWireDate {
  return { $date: text };
}

const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const DATE_TIME_PATTERN = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?(Z|[+-]\d{2}:\d{2})$/;

/** A wire value read into what it is, for display and for form fields. */
export type TariffValue =
  | { kind: "bool"; value: boolean }
  | { kind: "text"; value: string }
  | { kind: "none" }
  | { kind: "number"; value: string }
  | { kind: "quantity"; amount: string; unit: TariffUnit; unitText: string }
  | { kind: "date"; value: string }
  | {
      kind: "dateTime";
      value: string;
      date: string;
      hour: number;
      minute: number;
      second: number;
      offset: string;
    }
  | { kind: "list"; items: TariffValue[] }
  | { kind: "record"; fields: { name: string; value: TariffValue }[] };

export class TariffWireError extends Error {
  readonly path: string;
  constructor(path: string, detail: string) {
    super(`${path || "value"}: ${detail}`);
    this.name = "TariffWireError";
    this.path = path;
  }
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** Whether a value is a tagged wire number. */
export function isTariffNumber(value: unknown): value is TariffWireNumber {
  return isObject(value) && Object.keys(value).length === 1 && typeof value.$n === "string";
}

/** Whether a value is a tagged wire quantity. */
export function isTariffQuantity(value: unknown): value is TariffWireQuantity {
  return (
    isObject(value) &&
    Object.keys(value).length === 2 &&
    typeof value.$q === "string" &&
    typeof value.u === "string"
  );
}

/** Whether a value is a tagged wire date or date-time. */
export function isTariffDate(value: unknown): value is TariffWireDate {
  return isObject(value) && Object.keys(value).length === 1 && typeof value.$date === "string";
}

/**
 * Reads a wire value. A JSON string is text; give the type to read it as an enum variant
 * (which reads the same) or to check the shape against a declaration.
 *
 * @throws TariffWireError when the value is not a wire value, naming where in it.
 */
export function decodeTariffValue(wire: unknown, path = ""): TariffValue {
  if (wire === null) return { kind: "none" };
  if (typeof wire === "boolean") return { kind: "bool", value: wire };
  if (typeof wire === "string") return { kind: "text", value: wire };
  if (Array.isArray(wire)) {
    return {
      kind: "list",
      items: wire.map((item, index) => decodeTariffValue(item, `${path}[${index}]`)),
    };
  }
  if (!isObject(wire)) throw new TariffWireError(path, "not a Tariff value");
  if ("$n" in wire) {
    if (!isTariffNumber(wire) || !TARIFF_DECIMAL_PATTERN.test(wire.$n)) {
      throw new TariffWireError(path, "a number is {$n: decimal text}");
    }
    return { kind: "number", value: normalizeDecimal(wire.$n) };
  }
  if ("$q" in wire) {
    if (!isTariffQuantity(wire) || !TARIFF_DECIMAL_PATTERN.test(wire.$q)) {
      throw new TariffWireError(path, "a quantity is {$q: decimal text, u: unit}");
    }
    const unit = parseTariffUnit(wire.u);
    if (!unit) throw new TariffWireError(path, `${wire.u} is not a unit`);
    return {
      kind: "quantity",
      amount: normalizeDecimal(wire.$q),
      unit,
      unitText: formatTariffUnit(unit),
    };
  }
  if ("$date" in wire) {
    if (!isTariffDate(wire)) throw new TariffWireError(path, "a date is {$date: text}");
    return decodeDate(wire.$date, path);
  }
  const fields = Object.entries(wire).map(([name, value]) => {
    if (name.startsWith("$")) throw new TariffWireError(path, `${name} is not a record field`);
    return { name, value: decodeTariffValue(value, path ? `${path}.${name}` : name) };
  });
  return { kind: "record", fields };
}

function decodeDate(text: string, path: string): TariffValue {
  if (DATE_PATTERN.test(text)) return { kind: "date", value: text };
  const match = DATE_TIME_PATTERN.exec(text);
  if (!match) throw new TariffWireError(path, `${text} is not a date or a date-time`);
  const [, date = "", hour = "0", minute = "0", second = "0", offset = "Z"] = match;
  return {
    kind: "dateTime",
    value: text,
    date,
    hour: Number(hour),
    minute: Number(minute),
    second: Number(second),
    offset: offset === "Z" ? "+00:00" : offset,
  };
}

/** Writes a read value back into its wire form. */
export function encodeTariffValue(value: TariffValue): TariffWireValue {
  switch (value.kind) {
    case "none":
      return null;
    case "bool":
    case "text":
      return value.value;
    case "number":
      return tariffNumber(value.value);
    case "quantity":
      return tariffQuantity(value.amount, value.unitText);
    case "date":
    case "dateTime":
      return tariffDate(value.value);
    case "list":
      return value.items.map(encodeTariffValue);
    case "record": {
      const record: TariffWireRecord = {};
      for (const field of value.fields) record[field.name] = encodeTariffValue(field.value);
      return record;
    }
  }
}

/** Why a wire value does not fit a type, at a path inside it (`[2].rate`). */
export type TariffValueProblem = {
  path: string;
  problem:
    | "missing"
    | "not-bool"
    | "not-text"
    | "not-number"
    | "not-int"
    | "not-money"
    | "wrong-unit"
    | "not-percent"
    | "not-date"
    | "not-date-time"
    | "not-variant"
    | "not-list"
    | "not-record"
    | "unknown-field";
  /** What was expected: a unit's dimension, an enum's variants, a field name. */
  detail?: string;
};

/**
 * Checks a wire value against an input's type descriptor, the way `tariff.check` checks a
 * contract's values: units in the type's dimension, whole numbers for `int`, enum variants,
 * records field by field with no missing or extra fields, and `null` only for an optional.
 * The module stays the authority; this lets a form say what is wrong before it asks.
 */
export function tariffValueProblems(
  type: TariffType,
  wire: TariffWireValue | undefined,
  path = "",
): TariffValueProblem[] {
  if (wire === undefined) return [{ path, problem: "missing" }];
  switch (type.kind) {
    case "unknown":
      return [];
    case "optional":
      return wire === null ? [] : tariffValueProblems(type.inner, wire, path);
    case "bool":
      return typeof wire === "boolean" ? [] : [{ path, problem: "not-bool" }];
    case "text":
      return typeof wire === "string" ? [] : [{ path, problem: "not-text" }];
    case "number":
      return isTariffNumber(wire) && TARIFF_DECIMAL_PATTERN.test(wire.$n)
        ? []
        : [{ path, problem: "not-number" }];
    case "int":
      return isTariffNumber(wire) && /^-?\d+$/.test(normalizeDecimal(wire.$n))
        ? []
        : [{ path, problem: "not-int" }];
    case "money": {
      const unit = isTariffQuantity(wire) ? parseTariffUnit(wire.u) : null;
      return unit !== null &&
        isTariffQuantity(wire) &&
        TARIFF_DECIMAL_PATTERN.test(wire.$q) &&
        tariffUnitDimension(unit) === "Money"
        ? []
        : [{ path, problem: "not-money" }];
    }
    case "percent":
      return isTariffQuantity(wire) && wire.u === "%" && TARIFF_DECIMAL_PATTERN.test(wire.$q)
        ? []
        : [{ path, problem: "not-percent" }];
    case "quantity": {
      const unit = isTariffQuantity(wire) ? parseTariffUnit(wire.u) : null;
      const fits =
        unit !== null &&
        isTariffQuantity(wire) &&
        TARIFF_DECIMAL_PATTERN.test(wire.$q) &&
        tariffUnitDimension(unit) === type.dimension;
      return fits ? [] : [{ path, problem: "wrong-unit", detail: type.dimension }];
    }
    case "date":
      return isTariffDate(wire) && DATE_PATTERN.test(wire.$date)
        ? []
        : [{ path, problem: "not-date" }];
    case "dateTime":
      return isTariffDate(wire) && DATE_TIME_PATTERN.test(wire.$date)
        ? []
        : [{ path, problem: "not-date-time" }];
    case "enum":
      return typeof wire === "string" && type.variants.includes(wire)
        ? []
        : [{ path, problem: "not-variant", detail: type.variants.join(", ") }];
    case "list":
      return Array.isArray(wire)
        ? wire.flatMap((item, index) => tariffValueProblems(type.item, item, `${path}[${index}]`))
        : [{ path, problem: "not-list" }];
    case "record": {
      if (!isObject(wire) || isTariffNumber(wire) || isTariffQuantity(wire) || isTariffDate(wire)) {
        return [{ path, problem: "not-record" }];
      }
      const record = wire as TariffWireRecord;
      const known = new Set(type.fields.map((field) => field.name));
      const extra = Object.keys(record)
        .filter((name) => !known.has(name))
        .map((name): TariffValueProblem => ({ path, problem: "unknown-field", detail: name }));
      const fields = type.fields.flatMap((field) => {
        const at = path ? `${path}.${field.name}` : field.name;
        const value = record[field.name];
        if (value === undefined && field.type.kind === "optional") return [];
        return tariffValueProblems(field.type, value, at);
      });
      return [...fields, ...extra];
    }
  }
}

/** The empty value a new row of a list-of-records input starts with, field by field. */
export function emptyTariffValue(type: TariffType): TariffWireValue {
  switch (type.kind) {
    case "optional":
    case "unknown":
      return null;
    case "bool":
      return false;
    case "text":
      return "";
    case "int":
    case "number":
      return tariffNumber("0");
    case "money":
      return tariffMoney("0");
    case "percent":
      return tariffPercent("0");
    case "quantity":
      return tariffQuantity("0", type.unit);
    case "date":
    case "dateTime":
      return null;
    case "enum":
      return type.variants[0] ?? null;
    case "list":
      return [];
    case "record": {
      const record: TariffWireRecord = {};
      for (const field of type.fields) record[field.name] = emptyTariffValue(field.type);
      return record;
    }
  }
}
