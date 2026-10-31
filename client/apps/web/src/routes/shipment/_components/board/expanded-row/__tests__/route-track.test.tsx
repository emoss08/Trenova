import { cleanup, render, screen } from "@testing-library/react";
import { formatToUserTimezone, formatUnixTime } from "@trenova/shared/lib/date";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it } from "vitest";
import { RouteTrack, truckLeft } from "../route-track";

const DAY = 24 * 60 * 60;
const now = Math.floor(Date.now() / 1000);
const PICKUP_START = now - 14 * DAY;
const PICKUP_END = PICKUP_START + 2 * 60 * 60;
const DELIVERY_DUE = now - 3 * DAY;

// A load the board reads as late: it left its pickup two weeks ago and its delivery appointment
// has passed, so its progress sits at the ceiling short of the delivery.
function lateShipment(): Shipment {
  return {
    id: "shp_1",
    status: "InTransit",
    stage: "Late",
    eta: { slackMinutes: -3600, verdict: "Late", reason: "From the position on a check call" },
    moves: [
      {
        distance: 7800,
        stops: [
          {
            type: "Pickup",
            scheduledWindowStart: PICKUP_START,
            scheduledWindowEnd: PICKUP_END,
            actualDeparture: PICKUP_START + 60 * 60,
            location: { name: "Turner Foods Corp Reno Plant", city: "Reno" },
          },
          {
            type: "Delivery",
            scheduledWindowStart: DELIVERY_DUE,
            scheduledWindowEnd: null,
            location: { name: "Harvest Corp Store #5946", city: "Detroit" },
          },
        ],
      },
    ],
  } as unknown as Shipment;
}

afterEach(cleanup);

describe("RouteTrack", () => {
  it("ends a late load's hatch at the delivery, however close the truck is to it", () => {
    render(<RouteTrack shipment={lateShipment()} />);

    const hatch = screen.getByTestId("route-late-hatch");
    expect(hatch.style.left).toBe("98%");
    expect(hatch.style.right).toBe("0px");
    expect(hatch.style.width).toBe("");
  });

  it("keeps the truck clear of both ends of the track", () => {
    expect(truckLeft(98)).toBe("clamp(0.625rem, 98%, calc(100% - 0.625rem))");
    expect(truckLeft(2)).toBe("clamp(0.625rem, 2%, calc(100% - 0.625rem))");
  });

  it("writes a window's end as a time of day, not the last word of a date", () => {
    render(<RouteTrack shipment={lateShipment()} />);

    const start = formatToUserTimezone(PICKUP_START, { showTimeZone: false, showSeconds: false });
    expect(screen.getByText(`Reno · ${start} – ${formatUnixTime(PICKUP_END)}`)).toBeInTheDocument();
  });
});
