import { describe, expect, it } from "vitest";
import {
  addDecimalStrings,
  compareDecimalStrings,
  decimalPattern,
  decimalString,
  formatDecimalString,
  multiplyDecimalStrings,
  nonNegativeDecimalString,
  optionalNonNegativeDecimalString,
  optionalDecimalString,
  positiveDecimalString,
} from "@trenova/shared/types/decimal";

// Money and quantities travel as strings so the client never rounds what the
// server stored. Every helper here works on scaled BigInts, so "0.1" + "0.2"
// is "0.30" and never 0.30000000000000004. Invalid input is a programming
// error, not a user error: the zod schemas reject it before it reaches the
// arithmetic, so the arithmetic throws instead of guessing.

describe("decimalPattern", () => {
  it("allows up to the given number of fraction digits", () => {
    const twoPlaces = decimalPattern(2);
    expect(twoPlaces.test("12")).toBe(true);
    expect(twoPlaces.test("12.5")).toBe(true);
    expect(twoPlaces.test("12.50")).toBe(true);
    expect(twoPlaces.test("-12.50")).toBe(true);
    expect(twoPlaces.test("12.505")).toBe(false);
    expect(twoPlaces.test("12.")).toBe(false);
    expect(twoPlaces.test(".5")).toBe(false);
    expect(twoPlaces.test("1e3")).toBe(false);
    expect(twoPlaces.test("")).toBe(false);
  });

  it("widens with the scale", () => {
    expect(decimalPattern(4).test("3.8990")).toBe(true);
    expect(decimalPattern(4).test("3.89901")).toBe(false);
    expect(decimalPattern(3).test("124.350")).toBe(true);
  });

  it("only accepts integers at scale zero", () => {
    const integers = decimalPattern(0);
    expect(integers.test("42")).toBe(true);
    expect(integers.test("-42")).toBe(true);
    expect(integers.test("42.0")).toBe(false);
    expect(integers.test("42.")).toBe(false);
  });
});

describe("decimalString schemas", () => {
  it("trims and validates against the scale", () => {
    const schema = decimalString(2, "Enter an amount");
    expect(schema.parse("  12.50 ")).toBe("12.50");
    const failure = schema.safeParse("12.505");
    expect(failure.success).toBe(false);
    expect(failure.error?.issues[0]?.message).toBe("Enter an amount");
  });

  it("positiveDecimalString rejects zero and negatives with the same message", () => {
    const schema = positiveDecimalString(3, "Enter gallons above zero");
    expect(schema.parse("0.001")).toBe("0.001");
    expect(schema.safeParse("0").success).toBe(false);
    expect(schema.safeParse("0.000").success).toBe(false);
    expect(schema.safeParse("-1").success).toBe(false);
    expect(schema.safeParse("-1").error?.issues[0]?.message).toBe("Enter gallons above zero");
  });

  it("nonNegativeDecimalString accepts zero but not negatives", () => {
    const schema = nonNegativeDecimalString(2, "Tax cannot be negative");
    expect(schema.parse("0")).toBe("0");
    expect(schema.parse("0.00")).toBe("0.00");
    expect(schema.safeParse("-0.01").success).toBe(false);
    expect(schema.safeParse("-0.01").error?.issues[0]?.message).toBe("Tax cannot be negative");
  });

  it("optionalDecimalString turns blanks into null and keeps the scale check", () => {
    const schema = optionalDecimalString(2, "Enter a price");
    expect(schema.parse(null)).toBeNull();
    expect(schema.parse("")).toBeNull();
    expect(schema.parse("   ")).toBeNull();
    expect(schema.parse(" 4.25 ")).toBe("4.25");
    expect(schema.safeParse("4.255").success).toBe(false);
  });
});

describe("multiplyDecimalStrings", () => {
  it("multiplies gallons by unit price and rounds to the money scale", () => {
    expect(multiplyDecimalStrings("124.350", "3.899", 2)).toBe("484.84");
  });

  it("rounds half away from zero, not to even", () => {
    expect(multiplyDecimalStrings("1.005", "1", 2)).toBe("1.01");
    expect(multiplyDecimalStrings("-1.005", "1", 2)).toBe("-1.01");
    expect(multiplyDecimalStrings("2.5", "1", 0)).toBe("3");
    expect(multiplyDecimalStrings("-2.5", "1", 0)).toBe("-3");
  });

  it("keeps the sign straight for credits", () => {
    expect(multiplyDecimalStrings("-10.5", "2", 2)).toBe("-21.00");
    expect(multiplyDecimalStrings("-10.5", "-2", 2)).toBe("21.00");
  });

  it("never prints a negative zero", () => {
    expect(multiplyDecimalStrings("-0.001", "1", 2)).toBe("0.00");
  });

  it("pads short inputs out to the requested scale", () => {
    expect(multiplyDecimalStrings("3", "4", 4)).toBe("12.0000");
  });

  it("throws on anything that is not a plain decimal string", () => {
    expect(() => multiplyDecimalStrings("", "1", 2)).toThrow();
    expect(() => multiplyDecimalStrings("abc", "1", 2)).toThrow();
    expect(() => multiplyDecimalStrings("1e2", "1", 2)).toThrow();
    expect(() => multiplyDecimalStrings("1", "1", -1)).toThrow();
  });
});

describe("addDecimalStrings", () => {
  it("sums without float drift", () => {
    expect(addDecimalStrings(["0.1", "0.2"], 2)).toBe("0.30");
    expect(addDecimalStrings(["0.1", "0.2", "0.3"], 2)).toBe("0.60");
  });

  it("nets credits against charges", () => {
    expect(addDecimalStrings(["100.00", "-25.50", "-80.00"], 2)).toBe("-5.50");
  });

  it("rounds the total, not each line", () => {
    expect(addDecimalStrings(["0.004", "0.004"], 2)).toBe("0.01");
  });

  it("returns a zero at the requested scale for no lines", () => {
    expect(addDecimalStrings([], 2)).toBe("0.00");
    expect(addDecimalStrings([], 0)).toBe("0");
  });

  it("throws when any line is invalid", () => {
    expect(() => addDecimalStrings(["1.00", ""], 2)).toThrow();
  });
});

describe("formatDecimalString", () => {
  it("pads to a fixed scale for display", () => {
    expect(formatDecimalString("5", 2)).toBe("5.00");
    expect(formatDecimalString("5.5", 3)).toBe("5.500");
    expect(formatDecimalString("-5.5", 2)).toBe("-5.50");
  });

  it("rounds half away from zero when the value is finer than the scale", () => {
    expect(formatDecimalString("2.345", 2)).toBe("2.35");
    expect(formatDecimalString("-2.345", 2)).toBe("-2.35");
    expect(formatDecimalString("2.344", 2)).toBe("2.34");
  });

  it("drops a leading plus and surrounding whitespace", () => {
    expect(formatDecimalString(" 7.1 ", 2)).toBe("7.10");
  });

  it("throws on invalid input", () => {
    expect(() => formatDecimalString("seven", 2)).toThrow();
  });
});

describe("compareDecimalStrings", () => {
  it("compares numerically across different scales", () => {
    expect(compareDecimalStrings("1.10", "1.1")).toBe(0);
    expect(compareDecimalStrings("1.100", "1.2")).toBe(-1);
    expect(compareDecimalStrings("10", "9.999")).toBe(1);
    expect(compareDecimalStrings("-0.01", "0")).toBe(-1);
    expect(compareDecimalStrings("-0", "0.00")).toBe(0);
  });

  it("is not fooled by string ordering", () => {
    expect(compareDecimalStrings("9", "10")).toBe(-1);
    expect(compareDecimalStrings("-9", "-10")).toBe(1);
  });
});

describe("signed decimal schemas on input that is not a decimal", () => {
  it("report the message rather than throwing when the string is blank or malformed", () => {
    for (const value of ["", "   ", "abc", "1.2.3", "-"]) {
      const positive = positiveDecimalString(3, "Enter a positive quantity").safeParse(value);
      expect(positive.success, `positive ${JSON.stringify(value)}`).toBe(false);
      expect(positive.error?.issues[0]?.message).toBe("Enter a positive quantity");

      const nonNegative = nonNegativeDecimalString(2, "Enter an amount").safeParse(value);
      expect(nonNegative.success, `nonNegative ${JSON.stringify(value)}`).toBe(false);
      expect(nonNegative.error?.issues[0]?.message).toBe("Enter an amount");
    }
  });
});

describe("optionalNonNegativeDecimalString", () => {
  it("treats blank as null and keeps a well-formed non-negative value", () => {
    const schema = optionalNonNegativeDecimalString(4, "Enter a rate");
    expect(schema.parse("")).toBeNull();
    expect(schema.parse("  ")).toBeNull();
    expect(schema.parse(null)).toBeNull();
    expect(schema.parse("0.1100")).toBe("0.1100");
    expect(schema.parse("0")).toBe("0");
  });

  it("reports the message rather than throwing for malformed or negative input", () => {
    const schema = optionalNonNegativeDecimalString(4, "Enter a rate");
    for (const value of ["abc", "1.2.3", "-0.01", "0.12345"]) {
      const result = schema.safeParse(value);
      expect(result.success, JSON.stringify(value)).toBe(false);
      expect(result.error?.issues[0]?.message).toBe("Enter a rate");
    }
  });
});
