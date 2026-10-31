import { translate } from "@trenova/shared/i18n/runtime";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import type { Stop, StopEta } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { focusStop, moveHeadline } from "../move-headline";

const PICKUP_AT = 1_793_000_000;
const DELIVERY_AT = 1_793_300_000;
const ETA_AT = 1_793_400_000;

function stop(id: string, type: Stop["type"], city: string, extra: Partial<Stop> = {}): Stop {
  return {
    id,
    type,
    status: "New",
    locationId: `loc_${id}`,
    scheduledWindowStart: type === "Pickup" ? PICKUP_AT : DELIVERY_AT,
    city,
    ...extra,
  } as unknown as Stop;
}

const placeOf = (target: Stop) => (target as unknown as { city: string }).city;
const noEtas = new Map<string, StopEta>();

function headline(stops: Stop[], status = "New", etas = noEtas) {
  return moveHeadline({
    status: status as never,
    stops,
    etas,
    placeOf,
    t: translate,
  });
}

describe("moveHeadline", () => {
  it("says where and when a move that has not started picks up", () => {
    const result = headline([
      stop("a", "Pickup", "Reno, NV"),
      stop("b", "Delivery", "Detroit, MI"),
    ]);

    expect(result.title).toBe(`Picks up at Reno, NV, ${formatUnixDateTimeShort(PICKUP_AT)}`);
    expect(result.focus?.id).toBe("a");
  });

  it("says where a moving load is arriving, at its projected time over its appointment", () => {
    const stops = [
      stop("a", "Pickup", "Reno, NV", {
        actualArrival: 1,
        actualDeparture: 2,
        status: "Completed",
      }),
      stop("b", "Delivery", "Detroit, MI"),
    ];
    const etas = new Map([["b", { stopId: "b", estimatedArrival: ETA_AT } as StopEta]]);

    expect(headline(stops, "InTransit").title).toBe(
      `Arriving at Detroit, MI, ${formatUnixDateTimeShort(DELIVERY_AT)}`,
    );
    expect(headline(stops, "InTransit", etas).title).toBe(
      `Arriving at Detroit, MI, ${formatUnixDateTimeShort(ETA_AT)}`,
    );
  });

  it("says where the truck is while it is at a stop", () => {
    const arrivedAt = 1_793_310_000;
    const result = headline(
      [
        stop("a", "Pickup", "Reno, NV", { actualArrival: 1, actualDeparture: 2 }),
        stop("b", "Delivery", "Detroit, MI", { actualArrival: arrivedAt }),
      ],
      "InTransit",
    );

    expect(result.title).toBe(`At Detroit, MI since ${formatUnixDateTimeShort(arrivedAt)}`);
  });

  it("says where a finished move delivered", () => {
    const result = headline(
      [
        stop("a", "Pickup", "Reno, NV", { actualArrival: 1, actualDeparture: 2 }),
        stop("b", "Delivery", "Detroit, MI", { actualArrival: 3, actualDeparture: 4 }),
      ],
      "Completed",
    );

    expect(result).toEqual({ title: "Delivered to Detroit, MI", tone: "success", focus: null });
  });

  it("names a canceled move as canceled, and a move with no places as still to plan", () => {
    expect(headline([stop("a", "Pickup", "Reno, NV")], "Canceled").title).toBe("Canceled");
    expect(headline([stop("a", "Pickup", "", { locationId: "", addressLine: null })]).title).toBe(
      "Add the stops to plan this move",
    );
  });

  it("speaks of the delivery when a move's pickup was canceled before it started", () => {
    const result = headline([
      stop("a", "Pickup", "Reno, NV", { status: "Canceled" }),
      stop("b", "Delivery", "Detroit, MI"),
    ]);

    expect(result.title).toBe(`Delivers to Detroit, MI, ${formatUnixDateTimeShort(DELIVERY_AT)}`);
  });
});

describe("focusStop", () => {
  it("is the first open stop that has not been left", () => {
    expect(
      focusStop([
        stop("a", "Pickup", "Reno, NV", { actualDeparture: 2 }),
        stop("b", "Delivery", "Detroit, MI", { status: "Canceled" }),
        stop("c", "Delivery", "Toledo, OH"),
      ])?.id,
    ).toBe("c");
  });
});
