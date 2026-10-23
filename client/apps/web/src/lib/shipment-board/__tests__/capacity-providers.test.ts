import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import type { CapacityMatch, CapacityUnit } from "@/lib/graphql/shipment-board";
import {
  capacityProvidersFor,
  carrierCapacityProvider,
  driverCapacityProvider,
} from "../capacity-providers";

const t = ((message: string | null | undefined, ...args: unknown[]) =>
  formatMessage("en", message ?? "", args)) as TranslateFn;

const unit = (overrides: Partial<CapacityUnit>): CapacityUnit =>
  ({
    id: "u1",
    kind: "Driver",
    name: "Maria Ortega",
    initials: "MO",
    group: "ReadyNow",
    freeAt: null,
    city: "Dallas, TX",
    unitLabel: "T-300",
    ring: null,
    badgeCount: null,
    ratePerMile: null,
    acceptancePercent: null,
    driveRemainingMs: 7 * 3_600_000 + 48 * 60_000,
    tractorId: "trc_1",
    ...overrides,
  }) as CapacityUnit;

describe("capacity providers", () => {
  it("covers with drivers, carriers or both, drivers first", () => {
    expect(capacityProvidersFor("asset").map((p) => p.kind)).toEqual(["Driver"]);
    expect(capacityProvidersFor("brokerage").map((p) => p.kind)).toEqual(["Carrier"]);
    expect(capacityProvidersFor("both").map((p) => p.kind)).toEqual(["Driver", "Carrier"]);
  });

  it("captions a driver by city, or by when they come free", () => {
    const format = () => "13:40";
    expect(driverCapacityProvider.caption(unit({}), t, format)).toBe("Dallas");
    expect(
      driverCapacityProvider.caption(unit({ freeAt: 1, group: "WithinTwoHours" }), t, format),
    ).toBe("free 13:40");
  });

  it("shows hours of service only when an ELD supplies them", () => {
    expect(driverCapacityProvider.popoverMeta(unit({}), t, true)).toBe(
      "T-300 · 7:48 HOS · Dallas, TX",
    );
    expect(driverCapacityProvider.popoverMeta(unit({}), t, false)).toBe("T-300 · Dallas, TX");
  });

  it("badges a carrier with its posted trucks", () => {
    const carrier = unit({
      kind: "Carrier",
      group: "TrucksPosted",
      badgeCount: 3,
      acceptancePercent: 91,
      unitLabel: "MC 104000",
    });
    expect(carrierCapacityProvider.badge(carrier)).toEqual({ show: true, count: 3 });
    expect(carrierCapacityProvider.badge({ ...carrier, badgeCount: 1 })).toEqual({
      show: true,
      count: null,
    });
    expect(carrierCapacityProvider.popoverMeta(carrier, t, false)).toBe(
      "MC 104000 · 91% accept · 3 trucks posted",
    );
  });

  it("says one truck in the singular and none when nothing is posted", () => {
    const carrier = unit({ kind: "Carrier", acceptancePercent: 91, unitLabel: "MC 104000" });
    expect(carrierCapacityProvider.popoverMeta({ ...carrier, badgeCount: 1 }, t, false)).toBe(
      "MC 104000 · 91% accept · 1 truck posted",
    );
    expect(carrierCapacityProvider.popoverMeta({ ...carrier, badgeCount: null }, t, false)).toBe(
      "MC 104000 · 91% accept · no trucks posted",
    );
  });

  it.each([
    [driverCapacityProvider, 1, 2, "driver ready for 2 uncovered loads"],
    [driverCapacityProvider, 18, 1, "drivers ready for 1 uncovered load"],
    [driverCapacityProvider, 0, 0, "drivers ready for 0 uncovered loads"],
    [carrierCapacityProvider, 1, 1, "carrier posting trucks for 1 untendered load"],
    [carrierCapacityProvider, 12, 30, "carriers posting trucks for 30 untendered loads"],
  ])("agrees the headline with both counts", (provider, available, open, expected) => {
    expect(provider.headline(available, open, t)).toBe(expected);
  });
});

const match = (overrides: Partial<CapacityMatch>): CapacityMatch =>
  ({
    shipmentId: "shp_1",
    moveId: "smv_1",
    proNumber: "S-1001",
    originCity: "Dallas, TX",
    destinationCity: "Tulsa, OK",
    pickupAt: 1,
    revenue: "2400",
    deadheadMiles: 42.4,
    quote: "1850",
    marginPercent: 22.9,
    fitPercent: 88,
    ...overrides,
  }) as CapacityMatch;

describe("a carrier match's detail", () => {
  it("states the quote and margin to a reader with the Margins group", () => {
    expect(
      carrierCapacityProvider.matchDetail(match({}), t, { pickup: "13:40", marginsVisible: true }),
    ).toBe("pickup 13:40 · quote $1,850.00 · margin 23%");
  });

  it("leaves out a quote and margin the load does not have", () => {
    expect(
      carrierCapacityProvider.matchDetail(match({ quote: null, marginPercent: null }), t, {
        pickup: "13:40",
        marginsVisible: true,
      }),
    ).toBe("pickup 13:40");
  });

  it("says the quote and margin are hidden from a reader without the group", () => {
    expect(
      carrierCapacityProvider.matchDetail(match({ quote: null, marginPercent: null }), t, {
        pickup: "13:40",
        marginsVisible: false,
      }),
    ).toBe("pickup 13:40 · quote Hidden · margin Hidden");
  });

  it("states a driver match's revenue whatever the reader's groups", () => {
    expect(
      driverCapacityProvider.matchDetail(match({}), t, { pickup: "13:40", marginsVisible: false }),
    ).toBe("pickup 13:40 · 42 mi out · $2,400.00");
  });
});
