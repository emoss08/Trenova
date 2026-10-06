import type { CapacityGroup, CapacityUnitKind } from "@trenova/graphql/generated/graphql";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { formatCurrency, formatPerMile } from "@trenova/shared/lib/utils";
import type { CapacityMatch, CapacityUnit } from "@/lib/graphql/shipment-board";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * What differs between covering a load with a driver and with a carrier.
 * The strip, its popover and the dock render any unit through one of these,
 * so neither cares which kind it is drawing.
 */
export type CapacityProvider = {
  kind: CapacityUnitKind;
  tabLabel: string;
  avatarShape: "circle" | "square";
  groups: readonly CapacityGroup[];
  groupLabel: (group: CapacityGroup, t: TranslateFn) => string;
  /** The line under the avatar. */
  caption: (unit: CapacityUnit, t: TranslateFn, formatTime: (unix: number) => string) => string;
  popoverMeta: (unit: CapacityUnit, t: TranslateFn, hos: boolean) => string;
  matchesTitle: (unit: CapacityUnit, t: TranslateFn) => string;
  matchDetail: (match: CapacityMatch, t: TranslateFn, pickup: string) => string;
  commitLabel: (t: TranslateFn) => string;
  ringHint: (t: TranslateFn, hos: boolean) => string;
  /** Whether the unit's dot says it is ready now, and the count drawn on it. */
  badge: (unit: CapacityUnit) => { show: boolean; count: number | null };
};

const firstName = (name: string) => name.split(" ")[0] ?? name;

export const driverCapacityProvider: CapacityProvider = {
  kind: "Driver",
  tabLabel: "Drivers",
  avatarShape: "circle",
  groups: ["ReadyNow", "WithinTwoHours"],
  groupLabel: (group, t) => (group === "ReadyNow" ? t("Ready now") : t("Within 2h")),
  caption: (unit, t, formatTime) =>
    unit.freeAt ? t("free {0}", formatTime(unit.freeAt)) : (unit.city?.split(",")[0] ?? ""),
  popoverMeta: (unit, t, hos) =>
    [
      unit.unitLabel,
      hos && unit.driveRemainingMs != null
        ? t("{0} HOS", formatClockDurationMs(unit.driveRemainingMs))
        : null,
      unit.city,
    ]
      .filter(Boolean)
      .join(" · "),
  matchesTitle: (unit, t) => t("Best load for {0}", firstName(unit.name)),
  matchDetail: (match, t, pickup) =>
    [
      t("pickup {0}", pickup),
      match.deadheadMiles != null ? t("{0} mi out", Math.round(match.deadheadMiles)) : null,
      formatCurrency(Number(match.revenue)),
    ]
      .filter(Boolean)
      .join(" · "),
  commitLabel: (t) => t("Assign"),
  ringHint: (t, hos) =>
    hos
      ? t("Ring shows hours of service left · click a driver to see their best load")
      : t("Click a driver to see their best load"),
  badge: (unit) => ({ show: unit.group === "ReadyNow", count: null }),
};

export const carrierCapacityProvider: CapacityProvider = {
  kind: "Carrier",
  tabLabel: "Carriers",
  avatarShape: "square",
  groups: ["TrucksPosted", "UsuallyAccept"],
  groupLabel: (group, t) => (group === "TrucksPosted" ? t("Trucks posted") : t("Usually accept")),
  caption: (unit) => (unit.ratePerMile ? formatPerMile(Number(unit.ratePerMile)) : ""),
  popoverMeta: (unit, t) =>
    [
      unit.unitLabel,
      unit.acceptancePercent != null ? t("{0}% accept", Math.round(unit.acceptancePercent)) : null,
      unit.badgeCount
        ? unit.badgeCount === 1
          ? t("1 truck posted")
          : t("{0} trucks posted", unit.badgeCount)
        : t("no trucks posted"),
    ]
      .filter(Boolean)
      .join(" · "),
  matchesTitle: (_unit, t) => t("Best loads to tender"),
  matchDetail: (match, t, pickup) =>
    [
      t("pickup {0}", pickup),
      match.quote ? t("quote {0}", formatCurrency(Number(match.quote))) : null,
      match.marginPercent != null ? t("margin {0}%", Math.round(match.marginPercent)) : null,
    ]
      .filter(Boolean)
      .join(" · "),
  commitLabel: (t) => t("Tender"),
  ringHint: (t) => t("Ring shows acceptance rate on your lanes · click a carrier to tender"),
  badge: (unit) => ({
    show: unit.group === "TrucksPosted",
    count: unit.badgeCount && unit.badgeCount > 1 ? unit.badgeCount : null,
  }),
};

export const CAPACITY_PROVIDERS: Record<CapacityUnitKind, CapacityProvider> = {
  Driver: driverCapacityProvider,
  Carrier: carrierCapacityProvider,
};

/** The providers an organization covers loads with, in the order the strip offers them. */
export function capacityProvidersFor(operationType: "asset" | "brokerage" | "both"): CapacityProvider[] {
  if (operationType === "asset") return [driverCapacityProvider];
  if (operationType === "brokerage") return [carrierCapacityProvider];
  return [driverCapacityProvider, carrierCapacityProvider];
}
