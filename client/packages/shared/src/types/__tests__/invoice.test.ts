import { describe, expect, it } from "vitest";
import { invoiceLineSchema } from "@trenova/shared/types/invoice";

/**
 * The Go domain stamps `shipment_id`, `shipment_pro_number` and `shipment_bol`
 * onto every line of a grouped invoice (`domain/invoice/invoice.go`), and the
 * REST payload carries them. The parse boundary has to keep them, or a
 * consolidated invoice arrives in the UI with no way to tell which shipment a
 * charge belongs to.
 */
describe("invoiceLineSchema", () => {
  const base = {
    id: "invl_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    lineNumber: 1,
    type: "Freight",
    description: "Linehaul",
    quantity: "1",
    unitPrice: "100.00",
    amount: "100.00",
  };

  it("keeps the per-line shipment attribution", () => {
    const parsed = invoiceLineSchema.parse({
      ...base,
      shipmentId: "shp_1",
      shipmentProNumber: "PRO-1",
      shipmentBol: "BOL-1",
    });

    expect(parsed.shipmentId).toBe("shp_1");
    expect(parsed.shipmentProNumber).toBe("PRO-1");
    expect(parsed.shipmentBol).toBe("BOL-1");
  });

  it("normalizes the empty strings Go sends for unset nullzero columns", () => {
    // These columns are `nullzero` with a plain Go string, so an unset value
    // serialises as "" rather than null.
    const parsed = invoiceLineSchema.parse({
      ...base,
      shipmentId: "",
      shipmentProNumber: "",
      shipmentBol: "",
    });

    expect(parsed.shipmentId).toBeNull();
    expect(parsed.shipmentProNumber).toBeNull();
    expect(parsed.shipmentBol).toBeNull();
  });

  it("accepts a line with the attribution fields absent entirely", () => {
    const parsed = invoiceLineSchema.parse(base);

    expect(parsed.shipmentId ?? null).toBeNull();
    expect(parsed.shipmentProNumber ?? null).toBeNull();
  });

  it("accepts explicit nulls", () => {
    const parsed = invoiceLineSchema.parse({
      ...base,
      shipmentId: null,
      shipmentProNumber: null,
      shipmentBol: null,
    });

    expect(parsed.shipmentId).toBeNull();
  });
});
