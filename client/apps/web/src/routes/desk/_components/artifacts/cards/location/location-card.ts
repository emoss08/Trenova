import { choiceLabel, stopTypeChoices } from "@/lib/choices";
import type {
  DeskLocation,
  LocationDwell,
  LocationPolicy,
  TimeWindow,
} from "@/lib/graphql/location-card";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { MapOutline } from "../../record-card/map-tiles";
import { zoomToFit } from "../../record-card/map-tiles";
import type { CardFact, CardStat, CardTone } from "../../record-card/record-card";
import { cardDate, money, waitText } from "../../record-card/record-card-values";

const DAY = 86_400;
const HOUR = 3_600;
/** How far back the card reads how long trucks waited. */
export const DWELL_WINDOW_DAYS = 90;
/** A stop that waits past this share of the stops scored is a dock worth warning about. */
const BREACH_WARN_SHARE = 0.25;
/** A 90th-percentile wait past this many minutes reads as long. */
const LONG_P90_MINUTES = 180;
/** How close the map reads a location with no fence to fit: the block it is on. */
export const LOCATION_ZOOM = 15;
const FENCE_ZOOM_MIN = 11;
const FENCE_ZOOM_MAX = 17;

/**
 * The window the dwell is read over, ending at the top of the hour, so every
 * card opened in the same hour shares one read.
 */
export function dwellWindow(now: number): TimeWindow {
  const to = Math.floor(now / HOUR) * HOUR;
  return { from: to - DWELL_WINDOW_DAYS * DAY, to };
}

/** The location's street address on one line, or "" when none is on file. */
export function streetLine(location: Pick<DeskLocation, "addressLine1" | "addressLine2">): string {
  return [location.addressLine1.trim(), location.addressLine2.trim()].filter(Boolean).join(", ");
}

/** Whether the location takes trucks, as the card's pill says it. */
export function locationStanding(
  status: DeskLocation["status"],
  t: TranslateFn,
): { label: string; tone: CardTone } {
  return status === "Active"
    ? { label: t("Active"), tone: "success" }
    : { label: t("Inactive"), tone: "muted" };
}

/** What the facility offers a driver, from its category: appointments, parking, overnight. */
export function facilityTerms(
  category: DeskLocation["locationCategory"],
  t: TranslateFn,
): string[] {
  if (!category) return [];
  return [
    category.requiresAppointment ? t("Appointment required") : t("No appointment needed"),
    category.hasSecureParking ? t("Secure parking") : "",
    category.allowsOvernight ? t("Overnight allowed") : t("No overnight"),
    category.hasRestroom ? t("Restroom") : "",
  ].filter(Boolean);
}

/** One thing the facility offers or asks, and whether it does. */
export type Amenity = { key: string; label: string; on: boolean };

/**
 * What the facility offers a driver and what it asks of them, each said
 * whether it holds or not: an appointment, secure parking, an overnight stay
 * and a restroom.
 */
export function amenities(category: DeskLocation["locationCategory"], t: TranslateFn): Amenity[] {
  if (!category) return [];
  return [
    {
      key: "appointment",
      label: category.requiresAppointment ? t("Appointment required") : t("No appointment needed"),
      on: !category.requiresAppointment,
    },
    {
      key: "parking",
      label: category.hasSecureParking ? t("Secure parking") : t("No secure parking"),
      on: category.hasSecureParking,
    },
    {
      key: "overnight",
      label: category.allowsOvernight ? t("Overnight allowed") : t("No overnight"),
      on: category.allowsOvernight,
    },
    {
      key: "restroom",
      label: category.hasRestroom ? t("Restroom") : t("No restroom"),
      on: category.hasRestroom,
    },
  ];
}

/** One mark on the dwell ruler: what it measures, how long, and where it falls. */
export type DwellMark = {
  key: "median" | "avg" | "p90";
  label: string;
  minutes: number;
  share: number;
};

/**
 * The dwell ruler's scale and marks: the median, the average and the 90th
 * percentile, each placed along a scale that runs a little past the longest so
 * its mark never sits on the edge.
 */
export function dwellScale(
  dwell: Pick<LocationDwell, "avgDwellMinutes" | "medianDwellMinutes" | "p90DwellMinutes">,
  t: TranslateFn,
): { maxMinutes: number; marks: DwellMark[] } {
  const longest = Math.max(dwell.avgDwellMinutes, dwell.medianDwellMinutes, dwell.p90DwellMinutes);
  const maxMinutes = Math.max(60, Math.ceil((longest * 1.15) / 30) * 30);
  const mark = (key: DwellMark["key"], label: string, minutes: number): DwellMark => ({
    key,
    label,
    minutes,
    share: Math.min(1, Math.max(0, minutes / maxMinutes)),
  });
  return {
    maxMinutes,
    marks: [
      mark("median", t("Median"), dwell.medianDwellMinutes),
      mark("avg", t("Average"), dwell.avgDwellMinutes),
      mark("p90", t("90th percentile"), dwell.p90DwellMinutes),
    ],
  };
}

/**
 * The location's geofence as the map draws it: a circle round a point, or the
 * outline drawn for it. A fence with nothing to draw (an automatic one the
 * server sizes, a drawing with fewer than three corners) is null.
 */
export function geofenceOutline(
  location: Pick<DeskLocation, "geofenceType" | "geofenceRadiusMeters" | "geofenceVertices">,
): MapOutline | null {
  switch (location.geofenceType) {
    case "auto":
    case "circle":
      return location.geofenceRadiusMeters && location.geofenceRadiusMeters > 0
        ? { kind: "circle", radiusMeters: location.geofenceRadiusMeters }
        : null;
    case "rectangle":
    case "draw":
      return location.geofenceVertices.length >= 3
        ? { kind: "polygon", vertices: location.geofenceVertices }
        : null;
  }
}

/** How the geofence reads beside the map: "Circle fence, 250 m", "Drawn fence, 6 corners". */
export function geofenceLine(
  location: Pick<DeskLocation, "geofenceType" | "geofenceRadiusMeters" | "geofenceVertices">,
  t: TranslateFn,
): string {
  const radius = location.geofenceRadiusMeters;
  const meters = radius && radius > 0 ? formatNumber(Math.round(radius)) : null;
  switch (location.geofenceType) {
    case "auto":
      return meters ? t("Automatic fence, {0} m", meters) : t("Automatic fence");
    case "circle":
      return meters ? t("Circle fence, {0} m", meters) : t("Circle fence");
    case "rectangle":
      return t("Rectangle fence");
    case "draw":
      return t(
        "Drawn fence, {0, plural, one {# corner} other {# corners}}",
        location.geofenceVertices.length,
      );
  }
}

/** How close the map reads: as close as the fence allows, or the block the location is on. */
export function locationZoom(
  center: { latitude: number; longitude: number },
  outline: MapOutline | null,
  size: number,
): number {
  return outline
    ? zoomToFit({ center, outline, size, min: FENCE_ZOOM_MIN, max: FENCE_ZOOM_MAX })
    : LOCATION_ZOOM;
}

/** The wait figures the card lists for the location, the long tail marked when it runs long. */
export function dwellStats(dwell: LocationDwell, t: TranslateFn): CardStat[] {
  const share = dwell.stopCount > 0 ? dwell.breachCount / dwell.stopCount : 0;
  return [
    { key: "avg", label: t("Average"), value: waitText(dwell.avgDwellMinutes) },
    { key: "median", label: t("Median"), value: waitText(dwell.medianDwellMinutes) },
    {
      key: "p90",
      label: t("90th percentile"),
      value: waitText(dwell.p90DwellMinutes),
      state: dwell.p90DwellMinutes > LONG_P90_MINUTES ? "warn" : "ok",
    },
    {
      key: "breach",
      label: t("Past free time"),
      value: t("{0} of {1}", formatNumber(dwell.breachCount), formatNumber(dwell.stopCount)),
      state: share >= BREACH_WARN_SHARE ? "warn" : "ok",
    },
  ];
}

/** What the detention at the location came to: billed, waived, disputed. */
export function dwellMoneyLine(dwell: LocationDwell, t: TranslateFn): string {
  return [
    t(
      "{0} detention billed on {1, plural, one {# stop} other {# stops}}",
      money(dwell.billedAmount),
      dwell.stopCount,
    ),
    Number(dwell.waivedAmount) > 0 ? t("{0} waived", money(dwell.waivedAmount)) : "",
    dwell.disputeCount > 0
      ? t("{0, plural, one {# dispute} other {# disputes}}", dwell.disputeCount)
      : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** The free time a policy grants, as a dispatcher reads it: "2h at pickup · 3h at delivery". */
export function policyFreeTime(policy: LocationPolicy, t: TranslateFn): string {
  const pickup = policy.pickupFreeMinutes ?? null;
  const delivery = policy.deliveryFreeMinutes ?? null;
  if (pickup !== null || delivery !== null) {
    return [
      t("{0} at pickup", waitText(pickup ?? policy.billingFreeMinutes)),
      t("{0} at delivery", waitText(delivery ?? policy.billingFreeMinutes)),
    ].join(" · ");
  }
  return t("{0} free", waitText(policy.billingFreeMinutes));
}

/** What a policy applies to beyond the location: one customer, some stop types, appointments, a window. */
export function policyScope(policy: LocationPolicy, t: TranslateFn): string {
  return [
    policy.customerId ? t("One customer") : t("Every customer"),
    policy.stopTypes.length > 0
      ? policy.stopTypes.map((type) => choiceLabel(stopTypeChoices, type)).join(", ")
      : "",
    policy.appointmentStopsOnly ? t("Appointment stops only") : "",
    policy.effectiveEndDate ? t("Through {0}", cardDate(policy.effectiveEndDate)) : "",
  ]
    .filter(Boolean)
    .join(" · ");
}

/** The facts the card lists about the location, in the order the office reads them. */
export function locationFacts(location: DeskLocation, t: TranslateFn): CardFact[] {
  const category = location.locationCategory ?? null;
  return [
    { key: "code", label: t("Code"), value: location.code },
    { key: "category", label: t("Category"), value: category?.name ?? t("None") },
    {
      key: "timezone",
      label: t("Timezone"),
      value: location.timezone.trim() || t("None on file"),
    },
    {
      key: "coordinates",
      label: t("Coordinates"),
      value:
        location.latitude !== null &&
        location.latitude !== undefined &&
        location.longitude !== null &&
        location.longitude !== undefined
          ? t("{0}, {1}", location.latitude.toFixed(5), location.longitude.toFixed(5))
          : t("Not geocoded"),
    },
    { key: "added", label: t("Added"), value: cardDate(location.createdAt) },
    {
      key: "description",
      label: t("Notes"),
      value: location.description.trim() || t("None"),
    },
  ];
}
