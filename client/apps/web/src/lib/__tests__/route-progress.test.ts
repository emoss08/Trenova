import { describe, expect, it } from "vitest";
import type { Shipment } from "@trenova/shared/types/shipment";
import { getRouteProgress } from "../shipment-utils";

const HOUR = 3600;

function shipment(status: string, stops: Array<Record<string, unknown>>, distance = 300): Shipment {
  return { id: "s", status, moves: [{ id: "m", distance, stops }] } as unknown as Shipment;
}

const pickup = (departed?: number) => ({
  type: "Pickup",
  scheduledWindowStart: 0,
  actualDeparture: departed ?? null,
});
const delivery = (start: number, end?: number) => ({
  type: "Delivery",
  scheduledWindowStart: start,
  scheduledWindowEnd: end ?? null,
  actualArrival: null,
});

describe("getRouteProgress", () => {
  it("is empty before the load departs and full once it is delivered", () => {
    expect(getRouteProgress(shipment("Assigned", [pickup(), delivery(10 * HOUR)]), 5 * HOUR)).toEqual({
      percent: 0,
      milesDone: 0,
      milesLeft: 300,
      totalMiles: 300,
    });
    expect(
      getRouteProgress(shipment("Completed", [pickup(HOUR), delivery(10 * HOUR)]), 5 * HOUR).percent,
    ).toBe(100);
  });

  it("estimates a moving load's progress from departure to its appointment", () => {
    const progress = getRouteProgress(
      shipment("InTransit", [pickup(2 * HOUR), delivery(12 * HOUR)]),
      7 * HOUR,
    );
    expect(progress.percent).toBe(50);
    expect(progress.milesDone).toBe(150);
    expect(progress.milesLeft).toBe(150);
  });

  it("never shows a moving load as arrived or not started", () => {
    const early = getRouteProgress(shipment("InTransit", [pickup(2 * HOUR), delivery(12 * HOUR)]), 2 * HOUR);
    const overdue = getRouteProgress(shipment("Delayed", [pickup(2 * HOUR), delivery(12 * HOUR)]), 20 * HOUR);
    expect(early.percent).toBe(2);
    expect(overdue.percent).toBe(98);
  });

  it("copes with a shipment that has no moves", () => {
    expect(getRouteProgress({ id: "s", status: "New" } as Shipment, 0)).toEqual({
      percent: 0,
      milesDone: 0,
      milesLeft: 0,
      totalMiles: 0,
    });
  });
});
