import { describe, expect, it } from "vitest";
import {
  amountSeparators,
  amountTextToMinor,
  currencySymbol,
  formatMinor,
  formatMinorAmount,
  isMinorAmountField,
  minorToAmountText,
  parseAmountToMinor,
  withoutMinorSuffix,
} from "../money";

describe("parseAmountToMinor", () => {
  it("reads whole and fractional amounts exactly", () => {
    expect(parseAmountToMinor("12")).toBe(1200);
    expect(parseAmountToMinor("12.5")).toBe(1250);
    expect(parseAmountToMinor(" 0.29 ")).toBe(29);
    expect(parseAmountToMinor("1234.56")).toBe(123456);
  });

  it("refuses what it cannot read without changing the money", () => {
    expect(parseAmountToMinor("10.005")).toBeNull();
    expect(parseAmountToMinor("1,234.56")).toBeNull();
    expect(parseAmountToMinor("-5")).toBeNull();
    expect(parseAmountToMinor("")).toBeNull();
    expect(parseAmountToMinor("abc")).toBeNull();
  });
});

describe("formatMinor", () => {
  it("renders minor units as money, dividing only at the end", () => {
    expect(formatMinor(12_000)).toBe("$120.00");
    expect(formatMinor(0)).toBe("$0.00");
    expect(formatMinor(9_048_000)).toBe("$90,480.00");
    expect(formatMinor(1_234_567)).toBe("$12,345.67");
  });

  it("writes the record's currency, and dollars when it names none", () => {
    expect(formatMinor(258_222, "EUR")).toBe("€2,582.22");
    expect(formatMinor(258_222, "")).toBe("$2,582.22");
    expect(formatMinor(258_222, null)).toBe("$2,582.22");
  });
});

describe("formatMinorAmount", () => {
  it("writes minor units as an amount with two decimals and no currency", () => {
    expect(formatMinorAmount(258_222)).toBe("2,582.22");
    expect(formatMinorAmount(0)).toBe("0.00");
    expect(formatMinorAmount(-1_050)).toBe("-10.50");
  });
});

describe("minor-unit fields", () => {
  it("knows a field by its Minor suffix, at any depth", () => {
    expect(isMinorAmountField("balanceDueMinor")).toBe(true);
    expect(isMinorAmountField("lines.0.amountMinor")).toBe(true);
    expect(isMinorAmountField("minorityOwned")).toBe(false);
    expect(isMinorAmountField("Minor")).toBe(false);
    expect(isMinorAmountField("balanceDue")).toBe(false);
    expect(isMinorAmountField(undefined)).toBe(false);
  });

  it("names a field without the suffix", () => {
    expect(withoutMinorSuffix("balanceDueMinor")).toBe("balanceDue");
    expect(withoutMinorSuffix("lines.0.amountMinor")).toBe("lines.0.amount");
    expect(withoutMinorSuffix("minorityOwned")).toBe("minorityOwned");
  });
});

describe("currencySymbol", () => {
  it("is the mark the currency is written with, reading US dollars when none is given", () => {
    expect(currencySymbol()).toBe("$");
    expect(currencySymbol("USD")).toBe("$");
    expect(currencySymbol("EUR")).toBe("€");
  });
});

describe("the text of an amount being typed", () => {
  it("is read by its digits, so no cent is lost to a float", () => {
    expect(amountTextToMinor("0.29")).toBe(29);
    expect(amountTextToMinor("1.15")).toBe(115);
    expect(amountTextToMinor("2582.22")).toBe(258_222);
    expect(amountTextToMinor("650")).toBe(65_000);
  });

  it("is read as far as it has been typed", () => {
    expect(amountTextToMinor("")).toBe(0);
    expect(amountTextToMinor(".")).toBe(0);
    expect(amountTextToMinor("-")).toBe(0);
    expect(amountTextToMinor("12.")).toBe(1_200);
    expect(amountTextToMinor("12.5")).toBe(1_250);
    expect(amountTextToMinor(".5")).toBe(50);
  });

  it("keeps its sign, and never writes a negative zero", () => {
    expect(amountTextToMinor("-10.50")).toBe(-1_050);
    expect(Object.is(amountTextToMinor("-0.00"), 0)).toBe(true);
  });

  it("drops digits past the cents rather than rounding money nobody typed", () => {
    expect(amountTextToMinor("1.239")).toBe(123);
  });

  it("is zero for an amount too large to hold exactly", () => {
    expect(amountTextToMinor("99999999999999999999")).toBe(0);
  });
});

describe("an amount as the text a field holds", () => {
  it("always carries two decimals", () => {
    expect(minorToAmountText(0)).toBe("0.00");
    expect(minorToAmountText(5)).toBe("0.05");
    expect(minorToAmountText(65_000)).toBe("650.00");
    expect(minorToAmountText(258_222)).toBe("2582.22");
  });

  it("carries its sign, cents included", () => {
    expect(minorToAmountText(-1_050)).toBe("-10.50");
    expect(minorToAmountText(-5)).toBe("-0.05");
  });

  it("reads back to the amount it was written from", () => {
    for (const minor of [0, 1, 29, 99, 100, 115, 258_222, -1, -1_050]) {
      expect(amountTextToMinor(minorToAmountText(minor))).toBe(minor);
    }
  });
});

describe("amountSeparators", () => {
  it("are the marks the reader's locale writes an amount with", () => {
    expect(amountSeparators()).toEqual({ decimal: ".", group: "," });
  });
});
