import type {
  AssessedJurisdiction,
  EscortRole,
  Exceedance,
  PermitAssessment,
  PermitRequirement,
  RestrictionKind,
} from "../types/permit";

export const ESCORT_ROLE_LABELS: Record<EscortRole, string> = {
  Front: "Front escort",
  Rear: "Rear escort",
  Police: "Police escort",
  RouteSurvey: "Route survey",
};

export const RESTRICTION_LABELS: Record<RestrictionKind, string> = {
  DaylightOnly: "Daylight movement only",
  RushHour: "Rush hour restriction",
  Weekend: "Weekend restriction",
  Holiday: "Holiday restriction",
  NightOnly: "Night movement only",
};

/**
 * Renders a decimal foot measurement the way an operator says it out loud.
 * "12' 6"" is the number that gets read over the phone to a permit office;
 * "12.5 ft" is the number a spreadsheet stores. Showing the second where the
 * first belongs is a small thing that makes a tool feel foreign to the people
 * who have to use it under time pressure.
 */
export function formatFeetInches(value: number): string {
  if (!Number.isFinite(value)) return "—";

  const totalInches = Math.round(Math.abs(value) * 12);
  const feet = Math.floor(totalInches / 12);
  const inches = totalInches % 12;
  const body = inches === 0 ? `${feet}'` : `${feet}' ${inches}"`;

  return value < 0 ? `-${body}` : body;
}

export function formatPounds(value: number): string {
  if (!Number.isFinite(value)) return "—";
  return `${Math.round(value).toLocaleString()} lbs`;
}

export type CargoDimensions = {
  lengthFeet?: number | null;
  widthFeet?: number | null;
  heightFeet?: number | null;
};

/**
 * Renders one cargo line's dimensions as L × W × H. A missing side is shown as
 * "?" rather than collapsing the whole line to a dash: a line with a length and
 * no width is a half-entered line, and hiding that difference is how a load
 * reaches dispatch without the number a permit is derived from.
 */
export function describeCommodityDimensions(cargo: CargoDimensions | null | undefined): string {
  const sides = [cargo?.lengthFeet, cargo?.widthFeet, cargo?.heightFeet];

  if (sides.every((side) => !side)) return "—";

  return sides.map((side) => (side ? formatFeetInches(side) : "?")).join(" × ");
}

export type DimensionKey = "width" | "height" | "length" | "weight";

export type DimensionRow = {
  key: DimensionKey;
  label: string;
  unit: "ft" | "lbs";
  actual: number;
  /** Tightest limit across every jurisdiction on the route, null when unknown. */
  limit: number | null;
  /** Limit minus actual at the tightest jurisdiction; negative means over. */
  headroom: number | null;
  /** The jurisdiction that sets the tightest limit for this dimension. */
  governingStateCode: string | null;
  exceeded: boolean;
};

type DimensionSpec = {
  key: DimensionKey;
  label: string;
  unit: "ft" | "lbs";
  actual: (assessment: PermitAssessment) => number;
  limit: (jurisdiction: AssessedJurisdiction) => number;
  headroom: (jurisdiction: AssessedJurisdiction) => number;
};

const DIMENSION_SPECS: DimensionSpec[] = [
  {
    key: "width",
    label: "Width",
    unit: "ft",
    actual: (a) => a.measurements.widthFeet,
    limit: (j) => j.maxWidthFeet,
    headroom: (j) => j.widthHeadroomFeet,
  },
  {
    key: "height",
    label: "Overall height",
    unit: "ft",
    actual: (a) => a.measurements.heightFeet,
    limit: (j) => j.maxHeightFeet,
    headroom: (j) => j.heightHeadroomFeet,
  },
  {
    key: "length",
    label: "Length",
    unit: "ft",
    actual: (a) => a.measurements.lengthFeet,
    limit: (j) => j.maxLengthFeet,
    headroom: (j) => j.lengthHeadroomFeet,
  },
  {
    key: "weight",
    label: "Gross weight",
    unit: "lbs",
    actual: (a) => a.measurements.weightPounds,
    limit: (j) => j.maxWeightPounds,
    headroom: (j) => j.weightHeadroomPounds,
  },
];

/**
 * Reduces the route to one row per dimension, each reporting the tightest
 * jurisdiction rather than a national average. A load can be legal in eight
 * states and over in the ninth, and the ninth is the only one that matters —
 * so the row names it.
 */
export function dimensionRows(assessment: PermitAssessment | null | undefined): DimensionRow[] {
  if (!assessment) return [];

  const jurisdictions = assessment.jurisdictions ?? [];

  return DIMENSION_SPECS.map((spec) => {
    let tightest: AssessedJurisdiction | null = null;

    for (const jurisdiction of jurisdictions) {
      if (tightest === null || spec.headroom(jurisdiction) < spec.headroom(tightest)) {
        tightest = jurisdiction;
      }
    }

    const headroom = tightest ? spec.headroom(tightest) : null;

    return {
      key: spec.key,
      label: spec.label,
      unit: spec.unit,
      actual: spec.actual(assessment),
      limit: tightest ? spec.limit(tightest) : null,
      headroom,
      governingStateCode: tightest?.stateCode ?? null,
      exceeded: headroom !== null && headroom < 0,
    };
  });
}

export function formatMeasurement(value: number, unit: "ft" | "lbs"): string {
  return unit === "lbs" ? formatPounds(value) : formatFeetInches(value);
}

/** Share of the tightest limit at which a still-legal dimension starts to warn. */
export const NEAR_LIMIT_RATIO = 0.9;

export type DimensionMeterTone = "ok" | "near" | "over";

export type DimensionMeter = {
  /** Share of the tightest limit consumed, clamped to [0, 100] for bar width. */
  percentOfLimit: number;
  tone: DimensionMeterTone;
};

/**
 * How full a dimension runs against the tightest limit on the route, for
 * rendering as a capacity bar. A load at 94% of the width limit is legal but
 * one re-measure away from not being, and a table of raw numbers hides that;
 * the near band exists so the bar says it before the permit engine has to.
 * Returns null when no jurisdiction resolved a limit — an absent limit is not
 * an empty bar, it is no bar.
 */
export function dimensionMeter(row: DimensionRow): DimensionMeter | null {
  if (row.limit === null || row.limit <= 0) return null;

  const ratio = row.actual / row.limit;

  return {
    percentOfLimit: Math.min(Math.max(ratio * 100, 0), 100),
    tone: row.exceeded ? "over" : ratio >= NEAR_LIMIT_RATIO ? "near" : "ok",
  };
}

export function requirementStateCode(requirement: PermitRequirement): string {
  return requirement.provenance?.stateCode ?? "";
}

/**
 * One exceedance as an operator reads it: the dimension and the amount over,
 * in the unit that dimension is measured in.
 */
export function describeExceedance(exceedance: Exceedance): string {
  const over =
    exceedance.trigger === "Weight"
      ? formatPounds(exceedance.overBy)
      : formatFeetInches(exceedance.overBy);

  return `${exceedance.trigger.toLowerCase()} ${over} over`;
}

/**
 * Renders a requirement as an operator reads it: which state, which dimension
 * pushed it over, and by how much. Mirrors Requirement.Describe on the server
 * so the same sentence appears wherever the requirement is surfaced.
 */
export function describeRequirement(requirement: PermitRequirement): string {
  const code = requirementStateCode(requirement);
  const exceedances = requirement.exceedances ?? [];

  if (exceedances.length === 0) {
    return `${code} permit required`;
  }

  return `${code} permit required (${exceedances.map(describeExceedance).join(", ")})`;
}

export type EscortSummaryEntry = {
  role: EscortRole;
  label: string;
  stateCodes: string[];
};

/**
 * Groups escorts by role across the whole route. A front escort that runs
 * Georgia through Nebraska is one vehicle hired for the trip, so it is listed
 * once with the states that demand it rather than three times as three hires.
 */
export function escortSummary(
  requirements: PermitRequirement[] | null | undefined,
): EscortSummaryEntry[] {
  const byRole = new Map<EscortRole, Set<string>>();

  for (const requirement of requirements ?? []) {
    const code = requirementStateCode(requirement);

    for (const escort of requirement.escorts ?? []) {
      const codes = byRole.get(escort.role) ?? new Set<string>();
      if (code) codes.add(code);
      byRole.set(escort.role, codes);
    }
  }

  return [...byRole.entries()].map(([role, codes]) => ({
    role,
    label: ESCORT_ROLE_LABELS[role] ?? role,
    stateCodes: [...codes],
  }));
}

export type RestrictionSummaryEntry = {
  kind: RestrictionKind;
  label: string;
  stateCodes: string[];
};

/**
 * Collects movement restrictions only from jurisdictions that actually require
 * a permit. Every state restricts oversize movement somehow; listing the ones a
 * legal load never triggers would bury the two that matter.
 */
export function restrictionSummary(
  jurisdictions: AssessedJurisdiction[] | null | undefined,
): RestrictionSummaryEntry[] {
  const byKind = new Map<RestrictionKind, Set<string>>();

  for (const jurisdiction of jurisdictions ?? []) {
    if (!jurisdiction.requiresPermit) continue;

    for (const kind of jurisdiction.restrictions ?? []) {
      const codes = byKind.get(kind) ?? new Set<string>();
      codes.add(jurisdiction.stateCode);
      byKind.set(kind, codes);
    }
  }

  return [...byKind.entries()].map(([kind, codes]) => ({
    kind,
    label: RESTRICTION_LABELS[kind] ?? kind,
    stateCodes: [...codes],
  }));
}

/**
 * Jurisdictions whose thresholds came from the seeded research baseline rather
 * than a confirmed source. Surfacing them is the difference between a tool that
 * says "GA allows 8'6"" and one that says "GA allows 8'6", unconfirmed" — only
 * the second lets an operator judge whether to call the permit office.
 */
export function unverifiedJurisdictions(
  assessment: PermitAssessment | null | undefined,
): AssessedJurisdiction[] {
  return (assessment?.jurisdictions ?? []).filter(
    (jurisdiction) => jurisdiction.verificationState !== "Verified",
  );
}

export function hasOpenRequirements(assessment: PermitAssessment | null | undefined): boolean {
  return (assessment?.open?.length ?? 0) > 0;
}

export function isOversize(assessment: PermitAssessment | null | undefined): boolean {
  return (assessment?.requirements?.length ?? 0) > 0;
}

/**
 * True when the booked pickup lands before the slowest jurisdiction on the
 * route can issue. This is the number that quietly loses money: a Monday pickup
 * quoted against a five-day permit is an appointment that cannot be met, and
 * nothing else in the shipment says so.
 */
export function pickupIsTooSoon(
  assessment: PermitAssessment | null | undefined,
  scheduledPickupAt: number | null | undefined,
): boolean {
  if (!assessment || !assessment.maxLeadTimeDays) return false;
  if (!scheduledPickupAt || scheduledPickupAt <= 0) return false;

  return scheduledPickupAt < assessment.earliestPickup;
}

/**
 * What a carrier override actually narrows, as short display labels.
 *
 * An override is sparse by design — an unset field defers to the statutory rule
 * — so a caller cannot simply render every limit. This answers the question an
 * operator has of a row, which is what it changes, and returns empty for an
 * override that changes nothing rather than inventing a label for it.
 *
 * Restrictions appear only when turned on: an override can add a restriction
 * the state does not impose but cannot clear one it does, so `false` here means
 * "defer", not "permitted".
 */
export function overriddenLimits(override: {
  maxWidthFeet?: number | null;
  maxHeightFeet?: number | null;
  maxLengthFeet?: number | null;
  maxWeightPounds?: number | null;
  permitLeadTimeDays?: number | null;
  daylightOnly?: boolean | null;
  holidayRestricted?: boolean | null;
}): string[] {
  const applied: string[] = [];

  if (override.maxWidthFeet != null) {
    applied.push(`Width ${formatFeetInches(override.maxWidthFeet)}`);
  }
  if (override.maxHeightFeet != null) {
    applied.push(`Height ${formatFeetInches(override.maxHeightFeet)}`);
  }
  if (override.maxLengthFeet != null) {
    applied.push(`Length ${formatFeetInches(override.maxLengthFeet)}`);
  }
  if (override.maxWeightPounds != null) {
    applied.push(formatPounds(override.maxWeightPounds));
  }
  if (override.permitLeadTimeDays != null) {
    applied.push(`${override.permitLeadTimeDays}d lead`);
  }
  if (override.daylightOnly) applied.push("Daylight only");
  if (override.holidayRestricted) applied.push("No holidays");

  return applied;
}
