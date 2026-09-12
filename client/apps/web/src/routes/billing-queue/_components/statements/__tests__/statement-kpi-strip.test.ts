import type { OpenStatement } from "@trenova/shared/types/statement";
import { describe, expect, it } from "vitest";
import { summarizeStatements } from "../statement-kpi-strip";

const NOW = 1_774_000_000;
const DAY = 86_400;

function statement(overrides: Partial<OpenStatement> = {}): OpenStatement {
  return {
    customerId: "cus_1",
    customerName: "Acme Freight",
    customerCode: "ACME",
    customerStatus: "Active",
    cycle: "Monthly",
    billingCycleAnchorDay: 1,
    billingCycleTimezone: "America/Denver",
    periodStart: NOW - 10 * DAY,
    periodEnd: NOW + 10 * DAY,
    lastBilledPeriodEnd: null,
    shipmentCount: 4,
    invoiceCount: 1,
    totalAmount: 4200,
    currencyCode: "USD",
    splitBy: "Customer",
    sectionBy: "Shipment",
    detail: "Detailed",
    minimumAmount: null,
    autoBill: false,
    belowMinimum: false,
    heldCount: 0,
    heldAmount: 0,
    groups: [],
    ...overrides,
  };
}

describe("summarizeStatements", () => {
  it("adds up the statements with freight on them", () => {
    const metrics = summarizeStatements(
      [
        statement({ shipmentCount: 4, totalAmount: 4200 }),
        statement({ shipmentCount: 2, totalAmount: 800 }),
      ],
      NOW,
    );

    expect(metrics.open).toBe(2);
    expect(metrics.shipments).toBe(6);
    expect(metrics.value).toBe(5000);
  });

  // A customer with a quiet month is not an open statement. Counting them would
  // put every statement customer in the "accruing" tile forever.
  it("ignores statements with nothing accrued", () => {
    const metrics = summarizeStatements(
      [
        statement({ shipmentCount: 0, totalAmount: 0 }),
        statement({ shipmentCount: 3, totalAmount: 900 }),
      ],
      NOW,
    );

    expect(metrics.open).toBe(1);
    expect(metrics.shipments).toBe(3);
    expect(metrics.value).toBe(900);
  });

  it("counts only the statements closing inside a week", () => {
    const metrics = summarizeStatements(
      [
        statement({ periodEnd: NOW + 2 * DAY }),
        statement({ periodEnd: NOW + 20 * DAY }),
        statement({ periodEnd: NOW - DAY }),
      ],
      NOW,
    );

    expect(metrics.billingThisWeek).toBe(2);
  });

  it("counts the statements held under their customer's minimum", () => {
    const metrics = summarizeStatements(
      [statement({ belowMinimum: true }), statement({ belowMinimum: false })],
      NOW,
    );

    expect(metrics.held).toBe(1);
  });

  // Summing across currencies would produce a meaningless total, so the tile
  // takes the currency of the first statement that actually has freight rather
  // than whichever customer sorts first.
  it("takes its currency from the first statement with freight", () => {
    const metrics = summarizeStatements(
      [
        statement({ shipmentCount: 0, totalAmount: 0, currencyCode: "CAD" }),
        statement({ shipmentCount: 2, totalAmount: 500, currencyCode: "USD" }),
      ],
      NOW,
    );

    expect(metrics.currency).toBe("USD");
  });

  it("reports zeroes rather than NaN for an empty list", () => {
    const metrics = summarizeStatements([], NOW);

    expect(metrics).toMatchObject({
      open: 0,
      shipments: 0,
      value: 0,
      billingThisWeek: 0,
      held: 0,
    });
  });
});
