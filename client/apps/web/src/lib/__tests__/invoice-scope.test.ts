import { describe, expect, it } from "vitest";
import { periodRange } from "@/lib/billing-schedule";
import { invoiceBillingPeriod, invoiceBillsSingleShipment } from "@/lib/invoice-scope";

describe("invoiceBillsSingleShipment", () => {
  it.each([
    ["Shipment", true],
    ["Adjustment", true],
    ["Order", false],
    ["Consolidated", false],
  ] as const)("%s -> %s", (scope, expected) => {
    expect(invoiceBillsSingleShipment(scope)).toBe(expected);
  });
});

describe("invoiceBillingPeriod", () => {
  const start = 1_788_235_200;
  const end = 1_789_325_147;

  it("formats the period a consolidated invoice states", () => {
    expect(
      invoiceBillingPeriod({ scope: "Consolidated", periodStart: start, periodEnd: end }),
    ).toBe(periodRange(start, end));
  });

  it("has no period when a consolidated invoice is missing either bound", () => {
    expect(
      invoiceBillingPeriod({ scope: "Consolidated", periodStart: start, periodEnd: null }),
    ).toBeNull();
    expect(
      invoiceBillingPeriod({ scope: "Consolidated", periodStart: null, periodEnd: end }),
    ).toBeNull();
    expect(invoiceBillingPeriod({ scope: "Consolidated" })).toBeNull();
  });

  it("ignores period fields on a scope that does not state one", () => {
    expect(invoiceBillingPeriod({ scope: "Order", periodStart: start, periodEnd: end })).toBeNull();
    expect(
      invoiceBillingPeriod({ scope: "Shipment", periodStart: start, periodEnd: end }),
    ).toBeNull();
  });
});
