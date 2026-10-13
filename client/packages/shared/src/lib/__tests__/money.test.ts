import { describe, expect, it } from "vitest";
import { parseAmountToMinor } from "../money";

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
