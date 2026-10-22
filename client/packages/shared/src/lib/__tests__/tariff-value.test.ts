import { translate } from "@trenova/shared/i18n/runtime";
import { formatTariffMoney, formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { describe, expect, it } from "vitest";

const t = translate;

describe("formatTariffValue", () => {
  it("writes money, rates, quantities and percents as the reference writes them", () => {
    expect(formatTariffValue({ $q: "2.35", u: "$/mi" }, t)).toBe("$2.35/mi");
    expect(formatTariffValue({ $q: "18.5", u: "$/cwt" }, t)).toBe("$18.50/cwt");
    expect(formatTariffValue({ $q: "1200", u: "lb" }, t)).toBe("1,200 lb");
    expect(formatTariffValue({ $q: "18", u: "%" }, t)).toBe("18%");
    expect(formatTariffValue({ $q: "1908.2", u: "$" }, t)).toBe("$1,908.20");
    expect(formatTariffValue({ $q: "-75", u: "$" }, t)).toBe("-$75.00");
    expect(formatTariffValue({ $q: "-5", u: "°F" }, t)).toBe("-5 °F");
  });

  it("keeps every digit a rate was written with", () => {
    expect(formatTariffValue({ $q: "0.0425", u: "$/lb" }, t)).toBe("$0.0425/lb");
    expect(formatTariffValue({ $n: "12345678901234567890.5" }, t)).toBe(
      "12,345,678,901,234,567,890.5",
    );
  });

  it("shows the rating currency's symbol when it is known", () => {
    expect(formatTariffValue({ $q: "450", u: "$" }, t, { currency: "CAD" })).toBe("CA$450.00");
    expect(formatTariffValue({ $q: "2.35", u: "$/km" }, t, { currency: "CAD" })).toBe("CA$2.35/km");
    expect(formatTariffMoney("12.5", { currency: "USD" })).toBe("$12.50");
  });

  it("writes dates as the wall clock they spell, with a date-time's offset", () => {
    expect(formatTariffValue({ $date: "2026-10-22" }, t)).toBe("Oct 22, 2026");
    expect(formatTariffValue({ $date: "2026-10-22T08:00:00-05:00" }, t)).toBe(
      "Oct 22, 2026, 8:00 AM UTC-05:00",
    );
  });

  it("writes lists by their size, records field by field, and the scalars in words", () => {
    expect(formatTariffValue([], t)).toBe("0 items");
    expect(formatTariffValue(["752"], t)).toBe("1 item");
    expect(
      formatTariffValue({ from: { $q: "500", u: "lb" }, rate: { $q: "16", u: "$/cwt" } }, t),
    ).toBe("{ from: 500 lb, rate: $16.00/cwt }");
    expect(formatTariffValue(true, t)).toBe("Yes");
    expect(formatTariffValue(null, t)).toBe("None");
    expect(formatTariffValue(undefined, t)).toBe("None");
    expect(formatTariffValue("Delivery", t)).toBe("Delivery");
    expect(formatTariffValue("", t)).toBe("Empty");
  });

  it("shows a value that is not a wire value as its JSON instead of failing the render", () => {
    expect(formatTariffValue({ $q: "x", u: "lb" }, t)).toBe('{"$q":"x","u":"lb"}');
  });
});
