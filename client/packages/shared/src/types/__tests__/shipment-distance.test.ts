import { describe, expect, it } from "vitest";
import { shipmentDistanceResponseSchema } from "@trenova/shared/types/shipment";

/**
 * `ShipmentDistanceResponse` in shipment.graphqls declares `shipmentId`, and on each
 * `ShipmentDistanceMoveResult` `moveId`, `provider`, `routingType`, `dataVersion` and
 * `warnings`, as nullable. GraphQL sends a field it has no value for as `null`, so a
 * distance taken from a stored mileage or an override rather than PC*MILER arrives with
 * those fields null, and recalculating it must not fail to parse.
 */
const providerlessMove = {
  moveId: "smv_1",
  moveIndex: 0,
  distance: 412.5,
  source: "StoredMileage",
  provider: null,
  routingType: null,
  dataVersion: null,
  warnings: null,
  calculatedAt: 1_792_000_000,
};

describe("shipmentDistanceResponseSchema", () => {
  it("reads a recalculation whose moves carry no provider details", () => {
    const parsed = shipmentDistanceResponseSchema.parse({
      shipmentId: "shp_1",
      totalDistance: 825,
      moves: [providerlessMove, { ...providerlessMove, moveId: "smv_2", moveIndex: 1 }],
    });

    expect(parsed.moves).toHaveLength(2);
    expect(parsed.moves[0]).toEqual({
      moveId: "smv_1",
      moveIndex: 0,
      distance: 412.5,
      source: "StoredMileage",
      provider: null,
      routingType: null,
      dataVersion: null,
      warnings: [],
      calculatedAt: 1_792_000_000,
    });
    expect(parsed.moves[1].warnings).toEqual([]);
  });

  it("reads a calculation for a shipment and moves not yet saved", () => {
    const parsed = shipmentDistanceResponseSchema.parse({
      shipmentId: null,
      totalDistance: 412.5,
      moves: [{ ...providerlessMove, moveId: null }],
    });

    expect(parsed.shipmentId).toBeNull();
    expect(parsed.moves[0].moveId).toBeNull();
  });

  it("reads fields the server leaves out the same as null ones", () => {
    const parsed = shipmentDistanceResponseSchema.parse({
      totalDistance: 412.5,
      moves: [
        {
          moveIndex: 0,
          distance: 412.5,
          source: "Override",
          calculatedAt: 1_792_000_000,
        },
      ],
    });

    expect(parsed.moves[0].warnings).toEqual([]);
    expect(parsed.moves[0].provider).toBeUndefined();
  });

  it("keeps the provider details and warnings a routed distance carries", () => {
    const parsed = shipmentDistanceResponseSchema.parse({
      shipmentId: "shp_1",
      totalDistance: 412.5,
      moves: [
        {
          ...providerlessMove,
          source: "PCMiler",
          provider: "PCMiler",
          routingType: "Practical",
          dataVersion: "2026Q3",
          warnings: ["Stop 2 was geocoded to its city centre", "Toll roads were avoided"],
        },
      ],
    });

    expect(parsed.moves[0]).toMatchObject({
      provider: "PCMiler",
      routingType: "Practical",
      dataVersion: "2026Q3",
      warnings: ["Stop 2 was geocoded to its city centre", "Toll roads were avoided"],
    });
  });
});
