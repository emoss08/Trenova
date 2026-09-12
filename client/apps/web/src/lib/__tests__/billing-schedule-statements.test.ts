import { describe, expect, it } from "vitest";
import { billsInLabel, cadenceLabel, periodRange, splitLabel } from "../billing-schedule";

const NOW = 1_774_000_000;
const HOUR = 3_600;
const DAY = 86_400;

describe("billsInLabel", () => {
  it("counts whole days down", () => {
    expect(billsInLabel(NOW + 9 * DAY, NOW)).toBe("Bills in 9 days");
    expect(billsInLabel(NOW + 2 * DAY, NOW)).toBe("Bills in 2 days");
  });

  it("says tomorrow rather than 1 day", () => {
    expect(billsInLabel(NOW + DAY + HOUR, NOW)).toBe("Bills tomorrow");
  });

  it("drops to hours inside a day", () => {
    expect(billsInLabel(NOW + 5 * HOUR, NOW)).toBe("Bills in 5 hours");
  });

  it("collapses the last hour rather than counting minutes", () => {
    expect(billsInLabel(NOW + 20 * 60, NOW)).toBe("Bills within the hour");
  });

  // A boundary already past means the sweep has not caught up. Clamping it to
  // "Bills within the hour" would hide a stuck scheduler behind a countdown that
  // never finishes.
  it("says a passed boundary is due rather than clamping it", () => {
    expect(billsInLabel(NOW - DAY, NOW)).toBe("Due now");
    expect(billsInLabel(NOW, NOW)).toBe("Due now");
  });
});

describe("periodRange", () => {
  // The end is exclusive on the wire and is shown as the boundary the freight
  // stops at, not decremented by a day.
  it("shows both boundaries as given", () => {
    const start = Date.UTC(2026, 2, 1, 12) / 1000;
    const end = Date.UTC(2026, 3, 1, 12) / 1000;

    expect(periodRange(start, end)).toBe("Mar 1, 2026 – Apr 1, 2026");
  });
});

describe("cadenceLabel", () => {
  it("names every cycle in two words at most", () => {
    expect(cadenceLabel("Monthly")).toBe("Monthly");
    expect(cadenceLabel("BiWeekly")).toBe("Bi-weekly");
    expect(cadenceLabel("SemiMonthly")).toBe("Semi-monthly");
    expect(cadenceLabel("Immediate")).toBe("Per shipment");
  });
});

describe("splitLabel", () => {
  it("says how many invoices the period becomes", () => {
    expect(splitLabel("Customer")).toBe("One invoice");
    expect(splitLabel("CustomerAndPONumber")).toBe("One per PO");
    expect(splitLabel("CustomerAndDestination")).toBe("One per delivery");
  });
});
