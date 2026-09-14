import { describe, expect, it } from "vitest";
import { invoiceLineSchema, invoiceSchema } from "@trenova/shared/types/invoice";

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

/**
 * `domain/invoice/invoice.go` serialises what an invoice covers on every REST
 * payload: `scope`, the `invoiceRunId` that billed a statement, the period a
 * consolidated invoice states, and `shipmentCount`. The parse boundary strips
 * unknown keys, so without these fields a consolidated invoice reaches the UI
 * indistinguishable from a single-shipment one.
 */
describe("invoiceSchema scope", () => {
  const base = {
    id: "inv_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    billingQueueItemId: "bqi_1",
    customerId: "cus_1",
    number: "INV-1",
    billType: "Invoice",
    status: "Draft",
    paymentTerm: "Net30",
    currencyCode: "USD",
    invoiceDate: 1_788_000_000,
    billToName: "GlobalTrade Imports",
    subtotalAmount: "5000",
    otherAmount: "0",
    totalAmount: "5000",
    appliedAmount: "0",
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    version: 1,
    createdAt: 1_788_000_000,
    updatedAt: 1_788_000_000,
  };

  it("keeps what a consolidated invoice covers", () => {
    const parsed = invoiceSchema.parse({
      ...base,
      shipmentId: "",
      scope: "Consolidated",
      invoiceRunId: "invrun_1",
      periodStart: 1_788_235_200,
      periodEnd: 1_789_325_147,
      shipmentCount: 12,
    });

    expect(parsed.scope).toBe("Consolidated");
    expect(parsed.invoiceRunId).toBe("invrun_1");
    expect(parsed.periodStart).toBe(1_788_235_200);
    expect(parsed.periodEnd).toBe(1_789_325_147);
    expect(parsed.shipmentCount).toBe(12);
    expect(parsed.shipmentId).toBeNull();
  });

  it("reads the unset run and period of a single-shipment invoice as null", () => {
    const parsed = invoiceSchema.parse({
      ...base,
      shipmentId: "shp_1",
      scope: "Shipment",
      invoiceRunId: "",
      periodStart: null,
      periodEnd: null,
      shipmentCount: 1,
    });

    expect(parsed.scope).toBe("Shipment");
    expect(parsed.invoiceRunId).toBeNull();
    expect(parsed.periodStart).toBeNull();
    expect(parsed.periodEnd).toBeNull();
  });

  it("treats a payload from before scope existed as a single-shipment invoice", () => {
    const parsed = invoiceSchema.parse(base);

    expect(parsed.scope).toBe("Shipment");
    expect(parsed.shipmentCount).toBe(0);
  });

  it("rejects a scope the server does not define", () => {
    expect(() => invoiceSchema.parse({ ...base, scope: "Statement" })).toThrow();
  });
});

describe("invoiceLineSchema charge detail", () => {
  const base = {
    id: "invl_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    invoiceId: "inv_1",
    lineNumber: 2,
    type: "Accessorial",
    description: "Detention",
    quantity: "2",
    unitPrice: "75",
    amount: "150",
  };

  it("keeps what the charge is and how it was calculated", () => {
    const parsed = invoiceLineSchema.parse({
      ...base,
      accessorialChargeId: "acc_1",
      chargeCode: "DET",
      chargeMethod: "PerUnit",
      rateUnit: "Hour",
      rate: "75",
      rateBasisAmount: null,
      formulaTemplateName: "",
    });

    expect(parsed.accessorialChargeId).toBe("acc_1");
    expect(parsed.chargeCode).toBe("DET");
    expect(parsed.chargeMethod).toBe("PerUnit");
    expect(parsed.rateUnit).toBe("Hour");
    expect(parsed.rate).toBe(75);
    expect(parsed.rateBasisAmount).toBeNull();
    expect(parsed.formulaTemplateName).toBeNull();
  });

  it("reads the empty values Go sends for a line written before the detail existed", () => {
    const parsed = invoiceLineSchema.parse({
      ...base,
      accessorialChargeId: "",
      chargeCode: "",
      chargeMethod: "",
      rateUnit: "",
      rate: null,
      rateBasisAmount: null,
      formulaTemplateName: "",
    });

    expect(parsed.chargeMethod).toBeNull();
    expect(parsed.rateUnit).toBeNull();
    expect(parsed.rate).toBeNull();
    expect(parsed.chargeCode).toBeNull();
  });

  it("accepts a line with the detail fields absent", () => {
    const parsed = invoiceLineSchema.parse(base);

    expect(parsed.chargeMethod).toBeNull();
    expect(parsed.rate ?? null).toBeNull();
  });

  it("rejects a charge method the server does not define", () => {
    expect(() => invoiceLineSchema.parse({ ...base, chargeMethod: "Tiered" })).toThrow();
  });
});
