import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import type { PublicShipmentView } from "@trenova/shared/types/shipment-tracking";
import { describe, expect, it } from "vitest";
import {
  holdLabel,
  publicStatusAttrs,
  shows,
  stopPlace,
  stopStanding,
  stopWindow,
} from "../shipment-tracking";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const AT = 1_800_000_000;

describe("stopStanding", () => {
  it("prefers what happened over what was expected", () => {
    expect(
      stopStanding(t, {
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        arrivedAt: AT,
        departedAt: AT + 600,
        estimatedArrival: AT - 60,
      }),
    ).toBe(`Left ${formatUnixDateTime(AT + 600)}`);
    expect(
      stopStanding(t, {
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        arrivedAt: AT,
      }),
    ).toBe(`Arrived ${formatUnixDateTime(AT)}`);
  });

  it("says a late stop is now expected later", () => {
    expect(
      stopStanding(t, {
        sequence: 2,
        type: "Delivery",
        city: "Memphis",
        stateCode: "TN",
        estimatedArrival: AT,
        running: "Late",
      }),
    ).toBe(`Now expected ${formatUnixDateTime(AT)}`);
    expect(
      stopStanding(t, {
        sequence: 2,
        type: "Delivery",
        city: "Memphis",
        stateCode: "TN",
        estimatedArrival: AT,
        running: "OnTime",
      }),
    ).toBe(`Expected ${formatUnixDateTime(AT)}`);
  });

  it("says nothing about times the customer is not shown", () => {
    expect(
      stopStanding(t, { sequence: 2, type: "Delivery", city: "Memphis", stateCode: "TN" }),
    ).toBeNull();
  });
});

describe("stopWindow", () => {
  it("spells the appointment out only when windows are shown", () => {
    expect(
      stopWindow(t, { sequence: 1, type: "Pickup", city: "Dallas", stateCode: "TX" }),
    ).toBeNull();
    expect(
      stopWindow(t, {
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        windowStart: AT,
        windowEnd: AT + 3600,
      }),
    ).toBe(`Appointment ${formatUnixDateTime(AT)} to ${formatUnixDateTime(AT + 3600)}`);
  });
});

describe("stopPlace", () => {
  it("names the location only when the view carries it", () => {
    expect(stopPlace({ sequence: 1, type: "Pickup", city: "Dallas", stateCode: "TX" })).toBe(
      "Dallas, TX",
    );
    expect(
      stopPlace({
        sequence: 1,
        type: "Pickup",
        city: "Dallas",
        stateCode: "TX",
        locationName: "North DC",
      }),
    ).toBe("North DC · Dallas, TX");
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
