import { describe, expect, it } from "vitest";
import {
  dateWindowFromSearch,
  dateWindowFilter,
  dateWindowLabel,
  dayStart,
  parseDateWindow,
  serializeDateWindow,
  todayInUserCalendar,
} from "../billing-queue-date-window";

const t = ((message: string, ...args: unknown[]) => {
  const count = Number(args[0]);
  const plural = message.match(/one \{([^}]*)\} other \{([^}]*)\}/u);
  if (plural) {
    return (count === 1 ? plural[1]! : plural[2]!).replace("#", String(count));
  }
  return args.reduce<string>(
    (text, arg, index) => text.replace(`{${index}}`, String(arg)),
    message,
  );
}) as never;

describe("billing queue date windows", () => {
  it("keeps a window in the URL and reads it back", () => {
    expect(serializeDateWindow({ kind: "last", days: 7 })).toBe("last:7");
    expect(serializeDateWindow({ kind: "range", from: 1_793_000_000, to: 1_793_500_000 })).toBe(
      "range:1793000000:1793500000",
    );
    expect(parseDateWindow("last:30")).toEqual({ kind: "last", days: 30 });
    expect(parseDateWindow("range:1793000000:1793500000")).toEqual({
      kind: "range",
      from: 1_793_000_000,
      to: 1_793_500_000,
    });
  });

  it("reads anything it did not write as no window", () => {
    expect(parseDateWindow(null)).toBeNull();
    expect(parseDateWindow("last:0")).toBeNull();
    expect(parseDateWindow("last:week")).toBeNull();
    expect(parseDateWindow("range:1793500000:1793000000")).toBeNull();
    expect(parseDateWindow("since:1793000000")).toBeNull();
  });

  it("turns a window into the list filter on its field", () => {
    expect(dateWindowFilter("shipment.actualDeliveryDate", { kind: "last", days: 7 })).toEqual({
      field: "shipment.actualDeliveryDate",
      operator: "lastndays",
      value: 7,
    });
    expect(dateWindowFilter("createdAt", { kind: "range", from: 10, to: 20 })).toEqual({
      field: "createdAt",
      operator: "daterange",
      value: { from: 10, to: 20 },
    });
  });

  it("says a window as one phrase for its field", () => {
    expect(dateWindowLabel("delivered", { kind: "last", days: 7 }, t)).toBe(
      "Delivered in the last 7 days",
    );
    expect(dateWindowLabel("pickedUp", { kind: "last", days: 1 }, t)).toBe(
      "Picked up in the last day",
    );
    expect(
      dateWindowLabel("queued", { kind: "range", from: 1_793_000_000, to: 1_793_000_000 }, t),
    ).toMatch(/^Queued \S/u);
  });

  it("turns the days Nova read into the person's own window", () => {
    expect(dateWindowFromSearch({ lastDays: 7 })).toEqual({ kind: "last", days: 7 });
    const from = dayStart("2026-10-05");
    const to = dayStart("2026-10-11");
    expect(from).not.toBeNull();
    expect(to! - from!).toBe(6 * 86_400);
    expect(dateWindowFromSearch({ from: "2026-10-05", to: "2026-10-11" })).toEqual({
      kind: "range",
      from,
      to,
    });
    expect(dateWindowFromSearch({ from: "2026-10-11", to: "2026-10-05" })).toBeNull();
    expect(dateWindowFromSearch({ from: "Oct 19" })).toBeNull();
    expect(dateWindowFromSearch(null)).toBeNull();
  });

  it("names today in the person's calendar for Nova to count from", () => {
    expect(todayInUserCalendar()).toMatch(/^\d{4}-\d{2}-\d{2}$/u);
  });
});
