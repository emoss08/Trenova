import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import type { CapacityUnit } from "@/lib/graphql/shipment-board";
import {
  capacityProvidersFor,
  carrierCapacityProvider,
  driverCapacityProvider,
} from "../capacity-providers";

const t = ((message: string | null | undefined, ...args: unknown[]) =>
  args.reduce<string>(
    (text, arg, i) => text.replace(`{${i}}`, String(arg)),
    message ?? "",
  )) as TranslateFn;

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
    expect(driverCapacityProvider.caption(unit({ freeAt: 1, group: "WithinTwoHours" }), t, format)).toBe(
      "free 13:40",
    );
  });

  it("shows hours of service only when an ELD supplies them", () => {
    expect(driverCapacityProvider.popoverMeta(unit({}), t, true)).toBe("T-300 · 7:48 HOS · Dallas, TX");
    expect(driverCapacityProvider.popoverMeta(unit({}), t, false)).toBe("T-300 · Dallas, TX");
  });

  it("badges a carrier with its posted trucks", () => {
    const carrier = unit({ kind: "Carrier", group: "TrucksPosted", badgeCount: 3, acceptancePercent: 91, unitLabel: "MC 104000" });
    expect(carrierCapacityProvider.badge(carrier)).toEqual({ show: true, count: 3 });
    expect(carrierCapacityProvider.badge({ ...carrier, badgeCount: 1 })).toEqual({ show: true, count: null });
    expect(carrierCapacityProvider.popoverMeta(carrier, t, false)).toBe("MC 104000 · 91% accept · 3 trucks posted");
  });
});
