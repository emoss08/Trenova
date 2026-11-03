import { describe, expect, it } from "vitest";
import { groupLanes, laneArcPath, laneWeight } from "../map-lanes";
import { buildMapShipments } from "../map-shipment-model";
import { PLACES, shipment } from "./fixtures";

const { dallas, memphis, atlanta, chicago } = PLACES;

describe("groupLanes", () => {
  it("carries every load on the same origin and destination in one lane, lightest lane first", () => {
    const models = buildMapShipments([
      shipment("a", [{ at: dallas }, { at: atlanta, type: "Delivery" }], { stage: "Late" }),
      shipment("b", [{ at: dallas }, { at: atlanta, type: "Delivery" }]),
      shipment("c", [
        { at: dallas },
        { at: memphis, type: "Delivery" },
        { at: atlanta, type: "Delivery" },
      ]),
      shipment("d", [{ at: chicago }, { at: memphis, type: "Delivery" }]),
    ]);

    expect(groupLanes(models).map((lane) => [lane.key, lane.shipmentIds, lane.lateCount])).toEqual([
      ["Chicago, IL > Memphis, TN", ["d"], 0],
      ["Dallas, TX > Atlanta, GA", ["a", "b", "c"], 1],
    ]);
  });

  it("keeps a lane and its return lane apart", () => {
    const models = buildMapShipments([
      shipment("out", [{ at: dallas }, { at: atlanta, type: "Delivery" }]),
      shipment("back", [{ at: atlanta }, { at: dallas, type: "Delivery" }]),
    ]);

    expect(
      groupLanes(models)
        .map((lane) => lane.key)
        .sort(),
    ).toEqual(["Atlanta, GA > Dallas, TX", "Dallas, TX > Atlanta, GA"]);
  });
});

describe("laneArcPath", () => {
  const origin = { lat: dallas.lat, lng: dallas.lng };
  const destination = { lat: atlanta.lat, lng: atlanta.lng };

  it("starts at the origin and ends at the destination", () => {
    const path = laneArcPath(origin, destination);

    expect(path[0].lat).toBeCloseTo(origin.lat, 9);
    expect(path[0].lng).toBeCloseTo(origin.lng, 9);
    expect(path.at(-1)?.lat).toBeCloseTo(destination.lat, 9);
    expect(path.at(-1)?.lng).toBeCloseTo(destination.lng, 9);
  });

  it("bows to the left of travel, so the return lane bows the other way", () => {
    const straightMidLat = (origin.lat + destination.lat) / 2;
    const eastbound = laneArcPath(origin, destination);
    const westbound = laneArcPath(destination, origin);

    expect(eastbound[12].lat).toBeGreaterThan(straightMidLat);
    expect(westbound[12].lat).toBeLessThan(straightMidLat);
  });
});

describe("laneWeight", () => {
  it("thickens with each load and stops at ten pixels", () => {
    expect(laneWeight(1)).toBe(2);
    expect(laneWeight(3)).toBe(5);
    expect(laneWeight(40)).toBe(10);
  });
});
