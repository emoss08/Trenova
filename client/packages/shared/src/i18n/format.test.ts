import { setLocale } from "@trenova/shared/i18n/runtime";
import { afterEach, describe, expect, it } from "vitest";
import {
  dateTimeFormatter,
  formatList,
  formatNumber,
  formatOrdinal,
  formatRelativeTime,
  numberFormatter,
} from "./format";

const INSTANT = new Date(Date.UTC(2026, 2, 3, 14, 5, 9));

describe("cached Intl formatters", () => {
  it("hands back the same number formatter for the same locale and options", () => {
    const options = { style: "currency", currency: "USD" } as const;
    expect(numberFormatter({ ...options }, "en-US")).toBe(numberFormatter({ ...options }, "en-US"));
  });

  it("keeps formatters apart by locale, by options and by currency", () => {
    const usd = numberFormatter({ style: "currency", currency: "USD" }, "en-US");
    expect(numberFormatter({ style: "currency", currency: "USD" }, "es-419")).not.toBe(usd);
    expect(numberFormatter({ style: "currency", currency: "EUR" }, "en-US")).not.toBe(usd);
    expect(numberFormatter({ style: "percent" }, "en-US")).not.toBe(usd);
    expect(numberFormatter(undefined, "en-US")).not.toBe(usd);
  });

  it("formats exactly as a fresh Intl formatter would", () => {
    const money = { style: "currency", currency: "USD", minimumFractionDigits: 2 } as const;
    expect(numberFormatter(money, "es-419").format(1234.5)).toBe(
      new Intl.NumberFormat("es-419", money).format(1234.5),
    );
    const zoned = {
      timeZone: "America/Chicago",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    } as const;
    expect(dateTimeFormatter(zoned, "en-US").format(INSTANT)).toBe(
      new Intl.DateTimeFormat("en-US", zoned).format(INSTANT),
    );
  });

  it("tells time zones apart, so one zone's formatter never answers for another", () => {
    const chicago = dateTimeFormatter({ timeZone: "America/Chicago", hour: "numeric" }, "en-US");
    const tokyo = dateTimeFormatter({ timeZone: "Asia/Tokyo", hour: "numeric" }, "en-US");
    expect(chicago).not.toBe(tokyo);
    expect(chicago.format(INSTANT)).not.toBe(tokyo.format(INSTANT));
  });

  it("treats an undefined option the way Intl does, as absent", () => {
    expect(numberFormatter({ maximumFractionDigits: undefined }, "en-US")).toBe(
      numberFormatter({}, "en-US"),
    );
  });

  it("keeps a bounded number of formatters and still formats after evicting", () => {
    for (let digits = 0; digits <= 20; digits += 1) {
      for (const currency of [
        "USD",
        "EUR",
        "GBP",
        "JPY",
        "CAD",
        "MXN",
        "CNY",
        "TWD",
        "AUD",
        "CHF",
        "SEK",
        "NOK",
        "INR",
      ]) {
        numberFormatter({ style: "currency", currency, maximumFractionDigits: digits }, "en-US");
      }
    }
    const first = numberFormatter(
      { style: "currency", currency: "USD", maximumFractionDigits: 0 },
      "en-US",
    );
    expect(first.format(5)).toBe("$5");
  });

  it("keeps the active-locale helpers' output", () => {
    expect(formatNumber(1234.5)).toBe(new Intl.NumberFormat("en-US").format(1234.5));
    expect(formatList(["a", "b", "c"])).toBe(
      new Intl.ListFormat("en-US", { style: "long", type: "conjunction" }).format(["a", "b", "c"]),
    );
    expect(formatRelativeTime(-7200)).toBe(
      new Intl.RelativeTimeFormat("en-US", { numeric: "auto" }).format(-2, "hour"),
    );
  });
});

describe("formatRelativeTime", () => {
  afterEach(async () => {
    await setLocale("en");
  });

  it("writes the compact form a dense row uses, in the reader's language", async () => {
    expect(formatRelativeTime(-31, "narrow")).toBe("31s ago");
    expect(formatRelativeTime(-3 * 86_400, "narrow")).toBe("3d ago");
    expect(formatRelativeTime(-3 * 86_400)).toBe("3 days ago");

    await setLocale("es");
    expect(formatRelativeTime(-3 * 86_400, "narrow")).toBe("hace 3 días");
  });
});

describe("formatOrdinal", () => {
  afterEach(async () => {
    await setLocale("en");
  });

  it("gives English its suffix, the awkward teens included", () => {
    expect([1, 2, 3, 4, 11, 12, 13, 21, 22, 23, 101, 111].map(formatOrdinal)).toEqual([
      "1st",
      "2nd",
      "3rd",
      "4th",
      "11th",
      "12th",
      "13th",
      "21st",
      "22nd",
      "23rd",
      "101st",
      "111th",
    ]);
  });

  it("leaves the number plain where the sentence carries the marker", async () => {
    await setLocale("es");
    expect(formatOrdinal(15)).toBe("15");

    await setLocale("zh-CN");
    expect(formatOrdinal(3)).toBe("3");
  });
});
