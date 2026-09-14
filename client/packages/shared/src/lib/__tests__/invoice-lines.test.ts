import { describe, expect, it } from "vitest";
import {
  ORDER_CHARGES_GROUP_KEY,
  chargeComposition,
  describeChargeCalculation,
  groupInvoiceLinesByShipment,
  hasUnitBreakdown,
} from "@trenova/shared/lib/invoice-lines";
import { translate } from "@trenova/shared/i18n/runtime";
import type { InvoiceLine } from "@trenova/shared/types/invoice";

function line(overrides: Partial<InvoiceLine> & Pick<InvoiceLine, "id" | "lineNumber">) {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    shipmentId: null,
    shipmentProNumber: null,
    shipmentBol: null,
    type: "Freight",
    description: "Linehaul",
    quantity: 1,
    unitPrice: 100,
    amount: 100,
    ...overrides,
  } as InvoiceLine;
}

describe("groupInvoiceLinesByShipment", () => {
  it("returns one group for a single-shipment invoice", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_1", shipmentProNumber: "PRO-1" }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].shipmentId).toBe("shp_1");
    expect(groups[0].proNumber).toBe("PRO-1");
    expect(groups[0].lines.map((l) => l.id)).toEqual(["l1", "l2"]);
  });

  it("keeps shipments in first-appearance order", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_b" }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_a" }),
      line({ id: "l3", lineNumber: 3, shipmentId: "shp_b" }),
    ]);

    expect(groups.map((g) => g.shipmentId)).toEqual(["shp_b", "shp_a"]);
  });

  it("orders lines by line number within a group", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l3", lineNumber: 30, shipmentId: "shp_1" }),
      line({ id: "l1", lineNumber: 10, shipmentId: "shp_1" }),
      line({ id: "l2", lineNumber: 20, shipmentId: "shp_1" }),
    ]);

    expect(groups[0].lines.map((l) => l.lineNumber)).toEqual([10, 20, 30]);
  });

  it("puts unattributed order-level charges in a trailing group", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "oc", lineNumber: 1, shipmentId: null, description: "Customs brokerage" }),
      line({ id: "l1", lineNumber: 2, shipmentId: "shp_1" }),
      line({ id: "l2", lineNumber: 3, shipmentId: "shp_2" }),
    ]);

    expect(groups).toHaveLength(3);
    expect(groups.at(-1)?.key).toBe(ORDER_CHARGES_GROUP_KEY);
    expect(groups.at(-1)?.shipmentId).toBeNull();
    expect(groups.at(-1)?.lines.map((l) => l.id)).toEqual(["oc"]);
    // Never merged into the first shipment.
    expect(groups[0].lines.map((l) => l.id)).toEqual(["l1"]);
  });

  it("treats an empty-string shipment id as unattributed", () => {
    // Go stamps these columns `nullzero` with a plain string, so an unset value
    // serialises as "" rather than null.
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "" as unknown as null }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0].key).toBe(ORDER_CHARGES_GROUP_KEY);
  });

  it("falls back to a later line for a leg identity the first line lacks", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", shipmentProNumber: "" as never }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_1", shipmentProNumber: "PRO-9" }),
    ]);

    expect(groups[0].proNumber).toBe("PRO-9");
  });

  it("sums subtotals from the lines, not from the invoice header", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", amount: 125.5 }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_1", amount: 74.5 }),
      line({ id: "l3", lineNumber: 3, shipmentId: "shp_2", amount: 300 }),
    ]);

    expect(groups[0].subtotal).toBe(200);
    expect(groups[1].subtotal).toBe(300);
  });

  it("treats a null amount as zero", () => {
    const groups = groupInvoiceLinesByShipment([
      line({ id: "l1", lineNumber: 1, shipmentId: "shp_1", amount: null }),
      line({ id: "l2", lineNumber: 2, shipmentId: "shp_1", amount: 50 }),
    ]);

    expect(groups[0].subtotal).toBe(50);
  });

  it("returns no groups for no lines", () => {
    expect(groupInvoiceLinesByShipment([])).toEqual([]);
  });
});

describe("hasUnitBreakdown", () => {
  it("is false for a flat charge whose one unit is the whole amount", () => {
    expect(hasUnitBreakdown(line({ id: "l1", lineNumber: 1 }))).toBe(false);
  });

  it("is true when more than one unit was billed", () => {
    expect(
      hasUnitBreakdown(line({ id: "l1", lineNumber: 1, quantity: 3, unitPrice: 50, amount: 150 })),
    ).toBe(true);
  });

  it("is true for a fractional quantity", () => {
    expect(
      hasUnitBreakdown(
        line({ id: "l1", lineNumber: 1, quantity: 0.5, unitPrice: 200, amount: 100 }),
      ),
    ).toBe(true);
  });

  it("is true when a single unit is priced differently from the amount", () => {
    expect(
      hasUnitBreakdown(
        line({ id: "l1", lineNumber: 1, quantity: 1, unitPrice: 2.5, amount: 1250 }),
      ),
    ).toBe(true);
  });

  it("ignores sub-cent unit price precision the amount rounds away", () => {
    // Unit prices carry four decimal places; amounts are money.
    expect(
      hasUnitBreakdown(
        line({ id: "l1", lineNumber: 1, quantity: 1, unitPrice: 100.0049, amount: 100 }),
      ),
    ).toBe(false);
  });

  it("treats a null quantity as no breakdown worth showing", () => {
    expect(
      hasUnitBreakdown(
        line({ id: "l1", lineNumber: 1, quantity: null, unitPrice: null, amount: 100 }),
      ),
    ).toBe(false);
  });
});

describe("chargeComposition", () => {
  it("splits freight and accessorials into shares of their sum", () => {
    expect(chargeComposition(750, 250)).toEqual({ freightShare: 0.75, accessorialShare: 0.25 });
  });

  it("gives freight the whole bar when there are no accessorials", () => {
    expect(chargeComposition(400, 0)).toEqual({ freightShare: 1, accessorialShare: 0 });
  });

  it("has nothing to draw when both are zero", () => {
    expect(chargeComposition(0, 0)).toBeNull();
  });

  it("has nothing to draw for a credit memo's negative amounts", () => {
    expect(chargeComposition(-400, -50)).toBeNull();
    expect(chargeComposition(400, -50)).toBeNull();
  });

  it("has nothing to draw when an amount is not a number", () => {
    expect(chargeComposition(Number.NaN, 50)).toBeNull();
  });
});

describe("describeChargeCalculation", () => {
  const accessorial = (overrides: Partial<InvoiceLine>) =>
    line({ id: "a", lineNumber: 2, type: "Accessorial", description: "Detention", ...overrides });

  it("states a per-unit charge's rate and the units it was billed for", () => {
    expect(
      describeChargeCalculation(
        accessorial({
          chargeMethod: "PerUnit",
          rateUnit: "Hour",
          rate: 75,
          quantity: 2,
          unitPrice: 75,
          amount: 150,
        }),
        "USD",
        translate,
      ),
    ).toBe("$75.00 × 2 hours");
  });

  it("uses the singular unit for one unit and a generic unit when none was recorded", () => {
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "PerUnit", rateUnit: "Mile", rate: 2.5, quantity: 1 }),
        "USD",
        translate,
      ),
    ).toBe("$2.50 × 1 mile");
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "PerUnit", rateUnit: null, rate: 10, quantity: 3 }),
        "USD",
        translate,
      ),
    ).toBe("$10.00 × 3 units");
  });

  it("states a percentage charge against the line haul it was taken from", () => {
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "Percentage", rate: 10, rateBasisAmount: 2450, amount: 245 }),
        "USD",
        translate,
      ),
    ).toBe("10% of line haul ($2,450.00)");
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "Percentage", rate: 12.5, rateBasisAmount: null }),
        "USD",
        translate,
      ),
    ).toBe("12.5% of line haul");
  });

  it("calls a single flat charge flat and shows a repeated one's multiple", () => {
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "Flat", rate: 150, quantity: 1, unitPrice: 150, amount: 150 }),
        "USD",
        translate,
      ),
    ).toBe("Flat rate");
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: "Flat", rate: 150, quantity: 2, unitPrice: 150, amount: 300 }),
        "USD",
        translate,
      ),
    ).toBe("$150.00 × 2");
  });

  it("falls back to quantity and unit price for lines written before the method was recorded", () => {
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: null, rate: null, quantity: 3, unitPrice: 50, amount: 150 }),
        "USD",
        translate,
      ),
    ).toBe("3 × $50.00");
    expect(
      describeChargeCalculation(
        accessorial({ chargeMethod: null, rate: null, quantity: 1, unitPrice: 150, amount: 150 }),
        "USD",
        translate,
      ),
    ).toBeNull();
  });

  it("does not describe freight as an accessorial calculation", () => {
    expect(
      describeChargeCalculation(
        line({
          id: "f",
          lineNumber: 1,
          chargeMethod: "Flat",
          rate: 3.5,
          amount: 2450,
          unitPrice: 2450,
        }),
        "USD",
        translate,
      ),
    ).toBeNull();
  });
});
