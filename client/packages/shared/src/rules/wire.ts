// wire.ts shapes form and GraphQL values into what the rules module reads, for the typed
// wrappers beside it (allocation.ts, report.ts, lane.ts, matrix.ts).
//
// The module is strict on purpose: a decimal field holds a JSON number, a decimal string, or
// nothing (null, missing, a blank string), and anything else refuses the whole request as a
// caller bug. A form, though, passes through states that are neither while somebody types
// ("abc", "1.", NaN), and its own field schema already reports those. So the wrappers send
// such a value as "not filled in" rather than failing the screen, and pass every real number
// through untouched: a JS number is written by JSON.stringify as its shortest exact text, and a
// decimal string as it is, so no float arithmetic happens on the way in.

/** An exact decimal as the module writes it: always a string, never a float. */
export type DecimalString = string;

/** A decimal as a caller holds it: a number from a form, a string from GraphQL, or nothing. */
export type DecimalInput = number | string | null | undefined;

const DECIMAL_TEXT = /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/;

/** A decimal the module accepts, or null for "not filled in". */
export function wireDecimal(value: DecimalInput): number | string | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  return DECIMAL_TEXT.test(trimmed) ? trimmed : null;
}

/** A whole number the module accepts (counts, positions, epoch seconds), or null. */
export function wireInteger(value: unknown): number | null {
  const number = typeof value === "string" && value.trim() !== "" ? Number(value) : value;
  if (typeof number !== "number" || !Number.isFinite(number)) return null;
  return Math.trunc(number);
}

/** Text the module accepts, or null. */
export function wireText(value: unknown): string | null {
  return typeof value === "string" ? value : null;
}

/** A list the module accepts, or null (which it reads as empty). */
export function wireList<T>(value: readonly T[] | null | undefined): readonly T[] | null {
  return Array.isArray(value) ? value : null;
}
