import { choiceLabel, iftaFuelTypeChoices } from "@/lib/choices";
import type {
  DeskTractor,
  DeskTractorInspection,
  DeskTractorPosition,
} from "@/lib/graphql/tractor-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { worstFirst, type DueCheck } from "../../record-card/card-due";
import {
  equipmentFacts,
  equipmentStatusCheck,
  leaseCheck,
  registrationCheck,
} from "../../record-card/equipment";
import type { CardCheckState, CardFact } from "../../record-card/record-card";
import { cardDate, personName, type Person } from "../../record-card/record-card-values";

const METERS_PER_MILE = 1609.344;
/** A report newer than this is where the tractor is now; an older one is where it was. */
export const LIVE_WINDOW_SECONDS = 15 * 60;
/** Below this the tractor is standing, whatever the GPS jitter says. */
const MOVING_MPH = 3;

/** Who sits in the tractor and who owns it, each as a row the card lists. */
export type TractorPerson = { key: string; id: string; name: string; role: string };

export function tractorPeople(tractor: DeskTractor, t: TranslateFn): TractorPerson[] {
  const owner = tractor.ownershipType === "OwnerOperator" ? tractor.ownerWorker : null;
  const people: TractorPerson[] = [];
  const add = (key: string, person: (Person & { id: string }) | null | undefined, role: string) => {
    if (person && !people.some((listed) => listed.id === person.id)) {
      people.push({ key, id: person.id, name: personName(person) || t("Unnamed worker"), role });
    }
  };
  add(
    "primary",
    tractor.primaryWorker,
    owner && owner.id === tractor.primaryWorker?.id
      ? t("Owner-operator and primary driver")
      : t("Primary driver"),
  );
  add("secondary", tractor.secondaryWorker, t("Co-driver"));
  add("owner", owner, t("Owner-operator"));
  return people;
}

/**
 * What stands between the tractor and its next load, worst first: whether it
 * is in service, its registration, the lease on a leased tractor, and the
 * last driver inspection with any defects still open. `latest` is undefined
 * while the inspections are read, and null when there are none; the
 * inspection's check is left out until they are known.
 */
export function tractorChecks(
  tractor: DeskTractor,
  latest: DeskTractorInspection | null | undefined,
  now: number,
  t: TranslateFn,
): DueCheck[] {
  const checks: DueCheck[] = [];
  const status = equipmentStatusCheck(tractor.status, t);
  if (status) checks.push(status);
  checks.push(registrationCheck(tractor.registrationExpiry, now, t));
  const lease = leaseCheck(tractor, now, t);
  if (lease) checks.push(lease);

  if (latest === null) {
    checks.push({
      key: "inspection",
      state: "warn",
      title: t("Inspection"),
      detail: t("No driver inspection on record"),
    });
  } else if (latest) {
    const detail = t(
      "{0} · {1}",
      inspectionKind(latest.inspectionType, t),
      cardDate(latest.startedAt),
    );
    if (latest.unresolvedDefectCount > 0) {
      checks.push({
        key: "inspection",
        state: "bad",
        title: t(
          "{0, plural, one {# open defect} other {# open defects}}",
          latest.unresolvedDefectCount,
        ),
        detail,
      });
    } else if (isUnsafe(latest.safetyStatus)) {
      checks.push({
        key: "inspection",
        state: "bad",
        title: t("Last inspection found it unsafe"),
        detail,
      });
    } else {
      checks.push({ key: "inspection", state: "ok", title: t("Last inspection"), detail });
    }
  }

  return worstFirst(checks);
}

function isUnsafe(status: string): boolean {
  return status.trim().toLowerCase() === "unsafe";
}

/** A driver inspection's kind, as the provider reports it ("preTrip", "postTrip", …). */
export function inspectionKind(type: string, t: TranslateFn): string {
  switch (type.trim().toLowerCase()) {
    case "pretrip":
      return t("Pre-trip");
    case "posttrip":
      return t("Post-trip");
    case "mechanic":
      return t("Mechanic");
    case "":
    case "unspecified":
      return t("Inspection");
    default:
      return type;
  }
}

/** How one inspection reads in the card's list: clean, safe with defects fixed, or not. */
export type InspectionReading = { state: CardCheckState; text: string };

export function inspectionReading(
  inspection: DeskTractorInspection,
  t: TranslateFn,
): InspectionReading {
  if (inspection.unresolvedDefectCount > 0) {
    return {
      state: "bad",
      text: t(
        "{0, plural, one {# open defect} other {# open defects}}",
        inspection.unresolvedDefectCount,
      ),
    };
  }
  if (isUnsafe(inspection.safetyStatus)) {
    return { state: "bad", text: t("Unsafe") };
  }
  if (inspection.defectCount > 0) {
    return {
      state: "ok",
      text: t("{0, plural, one {# defect fixed} other {# defects fixed}}", inspection.defectCount),
    };
  }
  return { state: "ok", text: t("No defects") };
}

/** Whole miles from the meters a GPS unit counts. */
export function metersToMiles(meters: number): number {
  return Math.round(meters / METERS_PER_MILE);
}

const COMPASS = ["N", "NE", "E", "SE", "S", "SW", "W", "NW"] as const;

/** A heading in degrees as one of the eight compass points. */
export function compassPoint(degrees: number): (typeof COMPASS)[number] {
  const normalized = ((degrees % 360) + 360) % 360;
  return COMPASS[Math.round(normalized / 45) % COMPASS.length];
}

/** Where the tractor is, as its last report says it: now or a while ago, moving or standing. */
export type PositionReading = {
  live: boolean;
  /** "Moving 58 mph NE", "Idling", "Stopped". */
  motion: string;
  /** Seconds since the report. */
  age: number;
  place: string | null;
};

export function positionReading(
  position: DeskTractorPosition,
  now: number,
  t: TranslateFn,
): PositionReading {
  const age = Math.max(0, now - position.recordedAt);
  const live = age <= LIVE_WINDOW_SECONDS;
  const speed = Math.round(position.speedMph);
  const heading = compassPoint(position.headingDegrees);
  let motion: string;
  if (position.speedMph >= MOVING_MPH) {
    motion = live
      ? t("Moving {0} mph {1}", speed, heading)
      : t("Was moving {0} mph {1}", speed, heading);
  } else if (position.engineState === "Idle") {
    motion = live ? t("Idling") : t("Was idling");
  } else if (position.engineState === "On") {
    motion = live ? t("Standing, engine on") : t("Was standing, engine on");
  } else {
    motion = live ? t("Stopped") : t("Was stopped");
  }
  return { live, motion, age, place: position.formattedLocation?.trim() || null };
}

/** The facts the card lists about the tractor itself, in the order a dispatcher reads them. */
export function tractorFacts(tractor: DeskTractor, t: TranslateFn): CardFact[] {
  const fuel = choiceLabel(iftaFuelTypeChoices, tractor.fuelType);
  return equipmentFacts(
    tractor,
    tractor.state?.abbreviation,
    [
      {
        key: "fuel",
        label: t("Fuel"),
        value: tractor.iftaQualified ? t("{0} · IFTA", fuel) : t("{0} · not on IFTA", fuel),
      },
    ],
    t,
  );
}
