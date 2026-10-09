import { describe, expect, it } from "vitest";
import { getStopStrip, shipmentPanelPath } from "@/lib/shipment-utils";
import type { Shipment, Stop } from "@trenova/shared/types/shipment";

describe("shipmentPanelPath", () => {
  it("expands the shipment's row and opens its edit panel", () => {
    expect(shipmentPanelPath("shp_01M29TH78AH2BQ7NP65RE51PQN")).toBe(
      "/shipment-management/shipments?expanded=shp_01M29TH78AH2BQ7NP65RE51PQN&panelType=edit&panelEntityId=shp_01M29TH78AH2BQ7NP65RE51PQN",
    );
  });

  it("encodes an id that is not URL-safe rather than splicing it into the query", () => {
    expect(shipmentPanelPath("a&b=c")).toBe(
      "/shipment-management/shipments?expanded=a%26b%3Dc&panelType=edit&panelEntityId=a%26b%3Dc",
    );
  });
});

describe("getStopStrip", () => {
  const stop = (overrides: Partial<Stop>): Stop =>
    ({
      id: overrides.id,
      status: "New",
      type: "Pickup",
      scheduleType: "Appointment",
      sequence: 0,
      scheduledWindowStart: 1_000,
      scheduledWindowEnd: null,
      actualArrival: null,
      actualDeparture: null,
      location: { name: "Dock", city: "Dallas" },
      ...overrides,
    }) as Stop;

  const shipment = (status: Shipment["status"], stops: Stop[]): Shipment =>
    ({ status, moves: [{ stops }] }) as unknown as Shipment;

  it("marks the first unreached stop current once the load is moving", () => {
    const strip = getStopStrip(
      shipment("InTransit", [
        stop({ id: "s1", status: "Completed", actualArrival: 1_000 }),
        stop({ id: "s2", type: "Delivery", scheduledWindowStart: 50_000 }),
        stop({ id: "s3", type: "Delivery", scheduledWindowStart: 90_000 }),
      ]),
      10_000,
    );

    expect(strip.map((entry) => [entry.id, entry.kind, entry.state])).toEqual([
      ["s1", "pickup", "done"],
      ["s2", "delivery", "current"],
      ["s3", "delivery", "upcoming"],
    ]);
  });

  it("calls a stop late past its window's end plus grace, whether reached or not", () => {
    const strip = getStopStrip(
      shipment("InTransit", [
        stop({
          id: "arrived-late",
          status: "Completed",
          scheduledWindowStart: 1_000,
          scheduledWindowEnd: 2_000,
          actualArrival: 2_000 + 15 * 60 + 1,
        }),
        stop({ id: "within-grace", type: "Delivery", scheduledWindowStart: 9_500 }),
        stop({ id: "overdue", type: "Delivery", scheduledWindowStart: 5_000 }),
      ]),
      10_000,
    );

    expect(strip.map((entry) => [entry.id, entry.late])).toEqual([
      ["arrived-late", true],
      ["within-grace", false],
      ["overdue", true],
    ]);
  });

  it("never calls a canceled stop late and leaves nothing current before the load moves", () => {
    const strip = getStopStrip(
      shipment("New", [
        stop({ id: "c", status: "Canceled", scheduledWindowStart: 1 }),
        stop({ id: "n", scheduledWindowStart: 1 }),
      ]),
      10_000,
    );

    expect(strip.map((entry) => [entry.state, entry.late])).toEqual([
      ["canceled", false],
      ["upcoming", true],
    ]);
  });
});
