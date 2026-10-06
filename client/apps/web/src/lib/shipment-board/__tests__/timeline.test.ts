import { describe, expect, it } from "vitest";
import type { Shipment } from "@trenova/shared/types/shipment";
import { buildTimelineRows } from "../timeline";

const DAY = 1_790_000_000;
const HOUR = 3600;

type Fixture = {
  id: string;
  stage: Shipment["stage"];
  pickupHour: number | null;
  deliveryHour: number;
  deliveryEndHour?: number;
  etaHour?: number;
};

function shipment({
  id,
  stage,
  pickupHour,
  deliveryHour,
  deliveryEndHour,
  etaHour,
}: Fixture): Shipment {
  return {
    id,
    stage,
    eta: etaHour === undefined ? null : { estimatedArrival: DAY + etaHour * HOUR },
    moves: [
      {
        stops: [
          {
            type: "Pickup",
            sequence: 0,
            scheduledWindowStart: pickupHour === null ? null : DAY + pickupHour * HOUR,
          },
          {
            type: "Delivery",
            sequence: 1,
            scheduledWindowStart: DAY + deliveryHour * HOUR,
            scheduledWindowEnd: deliveryEndHour === undefined ? null : DAY + deliveryEndHour * HOUR,
          },
        ],
      },
    ],
  } as unknown as Shipment;
}

const ids = (rows: ReturnType<typeof buildTimelineRows>) => rows.map((row) => row.shipment.id);

describe("buildTimelineRows", () => {
  it("keeps the server's stage order and runs by pickup inside each stage", () => {
    const rows = buildTimelineRows(
      [
        shipment({ id: "late-10", stage: "Late", pickupHour: 10, deliveryHour: 14 }),
        shipment({ id: "moving-5", stage: "Moving", pickupHour: 5, deliveryHour: 9 }),
        shipment({ id: "late-8", stage: "Late", pickupHour: 8, deliveryHour: 12 }),
        shipment({ id: "delivered-4", stage: "Delivered", pickupHour: 4, deliveryHour: 6 }),
      ],
      DAY,
    );

    expect(ids(rows)).toEqual(["late-8", "late-10", "moving-5", "delivered-4"]);
  });

  it("leaves out a load with no pickup appointment, whatever its stage", () => {
    const rows = buildTimelineRows(
      [
        shipment({ id: "unscheduled", stage: "NeedsCoverage", pickupHour: null, deliveryHour: 9 }),
        shipment({ id: "moving", stage: "Moving", pickupHour: 6, deliveryHour: 9 }),
      ],
      DAY,
    );

    expect(ids(rows)).toEqual(["moving"]);
  });

  it("measures delivery to the end of the window and hatches only a late load's overrun", () => {
    const [late, moving] = buildTimelineRows(
      [
        shipment({
          id: "late",
          stage: "Late",
          pickupHour: 6,
          deliveryHour: 12,
          deliveryEndHour: 13,
          etaHour: 15.5,
        }),
        shipment({ id: "moving", stage: "Moving", pickupHour: 7, deliveryHour: 12, etaHour: 15 }),
      ],
      DAY,
    );

    expect(late).toMatchObject({ pickup: 6, delivery: 13, slipped: 15.5 });
    expect(moving).toMatchObject({ pickup: 7, delivery: 12, slipped: null });
  });

  it("does not hatch a late load whose projection is inside its window", () => {
    const [row] = buildTimelineRows(
      [
        shipment({
          id: "late",
          stage: "Late",
          pickupHour: 6,
          deliveryHour: 12,
          deliveryEndHour: 14,
          etaHour: 13,
        }),
      ],
      DAY,
    );

    expect(row?.slipped).toBeNull();
  });
});
