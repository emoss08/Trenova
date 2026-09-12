import type { OpenStatement, StatementShipment } from "@trenova/shared/types/statement";
import { describe, expect, it } from "vitest";
import {
  billsInLabel,
  cadenceLabel,
  periodRange,
  splitLabel,
  standaloneShipmentCount,
} from "../billing-schedule";

function shipmentRow(orderId: string | null): StatementShipment {
  return {
    billingQueueItemId: `bqi_${Math.random()}`,
    shipmentId: "shp_1",
    orderId,
    proNumber: "PRO-1",
    bol: null,
    poNumber: null,
    orderNumber: null,
    serviceDate: null,
    amount: 100,
  };
}

function statementWith(
  splitBy: OpenStatement["splitBy"],
  groups: StatementShipment[][],
): Pick<OpenStatement, "splitBy" | "groups"> {
  return {
    splitBy,
    groups: groups.map((shipments, index) => ({
      key: `g${index}`,
      label: `Group ${index}`,
      shipmentCount: shipments.length,
      totalAmount: 100 * shipments.length,
      belowMinimum: false,
      shipments,
    })),
  };
}

// Splitting by order is the customer's configuration and the system honours it.
// This only measures how many shipments that setting is billing on their own, so
// the statement can say so without ever refusing to bill.
describe("standaloneShipmentCount", () => {
  it("counts the shipments without an order under the order split", () => {
    const statement = statementWith("CustomerAndOrder", [
      [shipmentRow(null)],
      [shipmentRow(null)],
      [shipmentRow("ord_1"), shipmentRow("ord_1")],
    ]);

    expect(standaloneShipmentCount(statement)).toBe(2);
  });

  // Under any other split a missing order changes nothing about the invoices, so
  // there is nothing to point out.
  it("is zero for every other split", () => {
    const statement = statementWith("Customer", [[shipmentRow(null), shipmentRow(null)]]);

    expect(standaloneShipmentCount(statement)).toBe(0);
  });

  it("treats an empty order id as no order", () => {
    const statement = statementWith("CustomerAndOrder", [[shipmentRow("")], [shipmentRow("")]]);

    expect(standaloneShipmentCount(statement)).toBe(2);
  });

  // The list read omits members, so there is nothing to count and no reason to
  // guess — the detail read, which carries them, is where the note appears.
  it("is zero when the members were not loaded", () => {
    expect(
      standaloneShipmentCount({
        splitBy: "CustomerAndOrder",
        groups: [
          {
            key: "g",
            label: "g",
            shipmentCount: 3,
            totalAmount: 300,
            belowMinimum: false,
            shipments: null,
          },
        ],
      }),
    ).toBe(0);
    expect(standaloneShipmentCount({ splitBy: "CustomerAndOrder", groups: null })).toBe(0);
  });
});

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
