import { z } from "zod";

const DECIMAL_INPUT = /^([+-])?(\d+)(?:\.(\d+))?$/;

type ScaledDecimal = {
  unscaled: bigint;
  scale: number;
};

function assertScale(scale: number) {
  if (!Number.isInteger(scale) || scale < 0) {
    throw new RangeError(`Decimal scale must be a non-negative integer, got ${scale}`);
  }
}

function parseDecimal(value: string): ScaledDecimal {
  const match = DECIMAL_INPUT.exec(value.trim());
  if (!match) {
    throw new TypeError(`Invalid decimal string: "${value}"`);
  }
  const [, sign, integerPart, fractionPart = ""] = match;
  const magnitude = BigInt(integerPart + fractionPart);
  return {
    unscaled: sign === "-" ? -magnitude : magnitude,
    scale: fractionPart.length,
  };
}

function rescale(value: ScaledDecimal, scale: number): bigint {
  if (scale >= value.scale) {
    return value.unscaled * 10n ** BigInt(scale - value.scale);
  }
  const divisor = 10n ** BigInt(value.scale - scale);
  const negative = value.unscaled < 0n;
  const magnitude = negative ? -value.unscaled : value.unscaled;
  let quotient = magnitude / divisor;
  if ((magnitude % divisor) * 2n >= divisor) {
    quotient += 1n;
  }
  return negative ? -quotient : quotient;
}

function stringify(unscaled: bigint, scale: number): string {
  const negative = unscaled < 0n;
  const digits = (negative ? -unscaled : unscaled).toString().padStart(scale + 1, "0");
  const integerPart = digits.slice(0, digits.length - scale);
  const fractionPart = digits.slice(digits.length - scale);
  const body = scale === 0 ? integerPart : `${integerPart}.${fractionPart}`;
  return negative && unscaled !== 0n ? `-${body}` : body;
}

export function decimalPattern(scale: number): RegExp {
  assertScale(scale);
  return scale === 0 ? /^-?\d+$/ : new RegExp(`^-?\\d+(\\.\\d{1,${scale}})?$`);
}

export function compareDecimalStrings(a: string, b: string): -1 | 0 | 1 {
  const left = parseDecimal(a);
  const right = parseDecimal(b);
  const scale = Math.max(left.scale, right.scale);
  const leftScaled = rescale(left, scale);
  const rightScaled = rescale(right, scale);
  if (leftScaled < rightScaled) return -1;
  if (leftScaled > rightScaled) return 1;
  return 0;
}

export function decimalString(scale: number, message: string) {
  return z.string().trim().regex(decimalPattern(scale), { message });
}

export function isDecimalString(value: string): boolean {
  return DECIMAL_INPUT.test(value.trim());
}

export function positiveDecimalString(scale: number, message: string) {
  return decimalString(scale, message).refine(
    (value) => isDecimalString(value) && compareDecimalStrings(value, "0") > 0,
    { message },
  );
}

export function nonNegativeDecimalString(scale: number, message: string) {
  return decimalString(scale, message).refine(
    (value) => isDecimalString(value) && compareDecimalStrings(value, "0") >= 0,
    { message },
  );
}

export function optionalDecimalString(scale: number, message: string) {
  return z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().regex(decimalPattern(scale), { message }).nullable());
}

export function optionalNonNegativeDecimalString(scale: number, message: string) {
  return optionalDecimalString(scale, message).refine(
    (value) => value === null || (isDecimalString(value) && compareDecimalStrings(value, "0") >= 0),
    { message },
  );
}

export function multiplyDecimalStrings(a: string, b: string, scale: number): string {
  assertScale(scale);
  const left = parseDecimal(a);
  const right = parseDecimal(b);
  const product: ScaledDecimal = {
    unscaled: left.unscaled * right.unscaled,
    scale: left.scale + right.scale,
  };
  return stringify(rescale(product, scale), scale);
}

export function addDecimalStrings(values: readonly string[], scale: number): string {
  assertScale(scale);
  const parsed = values.map(parseDecimal);
  let workingScale = scale;
  for (const value of parsed) {
    if (value.scale > workingScale) workingScale = value.scale;
  }
  let total = 0n;
  for (const value of parsed) {
    total += rescale(value, workingScale);
  }
  return stringify(rescale({ unscaled: total, scale: workingScale }, scale), scale);
}

export function formatDecimalString(value: string, scale: number): string {
  assertScale(scale);
  return stringify(rescale(parseDecimal(value), scale), scale);
}
