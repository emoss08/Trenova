import type { RateAgreementRule, RateScopeType } from "../types/rate";

/**
 * How narrowly each scope reads, mirroring `rategeo.Weight`.
 *
 * These are duplicated rather than fetched because the lane editor scores lanes
 * as they are typed, before anything is saved. They must stay in step with the
 * server — a drift shows up as the editor predicting one winner and the engine
 * picking another, which is worse than showing no prediction at all.
 */
const SCOPE_WEIGHT: Record<RateScopeType, number> = {
  Any: 0,
  Country: 300,
  State: 500,
  Zone: 600,
  Radius: 650,
  CityState: 700,
  Zip3: 800,
  Zip5: 900,
  Location: 1000,
};

/**
 * Geography is scaled clear of the condition terms, which sum to 1023, so one
 * step of geographic granularity can never be outvoted by any combination of
 * conditions. Mirrors `rateagreement.GeographyScale`.
 */
const GEOGRAPHY_SCALE = 1024;

/**
 * One weight per condition, mirroring `rateagreement`'s constants.
 *
 * The two ends of a lane are weighted equally: an origin-specific rule and a
 * destination-specific rule of the same granularity are equally narrow, and
 * there is no principled reason to prefer one.
 */
const CONDITION_WEIGHT = {
  commodity: 512,
  freightClass: 256,
  equipmentType: 128,
  serviceType: 64,
  shipmentType: 32,
  weightRange: 16,
  distanceRange: 8,
  serviceModel: 4,
  hazmat: 2,
  tempControl: 1,
} as const;

function hasAny(values: readonly unknown[] | null | undefined): boolean {
  return Boolean(values && values.length > 0);
}

/**
 * Scores how narrowly a lane is written, the same way the engine does.
 *
 * A higher score wins. Where two lanes tie, the engine falls through to the
 * explicit priority, then the effective date, then the rule id — so the winner
 * is always the same rule, but never one anybody chose.
 */
export function laneSpecificity(rule: RateAgreementRule): number {
  const geography =
    (SCOPE_WEIGHT[rule.originScopeType] + SCOPE_WEIGHT[rule.destinationScopeType]) *
    GEOGRAPHY_SCALE;

  let conditions = 0;

  if (hasAny(rule.commodityIds)) {
    conditions += CONDITION_WEIGHT.commodity;
  }
  if (hasAny(rule.freightClasses)) {
    conditions += CONDITION_WEIGHT.freightClass;
  }
  if (hasAny(rule.tractorTypeIds) || hasAny(rule.trailerTypeIds) || hasAny(rule.equipmentClasses)) {
    conditions += CONDITION_WEIGHT.equipmentType;
  }
  if (hasAny(rule.serviceTypeIds)) {
    conditions += CONDITION_WEIGHT.serviceType;
  }
  if (hasAny(rule.shipmentTypeIds)) {
    conditions += CONDITION_WEIGHT.shipmentType;
  }
  if (rule.minWeight != null || rule.maxWeight != null) {
    conditions += CONDITION_WEIGHT.weightRange;
  }
  if (rule.minDistance != null || rule.maxDistance != null) {
    conditions += CONDITION_WEIGHT.distanceRange;
  }
  if (hasAny(rule.serviceModels)) {
    conditions += CONDITION_WEIGHT.serviceModel;
  }
  if (rule.hazmatOnly) {
    conditions += CONDITION_WEIGHT.hazmat;
  }
  if (rule.tempControlOnly) {
    conditions += CONDITION_WEIGHT.tempControl;
  }

  return geography + conditions;
}

/** The lane key the engine will store, or null for a lane matched by radius. */
export function laneKeyPreview(rule: RateAgreementRule): string | null {
  const origin = scopeKey(rule.originScopeType, rule.originScopeValue, rule.originCity);
  const destination = scopeKey(
    rule.destinationScopeType,
    rule.destinationScopeValue,
    rule.destinationCity,
  );

  if (origin === null || destination === null) {
    return null;
  }

  return `${origin}>${destination}`;
}

function scopeKey(type: RateScopeType, value: string, city: string): string | null {
  switch (type) {
    case "Any":
      return "ANY";
    case "Location":
      return value ? `LOC:${value}` : "";
    case "Zip5":
      return value ? `Z5:${normalizePostal(value)}` : "";
    case "Zip3":
      return value ? `Z3:${normalizePostal(value).slice(0, 3)}` : "";
    case "CityState":
      return value && city ? `CS:${value}|${city.trim().toUpperCase()}` : "";
    case "Zone":
      return value ? `ZN:${value}` : "";
    case "State":
      return value ? `ST:${value}` : "";
    case "Country":
      return value ? `CT:${value.trim().toUpperCase()}` : "";
    case "Radius":
      // Radius lanes are found through a geospatial index and have no key at
      // all, which the caller has to say rather than render something wrong.
      return null;
    default:
      return "";
  }
}

/**
 * Postal codes arrive as "60601", "60601-1234" and occasionally with stray
 * spacing. Only the leading segment identifies the delivery area, which is the
 * reading `rategeo.normalizePostalCode` takes.
 */
function normalizePostal(value: string): string {
  return value.trim().split("-")[0].replace(/\s/g, "").toUpperCase();
}

export type LaneCoverageIssue = {
  index: number;
  kind: "duplicate" | "shadowed";
  message: string;
};

/**
 * Finds the lanes that will never fire, before anybody wonders why a rate did
 * not apply.
 *
 * Two lanes written to the same geography with the same conditions are settled
 * by a tie-break — which is to say one of them silently never wins. A lane
 * sitting behind an identical one with a higher priority is the same problem
 * stated differently. Both are cheap to detect here and expensive to discover
 * from an invoice.
 */
export function findCoverageIssues(rules: RateAgreementRule[]): LaneCoverageIssue[] {
  const issues: LaneCoverageIssue[] = [];
  const seen = new Map<string, number>();

  rules.forEach((rule, index) => {
    if (rule.status !== "Active") {
      return;
    }

    const key = laneKeyPreview(rule);
    if (key === null) {
      // Overlap between radius lanes cannot be judged from the fields alone,
      // so claiming a conflict would be a guess presented as a fact.
      return;
    }

    const signature = `${key}|${laneSpecificity(rule)}`;
    const first = seen.get(signature);

    if (first === undefined) {
      seen.set(signature, index);
      return;
    }

    const winner = rules[first];

    if ((winner.priority ?? 0) === (rule.priority ?? 0)) {
      issues.push({
        index,
        kind: "duplicate",
        message:
          "This lane is written exactly as narrowly as another one, so which of the two applies is decided by a tie-break rather than by you.",
      });
      return;
    }

    issues.push({
      index,
      kind: "shadowed",
      message:
        "Another lane covers the same geography with a higher priority, so this one can never apply.",
    });
  });

  return issues;
}

/** The label a lane shows when nobody has named it. */
export function laneDisplayLabel(rule: RateAgreementRule, index: number): string {
  return rule.label || `Lane ${index + 1}`;
}
