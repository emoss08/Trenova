import { describe, expect, it } from "vitest";
import { buildMapShipments } from "../map-shipment-model";
import { alertBounds, matchShipmentsToAlerts, samplePath } from "../weather-alert-matching";
import { alert, PLACES, shipment, square } from "./fixtures";

const { dallas, memphis, houston, chicago } = PLACES;

/* Dallas to Memphis, heading for Memphis: the mark sits at Memphis and the
   leg from Dallas is still ahead. */
const dallasToMemphis = () =>
  buildMapShipments([
    shipment("dm", [
      { at: dallas, status: "Completed" },
      { at: memphis, type: "Delivery" },
    ]),
  ]);

const midLat = (dallas.lat + memphis.lat) / 2;
const midLng = (dallas.lng + memphis.lng) / 2;

describe("matchShipmentsToAlerts", () => {
  it("says the next stop is inside when the alert covers it", () => {
    const matches = matchShipmentsToAlerts(dallasToMemphis(), [
      alert("around-memphis", [square(memphis.lat, memphis.lng, 0.3)]),
    ]);

    expect(matches.byShipment.get("dm")).toEqual([
      { alertId: "around-memphis", shipmentId: "dm", at: "stop" },
    ]);
    expect(matches.byAlert.get("around-memphis")).toEqual([
      { alertId: "around-memphis", shipmentId: "dm", at: "stop" },
    ]);
  });

  it("says the route ahead passes through an alert between two stops", () => {
    const matches = matchShipmentsToAlerts(dallasToMemphis(), [
      alert("midway", [square(midLat, midLng, 0.2)]),
    ]);

    expect(matches.byShipment.get("dm")).toEqual([
      { alertId: "midway", shipmentId: "dm", at: "route" },
    ]);
  });

  it("ignores an alert on a leg the load has already driven", () => {
    const models = buildMapShipments([
      shipment("hdm", [
        { at: houston, status: "Completed" },
        { at: dallas, status: "Completed" },
        { at: memphis, type: "Delivery" },
      ]),
    ]);
    const matches = matchShipmentsToAlerts(models, [
      alert("behind", [square(houston.lat, houston.lng, 0.3)]),
    ]);

    expect(matches.byShipment.has("hdm")).toBe(false);
    expect(matches.byAlert.has("behind")).toBe(false);
  });

  it("does not match a delivered load", () => {
    const models = buildMapShipments([
      shipment("done", [{ at: dallas }, { at: memphis, type: "Delivery" }], { stage: "Delivered" }),
    ]);
    const matches = matchShipmentsToAlerts(models, [
      alert("around-memphis", [square(memphis.lat, memphis.lng, 0.3)]),
    ]);

    expect(matches.byShipment.size).toBe(0);
  });

  it("finds a load in any part of a multi-part alert", () => {
    const matches = matchShipmentsToAlerts(dallasToMemphis(), [
      alert("split", [square(chicago.lat, chicago.lng, 0.3), square(midLat, midLng, 0.2)], {
        multi: true,
      }),
    ]);

    expect(matches.byShipment.get("dm")).toEqual([
      { alertId: "split", shipmentId: "dm", at: "route" },
    ]);
  });

  it("lists a load's alerts most severe first, and the one at its stop first within a severity", () => {
    const matches = matchShipmentsToAlerts(dallasToMemphis(), [
      alert("minor-midway", [square(midLat, midLng, 0.2)], { severity: "Minor" }),
      alert("severe-midway", [square(midLat, midLng, 0.25)], { severity: "Severe" }),
      alert("severe-at-stop", [square(memphis.lat, memphis.lng, 0.3)], { severity: "Severe" }),
      alert("extreme-midway", [square(midLat, midLng, 0.3)], { severity: "Extreme" }),
    ]);

    expect(matches.byShipment.get("dm")?.map((hit) => hit.alertId)).toEqual([
      "extreme-midway",
      "severe-at-stop",
      "severe-midway",
      "minor-midway",
    ]);
  });

  it("lists every load an alert touches", () => {
    const models = buildMapShipments([
      shipment("one", [
        { at: dallas, status: "Completed" },
        { at: memphis, type: "Delivery" },
      ]),
      shipment("two", [{ at: memphis }, { at: chicago, type: "Delivery" }], { stage: "Scheduled" }),
      shipment("far", [{ at: houston }, { at: dallas, type: "Delivery" }]),
    ]);
    const matches = matchShipmentsToAlerts(models, [
      alert("around-memphis", [square(memphis.lat, memphis.lng, 0.3)]),
    ]);

    expect(matches.byAlert.get("around-memphis")).toEqual([
      { alertId: "around-memphis", shipmentId: "one", at: "stop" },
      { alertId: "around-memphis", shipmentId: "two", at: "stop" },
    ]);
  });

  it("skips an alert with no area", () => {
    const noArea = alert("no-area", [square(memphis.lat, memphis.lng, 0.3)]);
    noArea.geometry = { type: "Point", coordinates: [] as unknown as number[][][] };

    expect(matchShipmentsToAlerts(dallasToMemphis(), [noArea]).byShipment.size).toBe(0);
  });
});

describe("samplePath", () => {
  it("leaves no gap wider than a tenth of a degree along a leg", () => {
    const samples = samplePath([
      { lat: 30, lng: -100 },
      { lat: 30, lng: -98 },
    ]);

    expect(samples[0]).toEqual({ lat: 30, lng: -100 });
    expect(samples.at(-1)).toEqual({ lat: 30, lng: -98 });
    for (let i = 1; i < samples.length; i++) {
      expect(samples[i].lng - samples[i - 1].lng).toBeLessThanOrEqual(0.1 + 1e-9);
    }
  });
});

describe("alertBounds", () => {
  it("spans every part of a multi-part alert", () => {
    const feature = alert("split", [square(30, -95, 1), square(40, -85, 1)], { multi: true });

    expect(alertBounds(feature)).toEqual({ south: 29, north: 41, west: -96, east: -84 });
  });
});
