import { describe, expect, it } from "vitest";
import {
  ORDER_CHARGES_GROUP_KEY,
  groupInvoiceLinesByShipment,
} from "@trenova/shared/lib/invoice-lines";
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
