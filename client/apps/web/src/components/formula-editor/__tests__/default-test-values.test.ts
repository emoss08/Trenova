import { describe, expect, it } from "vitest";
import shipmentSchemaJSON from "../../../../../../../services/tms/internal/modules/rating/app/formula/schema/definitions/shipment.schema.json";
import { DEFAULT_TEST_VALUES } from "../default-test-values";

type SchemaProperty = {
  properties?: Record<string, SchemaProperty>;
};

const shipmentSchema = shipmentSchemaJSON as unknown as {
  properties: Record<string, SchemaProperty>;
};

function variablePaths(properties: Record<string, SchemaProperty>, prefix = ""): string[] {
  return Object.entries(properties).flatMap(([name, property]) => {
    const path = prefix ? `${prefix}.${name}` : name;
    return property.properties ? variablePaths(property.properties, path) : [path];
  });
}

describe("DEFAULT_TEST_VALUES", () => {
  it("has a sample for every variable the shipment schema exposes", () => {
    const missing = variablePaths(shipmentSchema.properties).filter(
      (path) => !(path in DEFAULT_TEST_VALUES),
    );

    expect(missing).toEqual([]);
  });

  it("samples only variables the shipment schema exposes", () => {
    const known = new Set(variablePaths(shipmentSchema.properties));
    const unknown = Object.keys(DEFAULT_TEST_VALUES).filter((key) => !known.has(key));

    expect(unknown).toEqual([]);
  });

  it("agrees with itself about the load it describes", () => {
    const stops = DEFAULT_TEST_VALUES.stops as Array<Record<string, unknown>>;
    const commodities = DEFAULT_TEST_VALUES.commodities as Array<Record<string, unknown>>;

    expect(stops).toHaveLength(DEFAULT_TEST_VALUES.totalStops as number);
    expect(commodities.reduce((sum, line) => sum + (line.weight as number), 0)).toBe(
      DEFAULT_TEST_VALUES.totalWeight,
    );
    expect(commodities.reduce((sum, line) => sum + (line.pieces as number), 0)).toBe(
      DEFAULT_TEST_VALUES.totalPieces,
    );
    expect(DEFAULT_TEST_VALUES.requiresTemperatureControl).toBe(true);
    expect(DEFAULT_TEST_VALUES.temperatureDifferential).toBe(
      (DEFAULT_TEST_VALUES.temperatureMax as number) -
        (DEFAULT_TEST_VALUES.temperatureMin as number),
    );
    expect(stops[0]?.city).toBe(DEFAULT_TEST_VALUES["origin.city"]);
    expect(stops.at(-1)?.city).toBe(DEFAULT_TEST_VALUES["destination.city"]);
  });
});
