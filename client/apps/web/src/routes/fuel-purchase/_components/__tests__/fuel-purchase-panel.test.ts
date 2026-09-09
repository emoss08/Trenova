import type { FuelPurchaseRow } from "@/lib/graphql/fuel-purchase";
import { createFuelPurchaseFormSchema } from "@trenova/shared/types/fuel-purchase";
import { describe, expect, it } from "vitest";
import { buildFuelPurchaseDefaults, toFuelPurchaseInput } from "../fuel-purchase-panel";

const NOW = 1_760_000_000;

const row: FuelPurchaseRow = {
  id: "fpur_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  tractorId: "trk_118",
  workerId: "wrk_1",
  jurisdictionId: "ij_tx",
  fuelCardId: "fcrd_1",
  cardLastFour: "4821",
  purchasedAt: NOW - 86_400,
  vendor: "Love's #412",
  vendorCity: "Amarillo",
  fuelType: "Diesel",
  quantity: "120.500",
  quantityUnit: "Gallon",
  gallons: "120.500",
  unitPrice: "3.8990",
  totalAmount: "469.83",
  currencyCode: "USD",
  odometer: 412_113,
  transactionReference: "EFS-77812",
  source: "CardImport",
  importBatchId: "fpib_1",
  taxPaid: true,
  notes: null,
  createdById: "usr_1",
  version: 1,
  createdAt: 1,
  updatedAt: 2,
  tractor: { id: "trk_118", code: "118" },
  worker: { id: "wrk_1", wholeName: "Dana Ruiz", firstName: "Dana", lastName: "Ruiz" },
  jurisdiction: { id: "ij_tx", countryCode: "US", code: "TX", name: "Texas" },
  fuelCard: { id: "fcrd_1", provider: "EFS", lastFour: "4821", label: "Unit 118 card" },
};

const schema = createFuelPurchaseFormSchema(NOW);

describe("fuel purchase form mapping", () => {
  it("starts a new purchase now, in gallons of diesel, tax paid, priced in USD", () => {
    const defaults = buildFuelPurchaseDefaults(null, NOW);
    expect(defaults).toEqual({
      tractorId: "",
      workerId: null,
      jurisdictionId: "",
      purchasedAt: NOW,
      vendor: null,
      vendorCity: null,
      fuelType: "Diesel",
      quantity: "",
      quantityUnit: "Gallon",
      unitPrice: null,
      totalAmount: "",
      currencyCode: "USD",
      odometer: null,
      fuelCardId: null,
      cardLastFour: null,
      transactionReference: null,
      taxPaid: true,
      notes: null,
    });
  });

  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildFuelPurchaseDefaults(row, NOW);
    const parsed = schema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    expect(toFuelPurchaseInput(defaults)).toEqual({
      tractorId: "trk_118",
      workerId: "wrk_1",
      jurisdictionId: "ij_tx",
      purchasedAt: NOW - 86_400,
      vendor: "Love's #412",
      vendorCity: "Amarillo",
      fuelType: "Diesel",
      quantity: "120.500",
      quantityUnit: "Gallon",
      unitPrice: "3.8990",
      totalAmount: "469.83",
      currencyCode: "USD",
      odometer: 412_113,
      fuelCardId: "fcrd_1",
      cardLastFour: "4821",
      transactionReference: "EFS-77812",
      taxPaid: true,
      notes: null,
    });
  });

  it("sends null for cleared optional fields rather than empty strings", () => {
    const input = toFuelPurchaseInput({
      ...buildFuelPurchaseDefaults(row, NOW),
      workerId: "",
      vendor: "",
      vendorCity: "  ",
      unitPrice: "",
      odometer: null,
      fuelCardId: "",
      cardLastFour: "",
      transactionReference: "",
      notes: "",
    });
    expect(input.workerId).toBeNull();
    expect(input.vendor).toBeNull();
    expect(input.vendorCity).toBeNull();
    expect(input.unitPrice).toBeNull();
    expect(input.odometer).toBeNull();
    expect(input.fuelCardId).toBeNull();
    expect(input.cardLastFour).toBeNull();
    expect(input.transactionReference).toBeNull();
    expect(input.notes).toBeNull();
  });

  it("rejects what the tax record cannot carry, naming the field", () => {
    const base = buildFuelPurchaseDefaults(row, NOW);
    const failures: Array<[Partial<typeof base>, string]> = [
      [{ quantity: "0" }, "quantity"],
      [{ quantity: "-5" }, "quantity"],
      [{ quantity: "12.3456" }, "quantity"],
      [{ unitPrice: "3.89901" }, "unitPrice"],
      [{ unitPrice: "-1" }, "unitPrice"],
      [{ purchasedAt: NOW + 60 }, "purchasedAt"],
      [{ tractorId: "" }, "tractorId"],
      [{ jurisdictionId: "" }, "jurisdictionId"],
      [{ cardLastFour: "48" }, "cardLastFour"],
      [{ cardLastFour: "48a1" }, "cardLastFour"],
      [{ totalAmount: "" }, "totalAmount"],
      [{ totalAmount: "-3.00" }, "totalAmount"],
      [{ totalAmount: "3.001" }, "totalAmount"],
      [{ odometer: -1 }, "odometer"],
      [{ currencyCode: "US" }, "currencyCode"],
    ];
    for (const [override, path] of failures) {
      const result = schema.safeParse({ ...base, ...override });
      expect(result.success, JSON.stringify(override)).toBe(false);
      expect(
        result.error?.issues.map((issue) => issue.path.join(".")),
        JSON.stringify(override),
      ).toContain(path);
    }
  });

  it("accepts a purchase made exactly now and one in litres", () => {
    const base = buildFuelPurchaseDefaults(row, NOW);
    expect(schema.safeParse({ ...base, purchasedAt: NOW }).success).toBe(true);
    expect(schema.safeParse({ ...base, quantityUnit: "Litre", quantity: "456.1" }).success).toBe(
      true,
    );
  });

  it("lets a zero total through for untaxed or bulk fuel", () => {
    const base = buildFuelPurchaseDefaults(row, NOW);
    expect(schema.safeParse({ ...base, totalAmount: "0.00", unitPrice: null }).success).toBe(true);
  });
});
