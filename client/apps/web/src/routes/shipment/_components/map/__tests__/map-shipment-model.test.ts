import { describe, expect, it } from "vitest";
import {
  buildMapShipments,
  countByStage,
  groupByStage,
  matchesMapSearch,
  nextOpenStop,
  remainingPath,
} from "../map-shipment-model";
import { PLACES, shipment } from "./fixtures";

const { dallas, memphis, atlanta, houston, chicago } = PLACES;

describe("buildMapShipments", () => {
  it("places a moving load at the stop it is heading to", () => {
    const [model] = buildMapShipments([
      shipment("a", [
        { at: dallas, status: "Completed" },
        { at: memphis, type: "Delivery" },
        { at: atlanta, type: "Delivery" },
      ]),
    ]);

    expect(model.position).toEqual({ lat: memphis.lat, lng: memphis.lng });
    expect(model.currentIndex).toBe(1);
  });

  it("places a load that has not started at its first pickup", () => {
    const [model] = buildMapShipments([
      shipment("a", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Scheduled" }),
    ]);

    expect(model.position).toEqual({ lat: dallas.lat, lng: dallas.lng });
    expect(model.stage).toBe("Scheduled");
  });

  it("names the lane from the first pickup to the last delivery", () => {
    const [model] = buildMapShipments([
      shipment("a", [
        { at: dallas },
        { at: memphis, type: "Delivery" },
        { at: atlanta, type: "Delivery" },
      ]),
    ]);

    expect(model.lane).toMatchObject({
      key: "Dallas, TX > Atlanta, GA",
      originLabel: "Dallas, TX",
      destinationLabel: "Atlanta, GA",
    });
  });

  it("treats a load with no stage yet as needing coverage", () => {
    const [model] = buildMapShipments([
      shipment("a", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: undefined }),
    ]);

    expect(model.stage).toBe("NeedsCoverage");
  });

  it("leaves off canceled loads, loads with no id, and loads with no stop it can place", () => {
    const models = buildMapShipments([
      shipment("canceled", [{ at: dallas }, { at: memphis, type: "Delivery" }], {
        stage: "Canceled",
      }),
      shipment(undefined, [{ at: dallas }, { at: memphis, type: "Delivery" }]),
      shipment("nowhere", [{ at: null }, { at: null, type: "Delivery" }]),
      shipment("kept", [{ at: null }, { at: memphis, type: "Delivery" }]),
    ]);

    expect(models.map((m) => m.id)).toEqual(["kept"]);
    expect(models[0].stops).toHaveLength(1);
    expect(models[0].lane).toBeNull();
  });
});

describe("remainingPath", () => {
  it("runs from the last stop the load left through every stop still ahead", () => {
    const [model] = buildMapShipments([
      shipment("a", [
        { at: houston, status: "Completed" },
        { at: dallas, status: "Completed" },
        { at: memphis, type: "Delivery" },
        { at: chicago, type: "Delivery" },
      ]),
    ]);

    expect(remainingPath(model)).toEqual([
      { lat: dallas.lat, lng: dallas.lng },
      { lat: memphis.lat, lng: memphis.lng },
      { lat: chicago.lat, lng: chicago.lng },
    ]);
  });

  it("is empty for a delivered load and for one whose stops are all closed", () => {
    const [delivered, closed] = buildMapShipments([
      shipment("d", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Delivered" }),
      shipment("c", [
        { at: dallas, status: "Completed" },
        { at: memphis, type: "Delivery", status: "Canceled" },
      ]),
    ]);

    expect(remainingPath(delivered)).toEqual([]);
    expect(remainingPath(closed)).toEqual([]);
    expect(nextOpenStop(closed)).toBeNull();
  });
});

describe("groupByStage", () => {
  it("lists stages most urgent first and each stage's most urgent load first", () => {
    const models = buildMapShipments([
      shipment("moving-late-eta", [{ at: dallas }, { at: memphis, type: "Delivery" }], {
        stage: "Moving",
        eta: { estimatedArrival: 2000, slackMinutes: 30, verdict: "OnTime" },
      }),
      shipment("late-a-little", [{ at: dallas }, { at: memphis, type: "Delivery" }], {
        stage: "Late",
        eta: { estimatedArrival: 1500, slackMinutes: -20, verdict: "Late" },
      }),
      shipment("moving-soon", [{ at: dallas }, { at: memphis, type: "Delivery" }], {
        stage: "Moving",
        eta: { estimatedArrival: 1000, slackMinutes: 10, verdict: "OnTime" },
      }),
      shipment("late-a-lot", [{ at: dallas }, { at: memphis, type: "Delivery" }], {
        stage: "Late",
        eta: { estimatedArrival: 1800, slackMinutes: -180, verdict: "Late" },
      }),
      shipment(
        "scheduled",
        [
          { at: dallas, window: 500 },
          { at: memphis, type: "Delivery" },
        ],
        {
          stage: "Scheduled",
        },
      ),
    ]);

    expect(
      groupByStage(models).map((group) => [group.stage, group.items.map((m) => m.id)]),
    ).toEqual([
      ["Late", ["late-a-lot", "late-a-little"]],
      ["Moving", ["moving-soon", "moving-late-eta"]],
      ["Scheduled", ["scheduled"]],
    ]);
  });
});

describe("countByStage", () => {
  it("counts every stage, including the empty ones", () => {
    const models = buildMapShipments([
      shipment("a", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Late" }),
      shipment("b", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Late" }),
      shipment("c", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Moving" }),
    ]);

    expect(countByStage(models)).toEqual({
      Late: 2,
      NeedsCoverage: 0,
      Moving: 1,
      Scheduled: 0,
      Delivered: 0,
    });
  });
});

describe("matchesMapSearch", () => {
  const [model] = buildMapShipments([
    shipment("a", [{ at: dallas }, { at: memphis, type: "Delivery" }]),
  ]);

  it("finds a load by PRO number, customer or city, ignoring case", () => {
    expect(matchesMapSearch(model, "pro-a")).toBe(true);
    expect(matchesMapSearch(model, "HALVERSON")).toBe(true);
    expect(matchesMapSearch(model, "memphis, tn")).toBe(true);
    expect(matchesMapSearch(model, "chicago")).toBe(false);
  });

  it("matches everything for an empty search", () => {
    expect(matchesMapSearch(model, "   ")).toBe(true);
  });
});
