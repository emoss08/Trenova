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
  return laneKeyText(rule, (_, value) => value);
}

/**
 * Resolves an id-backed scope value — a state, zone or location id — to the
 * record's name. Returning undefined means the name is not known yet, which the
 * display renders as pending rather than falling back to the id.
 */
export type LaneScopeLabelResolver = (type: RateScopeType, value: string) => string | undefined;

/**
 * The lane key as a person should read it: the same shape the engine matches
 * on, with every record id replaced by the record's name. Ids must never reach
 * the screen — a value whose name has not resolved yet reads as an ellipsis.
 */
export function laneKeyDisplay(
  rule: RateAgreementRule,
  resolveScopeValue: LaneScopeLabelResolver,
): string | null {
  return laneKeyText(rule, (type, value) => resolveScopeValue(type, value) ?? "…");
}

function laneKeyText(
  rule: RateAgreementRule,
  renderValue: (type: RateScopeType, value: string) => string,
): string | null {
  const origin = scopeKey(
    rule.originScopeType,
    rule.originScopeValue,
    rule.originCity,
    renderValue,
  );
  const destination = scopeKey(
    rule.destinationScopeType,
    rule.destinationScopeValue,
    rule.destinationCity,
    renderValue,
  );

  if (origin === null || destination === null) {
    return null;
  }

  return `${origin}>${destination}`;
}

/**
 * One end of a lane key. Scopes naming a record (state, zone, location) pass
 * their value through renderValue, so the same shape serves both the stored key
 * (the value verbatim) and the human reading of it (the record's name). Literal
 * scopes — postal codes, countries, cities — are their own display.
 */
function scopeKey(
  type: RateScopeType,
  value: string,
  city: string,
  renderValue: (type: RateScopeType, value: string) => string,
): string | null {
  switch (type) {
    case "Any":
      return "ANY";
    case "Location":
      return value ? `LOC:${renderValue(type, value)}` : "";
    case "Zip5":
      return value ? `Z5:${normalizePostal(value)}` : "";
    case "Zip3":
      return value ? `Z3:${normalizePostal(value).slice(0, 3)}` : "";
    case "CityState":
      return value && city ? `CS:${renderValue(type, value)}|${city.trim().toUpperCase()}` : "";
    case "Zone":
      return value ? `ZN:${renderValue(type, value)}` : "";
    case "State":
      return value ? `ST:${renderValue(type, value)}` : "";
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
 * The stored key prefixes whose payload is a record id rather than a literal.
 * Everything else in a key — postal digits, country codes, folded cities — is
 * its own display.
 */
const KEY_PREFIX_SCOPE: Record<string, RateScopeType> = {
  LOC: "Location",
  ZN: "Zone",
  ST: "State",
  CS: "CityState",
};

export type LaneKeyScopeRef = {
  type: RateScopeType;
  id: string;
};

function parseKeySegment(segment: string): LaneKeyScopeRef | null {
  const colon = segment.indexOf(":");
  if (colon < 0) {
    return null;
  }

  const type = KEY_PREFIX_SCOPE[segment.slice(0, colon)];
  if (!type) {
    return null;
  }

  const rest = segment.slice(colon + 1);
  const id = type === "CityState" ? rest.split("|")[0] : rest;

  return id ? { type, id } : null;
}

/**
 * The record ids a stored lane key references, typed by the scope that stored
 * them — what a caller has to look up before the key can be shown to a person.
 */
export function laneKeyScopeIds(key: string): LaneKeyScopeRef[] {
  const refs: LaneKeyScopeRef[] = [];
  for (const segment of key.split(">")) {
    const ref = parseKeySegment(segment);
    if (ref) {
      refs.push(ref);
    }
  }

  return refs;
}

/**
 * A stored lane key as a person should read it, for the keys that arrive from
 * the server already assembled — import diffs, simulation rows — where there is
 * no rule object to rebuild the key from. Same contract as `laneKeyDisplay`:
 * ids never reach the screen, an unresolved name reads as an ellipsis, and
 * everything literal is left exactly as stored.
 */
export function laneKeyStringDisplay(
  key: string,
  resolveScopeValue: LaneScopeLabelResolver,
): string {
  return key
    .split(">")
    .map((segment) => {
      const ref = parseKeySegment(segment);
      if (!ref) {
        return segment;
      }

      const label = resolveScopeValue(ref.type, ref.id) ?? "…";
      const idStart = segment.indexOf(":") + 1;

      return segment.slice(0, idStart) + label + segment.slice(idStart + ref.id.length);
    })
    .join(">");
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
