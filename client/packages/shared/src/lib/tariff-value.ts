// tariff-value.ts writes a Tariff value for a person to read: `$2.35/mi`, `1,200 lb`, `18%`,
// `Oct 22, 2026`, a list by its size. Numbers are grouped in the reader's locale but formatted
// from their exact decimal text, so a rate keeps every digit it was written with.

import { dateTimeFormatter, numberFormatter } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import {
  decodeTariffValue,
  isMoneyUnit,
  TariffWireError,
  type TariffValue,
} from "@trenova/shared/rules/tariff-wire";
import type { TariffWireValue } from "@trenova/shared/types/tariff";

export type TariffValueFormatOptions = {
  /**
   * The rating currency's ISO code. Money then shows that currency's symbol (`CA$450.00`);
   * without one it shows `$`, which in Tariff means "the rating currency".
   */
  currency?: string | null;
};

// Intl.NumberFormat reads a decimal string exactly (ES2023); the ES2022 lib this package
// compiles against only declares numbers, so the string overload is named here once.
type DecimalFormat = (value: string) => string;

function decimal(text: string, options: Intl.NumberFormatOptions): string {
  const formatter = numberFormatter({ maximumFractionDigits: 20, ...options });
  return (formatter.format as unknown as DecimalFormat)(text);
}

function money(amount: string, currency: string | null | undefined): string {
  if (currency) {
    return decimal(amount, { style: "currency", currency, minimumFractionDigits: 2 });
  }
  const negative = amount.startsWith("-");
  const digits = decimal(negative ? amount.slice(1) : amount, { minimumFractionDigits: 2 });
  return negative ? `-$${digits}` : `$${digits}`;
}

// A date has no zone and a date-time keeps its own offset, so both are formatted as the wall
// clock they spell, in UTC, never shifted into the reader's zone.
function date(value: string): string {
  const [year = 0, month = 1, day = 1] = value.split("-").map(Number);
  return dateTimeFormatter({ dateStyle: "medium", timeZone: "UTC" }).format(
    Date.UTC(year, month - 1, day),
  );
}

function show(value: TariffValue, t: TranslateFn, options: TariffValueFormatOptions): string {
  switch (value.kind) {
    case "none":
      return t("None");
    case "bool":
      return value.value ? t("Yes") : t("No");
    case "text":
      return value.value === "" ? t("Empty") : value.value;
    case "number":
      return decimal(value.value, {});
    case "quantity": {
      const { unit, unitText, amount } = value;
      if (isMoneyUnit(unit)) {
        const per = unit.denominator.map((part) => part.symbol).join("/");
        const shown = money(amount, options.currency);
        return per ? `${shown}/${per}` : shown;
      }
      if (unitText === "%") return `${decimal(amount, {})}%`;
      return `${decimal(amount, {})} ${unitText}`;
    }
    case "date":
      return date(value.value);
    case "dateTime": {
      const [year = 0, month = 1, day = 1] = value.date.split("-").map(Number);
      const wall = Date.UTC(year, month - 1, day, value.hour, value.minute, value.second);
      const text = dateTimeFormatter({
        dateStyle: "medium",
        timeStyle: "short",
        timeZone: "UTC",
      }).format(wall);
      return `${text} UTC${value.offset}`;
    }
    case "list":
      return t("{0, plural, one {# item} other {# items}}", value.items.length);
    case "record":
      return `{ ${value.fields
        .map((field) => `${field.name}: ${show(field.value, t, options)}`)
        .join(", ")} }`;
  }
}

/**
 * A wire value as a person reads it. A value that is not a wire value at all (a caller bug) is
 * shown as its JSON rather than throwing in the middle of a render.
 */
export function formatTariffValue(
  wire: TariffWireValue | undefined,
  t: TranslateFn,
  options: TariffValueFormatOptions = {},
): string {
  if (wire === undefined) return t("None");
  try {
    return show(decodeTariffValue(wire), t, options);
  } catch (error) {
    if (error instanceof TariffWireError) return JSON.stringify(wire);
    throw error;
  }
}

/** Money from its decimal text, as `formatTariffValue` writes a `$` quantity. */
export function formatTariffMoney(amount: string, options: TariffValueFormatOptions = {}): string {
  return money(amount, options.currency);
}
