import { setLocale } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { formatMinor, parseAmountToMinor } from "../money";

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
  it("renders minor units as money in the currency it is given", () => {
    expect(formatMinor(102_345, "USD")).toBe("$1,023.45");
    expect(formatMinor(5, "USD")).toBe("$0.05");
    expect(formatMinor(-2_500, "USD")).toBe("-$25.00");
    expect(formatMinor(102_345, "EUR")).toBe("€1,023.45");
    expect(formatMinor(102_345, "CAD")).toBe("CA$1,023.45");
  });

  it("reads US dollars when no currency is given", () => {
    expect(formatMinor(1_234_567)).toBe("$12,345.67");
  });

  it("writes the amount in the reader's language", async () => {
    await setLocale("es");
    try {
      expect(formatMinor(102_345, "USD")).toBe(
        (1_023.45).toLocaleString("es-419", { style: "currency", currency: "USD" }),
      );
    } finally {
      await setLocale("en");
    }
  });
});
