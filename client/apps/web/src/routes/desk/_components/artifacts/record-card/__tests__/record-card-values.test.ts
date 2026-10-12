import { describe, expect, it } from "vitest";
import { cardDate, cardMonth, minorMoney, money } from "../record-card-values";

describe("minorMoney", () => {
  it("writes an amount counted in cents as the card writes any amount", () => {
    expect(minorMoney(4_000_050)).toBe(money(40_000.5));
    expect(minorMoney(0)).toBe(money(0));
    expect(minorMoney(150_00, "CAD")).toBe(money(150, "CAD"));
  });

  it("writes a dash for an amount that is not there", () => {
    expect(minorMoney(null)).toBe("—");
    expect(minorMoney(undefined)).toBe("—");
  });
});

describe("cardDate and cardMonth", () => {
  const seconds = Date.UTC(2027, 2, 31, 12) / 1000;

  it("carry the year, for a date that may fall in another", () => {
    expect(cardDate(seconds)).toContain("2027");
    expect(cardMonth(seconds)).toContain("2027");
  });

  it("write nothing for a moment that is not there", () => {
    expect(cardDate(null)).toBe("");
    expect(cardMonth(0)).toBe("");
  });
});
