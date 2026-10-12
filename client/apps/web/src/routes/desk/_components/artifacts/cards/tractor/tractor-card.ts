import { choiceLabel, equipmentOwnershipChoices, iftaFuelTypeChoices } from "@/lib/choices";
import type {
  DeskTractor,
  DeskTractorInspection,
  DeskTractorPosition,
} from "@/lib/graphql/tractor-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardCheckState, CardTone } from "../../record-card/record-card";
import { cardDate } from "../../record-card/record-card-values";

const DAY_SECONDS = 86_400;
const METERS_PER_MILE = 1609.344;
/** A report newer than this is where the tractor is now; an older one is where it was. */
export const LIVE_WINDOW_SECONDS = 15 * 60;
/** Below this the tractor is standing, whatever the GPS jitter says. */
const MOVING_MPH = 3;
/** Registration expiring inside this many days is worth a look. */
export const REGISTRATION_NOTICE_DAYS = 30;
/** A lease ending inside this many days is worth a look. */
export const LEASE_NOTICE_DAYS = 60;

type TractorStatus = DeskTractor["status"];

/** How a tractor's status reads on its card: in service, held, or gone. */
export type TractorStanding = { label: string; tone: CardTone; mark: "dot" | "check" | "pause" };

export function tractorStanding(status: TractorStatus, t: TranslateFn): TractorStanding {
  switch (status) {
    case "Available":
      return { label: t("Available"), tone: "success", mark: "dot" };
    case "AtMaintenance":
      return { label: t("At maintenance"), tone: "warn", mark: "pause" };
    case "OutOfService":
      return { label: t("Out of service"), tone: "warn", mark: "pause" };
    case "Sold":
      return { label: t("Sold"), tone: "muted", mark: "dot" };
  }
}

/** The tractor by what it is, "2022 Freightliner Cascadia"; its code when nothing is on file. */
export function tractorName(tractor: Pick<DeskTractor, "year" | "make" | "model" | "code">) {
  const name = [tractor.year ? String(tractor.year) : "", tractor.make, tractor.model]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(" ");
  return name || tractor.code;
}

type Person = { firstName: string; lastName: string } | null | undefined;

/** A driver's name as the card writes it; empty for none. */
export function personName(person: Person): string {
  if (!person) return "";
  return [person.firstName, person.lastName]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(" ");
}

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

/** When something falls due, against now: past, close, or comfortably off. */
export type DueReading = { state: CardCheckState; days: number };

export function dueReading(at: number, now: number, noticeDays: number): DueReading {
  const days = Math.ceil((at - now) / DAY_SECONDS);
  if (at <= now) return { state: "bad", days };
  if (days <= noticeDays) return { state: "warn", days };
  return { state: "ok", days };
}

/** One thing the card checks on the tractor, drawn as a `CardCheck`. */
export type TractorCheck = {
  key: string;
  state: CardCheckState;
  title: string;
  detail: string;
};

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
): TractorCheck[] {
  const checks: TractorCheck[] = [];

  if (tractor.status === "OutOfService") {
    checks.push({
      key: "status",
      state: "bad",
      title: t("Out of service"),
      detail: t("It cannot be dispatched until it is put back in service."),
    });
  } else if (tractor.status === "AtMaintenance") {
    checks.push({
      key: "status",
      state: "warn",
      title: t("At maintenance"),
      detail: t("It cannot be dispatched until maintenance releases it."),
    });
  }

  const expiry = tractor.registrationExpiry;
  if (expiry === null || expiry === undefined) {
    checks.push({
      key: "registration",
      state: "warn",
      title: t("Registration"),
      detail: t("No expiry on file"),
    });
  } else {
    const due = dueReading(expiry, now, REGISTRATION_NOTICE_DAYS);
    checks.push({
      key: "registration",
      state: due.state,
      title:
        due.state === "bad"
          ? t("Registration expired")
          : due.state === "warn"
            ? t("Registration expires in {0, plural, one {# day} other {# days}}", due.days)
            : t("Registration"),
      detail:
        due.state === "bad"
          ? t("Expired {0}", cardDate(expiry))
          : t("Through {0}", cardDate(expiry)),
    });
  }

  if (tractor.ownershipType === "Leased" && tractor.leaseEndDate) {
    const due = dueReading(tractor.leaseEndDate, now, LEASE_NOTICE_DAYS);
    const lessor = tractor.lessorName.trim();
    checks.push({
      key: "lease",
      state: due.state,
      title:
        due.state === "bad"
          ? t("Lease ended")
          : due.state === "warn"
            ? t("Lease ends in {0, plural, one {# day} other {# days}}", due.days)
            : t("Lease"),
      detail: lessor
        ? due.state === "bad"
          ? t("Ended {0} · {1}", cardDate(tractor.leaseEndDate), lessor)
          : t("Through {0} · {1}", cardDate(tractor.leaseEndDate), lessor)
        : due.state === "bad"
          ? t("Ended {0}", cardDate(tractor.leaseEndDate))
          : t("Through {0}", cardDate(tractor.leaseEndDate)),
    });
  }

  if (latest === null) {
    checks.push({
      key: "inspection",
      state: "warn",
      title: t("Inspection"),
      detail: t("No driver inspection on record"),
    });
  } else if (latest) {
    const kind = inspectionKind(latest.inspectionType, t);
    const when = cardDate(latest.startedAt);
    if (latest.unresolvedDefectCount > 0) {
      checks.push({
        key: "inspection",
        state: "bad",
        title: t(
          "{0, plural, one {# open defect} other {# open defects}}",
          latest.unresolvedDefectCount,
        ),
        detail: t("{0} · {1}", kind, when),
      });
    } else if (isUnsafe(latest.safetyStatus)) {
      checks.push({
        key: "inspection",
        state: "bad",
        title: t("Last inspection found it unsafe"),
        detail: t("{0} · {1}", kind, when),
      });
    } else {
      checks.push({
        key: "inspection",
        state: "ok",
        title: t("Last inspection"),
        detail: t("{0} · {1}", kind, when),
      });
    }
  }

  const rank: Record<CardCheckState, number> = { bad: 0, warn: 1, ok: 2 };
  return checks
    .map((check, index) => ({ check, index }))
    .sort((a, b) => rank[a.check.state] - rank[b.check.state] || a.index - b.index)
    .map(({ check }) => check);
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
export type TractorFact = { key: string; label: string; value: string };

export function tractorFacts(tractor: DeskTractor, t: TranslateFn): TractorFact[] {
  const plate = tractor.licensePlateNumber.trim();
  const plateState = tractor.state?.abbreviation ?? "";
  const ownership = choiceLabel(equipmentOwnershipChoices, tractor.ownershipType);
  const lessor = tractor.lessorName.trim();
  const lease = tractor.leaseReference.trim();
  return [
    {
      key: "plate",
      label: t("Plate"),
      value: plate ? [plate, plateState].filter(Boolean).join(" · ") : t("None on file"),
    },
    {
      key: "registration",
      label: t("Registration number"),
      value: tractor.registrationNumber.trim() || t("None on file"),
    },
    { key: "vin", label: t("VIN"), value: tractor.vin.trim() || t("None on file") },
    {
      key: "manufacturer",
      label: t("Manufacturer"),
      value: tractor.equipmentManufacturer?.name ?? t("None on file"),
    },
    {
      key: "fuel",
      label: t("Fuel"),
      value: tractor.iftaQualified
        ? t("{0} · IFTA", choiceLabel(iftaFuelTypeChoices, tractor.fuelType))
        : t("{0} · not on IFTA", choiceLabel(iftaFuelTypeChoices, tractor.fuelType)),
    },
    {
      key: "ownership",
      label: t("Ownership"),
      value:
        tractor.ownershipType === "Leased" && (lessor || lease)
          ? [ownership, lessor, lease].filter(Boolean).join(" · ")
          : ownership,
    },
    {
      key: "external",
      label: t("External ID"),
      value: tractor.externalId.trim() || t("None"),
    },
    { key: "since", label: t("In the fleet since"), value: cardDate(tractor.createdAt) },
  ];
}
