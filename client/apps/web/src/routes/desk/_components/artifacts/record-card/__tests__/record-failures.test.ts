import type { RecordFailure } from "@/lib/graphql/record-failures";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { failureLine, failureStatus } from "../record-failures";

const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const NOW = 1_791_720_000;

function failure(overrides: Partial<RecordFailure> = {}): RecordFailure {
  return {
    id: "sf_1",
    number: "SF-1",
    type: "LateDelivery",
    status: "Open",
    stopType: "Delivery",
    lateMinutes: 45,
    actualArrival: NOW,
    shipment: { id: "shp_1", proNumber: "S2610-1366" },
    stop: {
      id: "stp_1",
      location: {
        id: "loc_1",
        name: "Tyson Foods DC",
        city: "Springdale",
        state: { abbreviation: "AR" },
      },
    },
    ...overrides,
  } as RecordFailure;
}

describe("failureLine", () => {
  it("names the load at a location, since the place is the card's own", () => {
    expect(failureLine(failure(), "location", t)).toMatch(
      /^Late delivery · S2610-1366 · 45m late · /u,
    );
  });

  it("names the place on a shipment, since the load is the card's own", () => {
    expect(failureLine(failure(), "shipment", t)).toMatch(
      /^Late delivery · Tyson Foods DC · 45m late · /u,
    );
  });

  it("falls back to the town for a place with no name, and leaves out what is not known", () => {
    const unnamed = failure({
      lateMinutes: 0,
      stop: {
        id: "stp_1",
        location: { id: "loc_1", name: " ", city: "Springdale", state: { abbreviation: "AR" } },
      },
    } as Partial<RecordFailure>);
    expect(failureLine(unnamed, "shipment", t)).toMatch(/^Late delivery · Springdale, AR · [A-Z]/u);
    expect(failureLine(failure({ shipment: null }), "location", t)).toMatch(
      /^Late delivery · 45m late · /u,
    );
    expect(failureLine(failure({ stop: null }), "shipment", t)).toMatch(
      /^Late delivery · 45m late · /u,
    );
  });
});

describe("failureStatus", () => {
  it("asks for a look at an open failure and settles a resolved one", () => {
    expect(failureStatus("Open")).toEqual({ label: "Open", state: "warn" });
    expect(failureStatus("Reviewed")).toEqual({ label: "Reviewed", state: null });
    expect(failureStatus("Resolved")).toEqual({ label: "Resolved", state: "ok" });
    expect(failureStatus("Voided")).toEqual({ label: "Voided", state: null });
  });
});
