// Typed calls for the rules module's lane operations (`lane.*`): the keys a rate agreement rule
// stores, how specific it is, how a stored key reads back, and which lanes can never apply.
// The rules are in docs/engineering/rules-engine.md, "Lanes"; testdata/lane.json pins them.
//
// Each call takes every lane of one agreement (or every key on screen) at once. A rule is sent
// as the fields the lane rules read, so a form's half-typed row never refuses the request.

import type { CatalogMessage } from "@trenova/shared/i18n/message";
import type { RulesRuntime } from "./runtime";
import { wireDecimal, wireInteger, wireList, wireText } from "./wire";

/** A rate agreement rule as a form or GraphQL holds it; the fields the lane rules read. */
export type LaneRuleInput = {
  readonly status?: string | null;
  readonly direction?: string | null;
  readonly originScopeType?: string | null;
  readonly originScopeValue?: string | null;
  readonly originCity?: string | null;
  readonly destinationScopeType?: string | null;
  readonly destinationScopeValue?: string | null;
  readonly destinationCity?: string | null;
  readonly priority?: number | null;
  readonly effectiveFrom?: number | null;
  readonly effectiveTo?: number | null;
  readonly commodityIds?: readonly unknown[] | null;
  readonly freightClasses?: readonly unknown[] | null;
  readonly tractorTypeIds?: readonly unknown[] | null;
  readonly trailerTypeIds?: readonly unknown[] | null;
  readonly equipmentClasses?: readonly unknown[] | null;
  readonly serviceTypeIds?: readonly unknown[] | null;
  readonly shipmentTypeIds?: readonly unknown[] | null;
  readonly serviceModels?: readonly unknown[] | null;
  readonly minWeight?: number | string | null;
  readonly maxWeight?: number | string | null;
  readonly minDistance?: number | string | null;
  readonly maxDistance?: number | string | null;
  readonly hazmatOnly?: boolean | null;
  readonly tempControlOnly?: boolean | null;
};

/** A scope type whose lane key end names a record. */
export type LaneRecordScope = "Location" | "Zone" | "State" | "CityState";

/**
 * One end of a lane key. An end that names a record carries its `scope`, the record `id` and
 * the text around it, so a display can write `prefix + name + suffix`; a literal end has only
 * its `text`.
 */
export type LaneSegment = {
  readonly text: string;
  readonly scope?: LaneRecordScope;
  readonly id?: string;
  readonly prefix?: string;
  readonly suffix?: string;
};

/** The keys one rule stores. */
export type LaneRuleKeys = {
  /** The origin end's key; null when it cannot be keyed (a radius, no value). */
  readonly origin: string | null;
  readonly destination: string | null;
  /** `origin>destination`, or null when either end cannot be keyed. */
  readonly key: string | null;
  /** What the rule stores: its key, or `RADIUS` for a lane matched geospatially. */
  readonly stored: string;
  /** The swapped key of a bidirectional rule. */
  readonly reverse: string | null;
  /** `key` read back into its ends; empty when there is no key. */
  readonly segments: readonly LaneSegment[];
};

/** A lane that can never apply. */
export type LaneCoverageIssue = {
  readonly index: number;
  readonly kind: "duplicate" | "shadowed";
  /** The first lane it collides with. */
  readonly other: number;
  readonly message: CatalogMessage;
};

/** The fields of a rule the lane operations read, safe to send while it is being typed. */
export function laneRuleWire(rule: LaneRuleInput | null | undefined) {
  if (!rule) return {};
  return {
    status: wireText(rule.status),
    direction: wireText(rule.direction),
    originScopeType: wireText(rule.originScopeType),
    originScopeValue: wireText(rule.originScopeValue),
    originCity: wireText(rule.originCity),
    destinationScopeType: wireText(rule.destinationScopeType),
    destinationScopeValue: wireText(rule.destinationScopeValue),
    destinationCity: wireText(rule.destinationCity),
    priority: wireInteger(rule.priority),
    effectiveFrom: wireInteger(rule.effectiveFrom),
    effectiveTo: wireInteger(rule.effectiveTo),
    commodityIds: wireList(rule.commodityIds),
    freightClasses: wireList(rule.freightClasses),
    tractorTypeIds: wireList(rule.tractorTypeIds),
    trailerTypeIds: wireList(rule.trailerTypeIds),
    equipmentClasses: wireList(rule.equipmentClasses),
    serviceTypeIds: wireList(rule.serviceTypeIds),
    shipmentTypeIds: wireList(rule.shipmentTypeIds),
    serviceModels: wireList(rule.serviceModels),
    minWeight: wireDecimal(rule.minWeight),
    maxWeight: wireDecimal(rule.maxWeight),
    minDistance: wireDecimal(rule.minDistance),
    maxDistance: wireDecimal(rule.maxDistance),
    hazmatOnly: rule.hazmatOnly === true,
    tempControlOnly: rule.tempControlOnly === true,
  };
}

/** `lane.keys`: the keys each rule stores, in request order. */
export function laneKeys(
  rules: RulesRuntime,
  laneRules: readonly (LaneRuleInput | null | undefined)[],
): LaneRuleKeys[] {
  const output = rules.call("lane.keys", { rules: laneRules.map(laneRuleWire) }) as {
    rules: LaneRuleKeys[];
  };
  return output.rules;
}

/** `lane.specificity`: each rule's specificity score, in request order. */
export function laneSpecificities(
  rules: RulesRuntime,
  laneRules: readonly (LaneRuleInput | null | undefined)[],
): number[] {
  const output = rules.call("lane.specificity", { rules: laneRules.map(laneRuleWire) }) as {
    scores: number[];
  };
  return output.scores;
}

/** `lane.parse`: each stored key read back into its ends, in request order. */
export function parseLaneKeys(rules: RulesRuntime, keys: readonly string[]): LaneSegment[][] {
  const output = rules.call("lane.parse", { keys }) as { keys: { segments: LaneSegment[] }[] };
  return output.keys.map((key) => key.segments);
}

/** `lane.coverage`: the lanes of one agreement that can never apply, by position. */
export function laneCoverage(
  rules: RulesRuntime,
  laneRules: readonly (LaneRuleInput | null | undefined)[],
): LaneCoverageIssue[] {
  const output = rules.call("lane.coverage", { rules: laneRules.map(laneRuleWire) }) as {
    issues: LaneCoverageIssue[];
  };
  return output.issues;
}
