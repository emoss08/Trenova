import { describe, expect, it } from "vitest";
import {
  cardDate,
  cardMomentAfter,
  cardMonth,
  cardTime,
  cityLine,
  minorMoney,
  money,
  waitText,
} from "../record-card-values";

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

describe("waitText", () => {
  it("reads minutes as hours and minutes, and never below none", () => {
    expect(waitText(96)).toBe("1h 36m");
    expect(waitText(80.4)).toBe("1h 20m");
    expect(waitText(0)).toBe("0m");
    expect(waitText(-5)).toBe("0m");
  });
});

describe("cityLine", () => {
  it("reads the city, state and postal code on one line", () => {
    expect(cityLine({ city: " Dallas ", postalCode: "75201", state: { abbreviation: "TX" } })).toBe(
      "Dallas, TX 75201",
    );
  });

  it("leaves out what is not on file, including a postal code the place does not carry", () => {
    expect(cityLine({ city: "Dallas", postalCode: "", state: null })).toBe("Dallas");
    expect(cityLine({ city: "Springdale", state: { abbreviation: "AR" } })).toBe("Springdale, AR");
    expect(cityLine({ city: " ", postalCode: null, state: { abbreviation: "AR" } })).toBe("AR");
  });
});

describe("cardMomentAfter", () => {
  const morning = new Date(2026, 9, 11, 7, 25).getTime() / 1000;

  it("gives the time alone for a moment on the same day as the one before it", () => {
    const later = new Date(2026, 9, 11, 9, 30).getTime() / 1000;
    expect(cardMomentAfter(morning, later)).toBe(cardTime(later));
    expect(cardMomentAfter(morning, later)).not.toMatch(/Oct/u);
  });

  it("gives the day too for a moment on another day", () => {
    const nextDay = new Date(2026, 9, 12, 0, 15).getTime() / 1000;
    expect(cardMomentAfter(morning, nextDay)).toMatch(/Oct 12/u);
  });
});
