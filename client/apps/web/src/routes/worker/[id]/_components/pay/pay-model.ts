import type { EffectiveWorkerPayAssignment } from "@/lib/graphql/driver-settlement";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { formatNumber } from "@trenova/shared/i18n/format";

/** What one unit of a component's rate is paid for, after the amount ("$0.62 a mile"). */
export const PAY_METHOD_UNITS: Record<string, string> = defineLabels({
  PerLoadedMile: "a mile",
  PerEmptyMile: "a mile",
  PerTotalMile: "a mile",
  PercentOfRevenue: "of revenue",
  FlatPerShipment: "a shipment",
  PerStop: "a stop",
  PerHour: "an hour",
  PerDay: "a day",
  PerEvent: "an event",
});

/** A component's name when the profile gave it no description. */
export const PAY_COMPONENT_KINDS: Record<string, string> = defineLabels({
  Linehaul: "Linehaul",
  FuelSurcharge: "Fuel surcharge",
  StopPay: "Stop pay",
  Detention: "Detention",
  Layover: "Layover",
  Breakdown: "Breakdown",
  Tarp: "Tarp",
  Hazmat: "Hazmat",
  Bonus: "Bonus",
  Custom: "Other pay",
});

type ComponentLike = { kind: string; method: string; description?: string | null; rate: string };

export function payComponentName(component: Pick<ComponentLike, "kind" | "description">): string {
  return component.description?.trim() || (PAY_COMPONENT_KINDS[component.kind] ?? component.kind);
}

/** A rate as money (fractional cents kept, as per-mile rates have them) or as a percentage. */
export function formatPayRate(method: string, rate: string | number): string {
  const value = Number(rate);
  if (!Number.isFinite(value)) return String(rate);
  if (method === "PercentOfRevenue") {
    return formatNumber(value / 100, { style: "percent", maximumFractionDigits: 2 });
  }
  return formatNumber(value, {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 4,
  });
}

export type PayComponentLine = {
  id: string;
  name: string;
  method: string;
  profileRate: string;
  /** This driver's rate when it differs from the profile's; null pays the profile's. */
  override: string | null;
  bandCount: number;
};

/** The active components of the assignment's profile, each with the driver's override if any. */
export function payComponentLines(assignment: EffectiveWorkerPayAssignment): PayComponentLine[] {
  const overrides = new Map(
    (assignment.rateOverrides ?? []).map((override) => [override.componentId, override.rate]),
  );
  return (assignment.payProfile?.components ?? [])
    .filter((component) => component.isActive)
    .map((component) => {
      const override = overrides.get(component.id);
      return {
        id: component.id,
        name: payComponentName(component),
        method: component.method,
        profileRate: component.rate,
        override: override != null && Number(override) !== Number(component.rate) ? override : null,
        bandCount: component.bands?.length ?? 0,
      };
    });
}

/** The driver's current overrides, as the change form's starting rates. */
export function overrideStart(
  assignment: EffectiveWorkerPayAssignment | null | undefined,
): Record<string, string> {
  return Object.fromEntries(
    (assignment?.rateOverrides ?? []).map((override) => [
      override.componentId,
      String(Number(override.rate)),
    ]),
  );
}
