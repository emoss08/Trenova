import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type {
  PublicShipmentStop,
  PublicShipmentView,
} from "@trenova/shared/types/shipment-tracking";
import { describe, expect, it } from "vitest";
import {
  dayAndTime,
  holdLabel,
  journeyOf,
  minutesOutsideWindow,
  nextStopOf,
  publicStatusAttrs,
  shows,
  trackingHeadline,
  windowRange,
} from "../shipment-tracking";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const AT = 1_800_000_000;

function stop(
  sequence: number,
  overrides: Partial<PublicShipmentStop> = {},
): PublicShipmentStop {
  return { sequence, type: "Delivery", city: `City ${sequence}`, stateCode: "TX", ...overrides };
}

function viewOf(
  stops: PublicShipmentStop[],
  overrides: Partial<PublicShipmentView> = {},
): PublicShipmentView {
  return {
    companyName: "Acme Logistics",
    shipmentProNumber: "PRO-77",
    fields: [],
    expiresAt: AT + 86_400,
    closed: false,
    status: "InTransit",
    stops,
    ...overrides,
  };
}

describe("journeyOf", () => {
  it("sits at the start before anything has happened", () => {
    const journey = journeyOf(viewOf([stop(1), stop(2)], { status: "Planned" }), AT);
    expect(journey).toEqual({
      progress: 0,
      activeIndex: 0,
      atStop: false,
      started: false,
      finished: false,
    });
  });

  it("moves along a segment with time toward the next expected arrival", () => {
    const view = viewOf([
      stop(1, { arrivedAt: AT - 7200, departedAt: AT - 3600 }),
      stop(2, { estimatedArrival: AT + 3600 }),
      stop(3, { windowStart: AT + 20_000 }),
    ]);
    const journey = journeyOf(view, AT);
    expect(journey.progress).toBeCloseTo(0.25);
    expect(journey.activeIndex).toBe(1);
    expect(journey.atStop).toBe(false);
    expect(journey.started).toBe(true);
  });

  it("stands at a stop it has reached and not left", () => {
    const view = viewOf([
      stop(1, { arrivedAt: AT - 7200, departedAt: AT - 3600 }),
      stop(2, { arrivedAt: AT - 600 }),
      stop(3),
    ]);
    expect(journeyOf(view, AT)).toMatchObject({ progress: 0.5, activeIndex: 1, atStop: true });
  });

  it("keeps the truck short of the next stop when it is overdue", () => {
    const view = viewOf([
      stop(1, { departedAt: AT - 7200 }),
      stop(2, { estimatedArrival: AT - 3600 }),
    ]);
    expect(journeyOf(view, AT).progress).toBeCloseTo(0.94);
  });

  it("sits halfway between stops when nothing says when it gets there", () => {
    const view = viewOf([stop(1, { departedAt: AT - 600 }), stop(2)]);
    expect(journeyOf(view, AT).progress).toBeCloseTo(0.5);
  });

  it("is finished once delivered", () => {
    const view = viewOf([stop(1, { departedAt: AT - 600 }), stop(2)], { status: "Delivered" });
    expect(journeyOf(view, AT)).toMatchObject({ progress: 1, finished: true });
  });
});

describe("nextStopOf", () => {
  it("skips a stop the load has left even when only its departure was reported", () => {
    const view = viewOf([stop(1, { departedAt: AT - 600 }), stop(2), stop(3)]);
    expect(nextStopOf(view)?.sequence).toBe(2);
  });

  it("is the last stop once every stop is reached", () => {
    const view = viewOf([stop(1, { arrivedAt: AT - 600 }), stop(2, { arrivedAt: AT })]);
    expect(nextStopOf(view)?.sequence).toBe(2);
  });
});

describe("trackingHeadline", () => {
  const stops = [stop(1, { city: "Omaha" }), stop(2, { city: "Minneapolis" })];

  it("says where the load is going in one sentence per status", () => {
    expect(trackingHeadline(t, viewOf(stops))).toBe("On its way to Minneapolis.");
    expect(trackingHeadline(t, viewOf(stops, { status: "Delayed" }))).toBe(
      "Running late to Minneapolis.",
    );
    expect(trackingHeadline(t, viewOf(stops, { status: "Delivered" }))).toBe(
      "Delivered in Minneapolis.",
    );
    expect(trackingHeadline(t, viewOf(stops, { status: "Planned" }))).toBe(
      "Getting ready to leave Omaha.",
    );
    expect(trackingHeadline(t, viewOf(stops, { status: "Canceled" }))).toBe(
      "This shipment was canceled.",
    );
  });

  it("reads as on its way when the customer is not shown the status", () => {
    expect(trackingHeadline(t, viewOf(stops, { status: undefined }))).toBe(
      "On its way to Minneapolis.",
    );
  });
});

describe("minutesOutsideWindow", () => {
  const appointment = stop(1, { windowStart: AT, windowEnd: AT + 3600 });

  it("is negative before the window, zero inside it and positive after it", () => {
    expect(minutesOutsideWindow(appointment, AT - 1200)).toBe(-20);
    expect(minutesOutsideWindow(appointment, AT + 1800)).toBe(0);
    expect(minutesOutsideWindow(appointment, AT + 3600 + 900)).toBe(15);
  });

  it("treats a window with no end as a single moment", () => {
    expect(minutesOutsideWindow(stop(1, { windowStart: AT }), AT + 600)).toBe(10);
  });

  it("has nothing to say without an appointment", () => {
    expect(minutesOutsideWindow(stop(1), AT)).toBeNull();
  });
});

describe("dayAndTime", () => {
  it("names today, tomorrow and yesterday, and dates anything further", () => {
    const now = AT;
    expect(dayAndTime(t, now, now)).toMatch(/^Today, /);
    expect(dayAndTime(t, now + 86_400, now)).toMatch(/^Tomorrow, /);
    expect(dayAndTime(t, now - 86_400, now)).toMatch(/^Yesterday, /);
    expect(dayAndTime(t, now + 3 * 86_400, now)).not.toMatch(/^(Today|Tomorrow|Yesterday)/);
  });

  it("reads inside a sentence when inline", () => {
    expect(dayAndTime(t, AT, AT, { inline: true })).toMatch(/^today at /);
    expect(dayAndTime(t, AT + 86_400, AT, { inline: true })).toMatch(/^tomorrow at /);
  });
});

describe("windowRange", () => {
  it("is a range, or the start alone when the window is one moment", () => {
    expect(windowRange(t, stop(1, { windowStart: AT, windowEnd: AT + 3600 }))).toContain(" – ");
    expect(windowRange(t, stop(1, { windowStart: AT }))).not.toContain(" – ");
    expect(windowRange(t, stop(1))).toBeNull();
  });
});

describe("shows and statuses", () => {
  it("reads the fields the customer's settings list", () => {
    const view = { fields: ["Status", "Eta"] } as unknown as PublicShipmentView;
    expect(shows(view, "Eta")).toBe(true);
    expect(shows(view, "Carrier")).toBe(false);
  });

  it("gives every customer status a lifecycle phase", () => {
    const attrs = publicStatusAttrs(t);
    expect(attrs.Planned.phase).toBe("queued");
    expect(attrs.InTransit.phase).toBe("active");
    expect(attrs.Delayed.phase).toBe("attention");
    expect(attrs.Delivered.phase).toBe("complete");
    expect(attrs.Canceled.phase).toBe("closed");
  });

  it("labels a hold by its reason, else by its kind", () => {
    expect(holdLabel(t, { type: "CustomerHold", label: "Appointment needed" })).toBe(
      "Appointment needed",
    );
    expect(holdLabel(t, { type: "CustomerHold" })).toBe("Waiting on you");
  });
});
