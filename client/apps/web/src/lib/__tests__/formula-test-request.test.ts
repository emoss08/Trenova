import { describe, expect, it } from "vitest";
import { formulaTestRequest, type FormulaPricingContent } from "../formula-test-request";

function content(extra: Partial<FormulaPricingContent> = {}): FormulaPricingContent {
  return {
    expression: "baseRate * totalDistance",
    schemaId: "shipment",
    variableDefinitions: [
      { name: "baseRate", type: "Number", description: "", required: true, defaultValue: 2.85 },
    ],
    breakdownDefinitions: [],
    minCharge: null,
    maxCharge: null,
    roundingMode: "HalfUp",
    roundingPrecision: 2,
    ...extra,
  };
}

const SAMPLE = { totalDistance: 500, baseRate: 1, totalWeight: 10000 };

describe("formulaTestRequest", () => {
  it("prices nothing when there is no expression", () => {
    expect(formulaTestRequest(content({ expression: "" }), { sampleValues: SAMPLE })).toBeNull();
    expect(
      formulaTestRequest(content({ expression: "  \n " }), { sampleValues: SAMPLE }),
    ).toBeNull();
  });

  it("lays the template's defaults over the sample load and the person's values over those", () => {
    const request = formulaTestRequest(
      content({
        variableDefinitions: [
          { name: "baseRate", type: "Number", description: "", required: true, defaultValue: 2.85 },
          { name: "fuelPct", type: "Number", description: "", required: false, defaultValue: 0.1 },
        ],
      }),
      { sampleValues: SAMPLE, overrides: { fuelPct: 0.2 } },
    );

    expect(request?.variables).toEqual({
      totalDistance: 500,
      baseRate: 2.85,
      totalWeight: 10000,
      fuelPct: 0.2,
    });
  });

  it("keeps a sample value a variable has no default for", () => {
    const request = formulaTestRequest(
      content({
        variableDefinitions: [
          { name: "baseRate", type: "Number", description: "", required: true, defaultValue: null },
          { name: "totalWeight", type: "Number", description: "", required: false },
        ],
      }),
      { sampleValues: SAMPLE },
    );

    expect(request?.variables).toEqual(SAMPLE);
  });

  it("ignores an override that holds no value", () => {
    const request = formulaTestRequest(content(), {
      sampleValues: SAMPLE,
      overrides: { baseRate: undefined },
    });

    expect(request?.variables.baseRate).toBe(2.85);
  });

  it("prices a saved shipment from its own values rather than the sample load", () => {
    const request = formulaTestRequest(content(), {
      sampleValues: SAMPLE,
      shipmentId: "shp_01JSAVED0000000000000000",
    });

    expect(request?.shipmentId).toBe("shp_01JSAVED0000000000000000");
    expect(request?.variables).toEqual({ baseRate: 2.85 });
  });

  it("sends no shipment and only defaults when there is neither a shipment nor a sample", () => {
    const request = formulaTestRequest(content(), { sampleValues: null });

    expect(request).not.toHaveProperty("shipmentId");
    expect(request?.variables).toEqual({ baseRate: 2.85 });
  });

  it("drops breakdown lines without a name or an expression", () => {
    const request = formulaTestRequest(
      content({
        breakdownDefinitions: [
          { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
          { name: " ", label: "Blank name", expression: "1" },
          { name: "fuel", label: "Fuel", expression: "  " },
        ],
      }),
      { sampleValues: SAMPLE },
    );

    expect(request?.breakdowns).toEqual([
      { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
    ]);
  });

  it("leaves breakdowns out when none remain", () => {
    const request = formulaTestRequest(
      content({ breakdownDefinitions: [{ name: "", label: "Nothing", expression: "" }] }),
      { sampleValues: SAMPLE },
    );

    expect(request).not.toHaveProperty("breakdowns");
  });

  it("sends the limits as decimal strings, a zero included, and leaves out the unset ones", () => {
    expect(
      formulaTestRequest(content({ minCharge: 0, maxCharge: 5000 }), { sampleValues: SAMPLE }),
    ).toMatchObject({ minCharge: "0", maxCharge: "5000" });

    const unset = formulaTestRequest(content({ minCharge: null, maxCharge: undefined }), {
      sampleValues: SAMPLE,
    });
    expect(unset).not.toHaveProperty("minCharge");
    expect(unset).not.toHaveProperty("maxCharge");
  });

  it("sends a rounding precision of zero", () => {
    const request = formulaTestRequest(content({ roundingMode: "Down", roundingPrecision: 0 }), {
      sampleValues: SAMPLE,
    });

    expect(request).toMatchObject({ roundingMode: "Down", roundingPrecision: 0 });
  });

  it("prices against the shipment schema when the template names none", () => {
    expect(formulaTestRequest(content({ schemaId: "" }), { sampleValues: SAMPLE })?.schemaId).toBe(
      "shipment",
    );
    expect(
      formulaTestRequest(content({ schemaId: undefined }), { sampleValues: SAMPLE })?.schemaId,
    ).toBe("shipment");
  });

  it("leaves what it was given unchanged", () => {
    const template = content({
      breakdownDefinitions: [{ name: "linehaul", label: "Linehaul", expression: "1" }],
    });
    const sample = { ...SAMPLE };
    const overrides = { baseRate: 3 };
    const before = structuredClone({ template, sample, overrides });

    formulaTestRequest(template, { sampleValues: sample, overrides });

    expect({ template, sample, overrides }).toEqual(before);
  });
});
