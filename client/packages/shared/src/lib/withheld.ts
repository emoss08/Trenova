import type { UnavailableFeature } from "@trenova/shared/types/permission";
import { isRecord } from "@trenova/shared/lib/utils";

/**
 * Names, without list indices, every record a REST response answered with only part of its
 * fields because the reader cannot read its kind. See docs/engineering/reference-views.md.
 */
export const WITHHELD_BY_ACCESS_HEADER = "Withheld-By-Access";

/**
 * Names, without list indices, every record or field a REST response withheld because the
 * organization's plan does not include its feature, each as `path=feature`.
 */
export const WITHHELD_BY_PLAN_HEADER = "Withheld-By-Plan";

/** A path the server withheld because the organization's plan does not include its feature. */
export type PlanWithholding = {
  path: string;
  feature: string;
};

function splitList(value: string | null | undefined): string[] {
  if (!value) return [];
  return value
    .split(",")
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}

/** The paths a `Withheld-By-Access` header names. */
export function parseWithheldByAccessHeader(value: string | null | undefined): string[] {
  return splitList(value);
}

/**
 * The paths and features a `Withheld-By-Plan` header names. An entry without a path or a
 * feature is ignored.
 */
export function parseWithheldByPlanHeader(value: string | null | undefined): PlanWithholding[] {
  const out: PlanWithholding[] = [];
  for (const entry of splitList(value)) {
    const separator = entry.lastIndexOf("=");
    const path = entry.slice(0, separator).trim();
    const feature = entry.slice(separator + 1).trim();
    if (separator <= 0 || !path || !feature) continue;
    out.push({ path, feature });
  }
  return out;
}

/** The paths a GraphQL response's `extensions.withheldByAccess` names. */
export function withheldByAccessOf(extensions: unknown): string[] {
  if (!isRecord(extensions) || !Array.isArray(extensions.withheldByAccess)) return [];
  return extensions.withheldByAccess.filter(
    (path): path is string => typeof path === "string" && path.length > 0,
  );
}

/** The paths and features a GraphQL response's `extensions.withheldByPlan` names. */
export function withheldByPlanOf(extensions: unknown): PlanWithholding[] {
  if (!isRecord(extensions) || !Array.isArray(extensions.withheldByPlan)) return [];
  const out: PlanWithholding[] = [];
  for (const entry of extensions.withheldByPlan) {
    if (!isRecord(entry)) continue;
    const { path, feature } = entry;
    if (typeof path !== "string" || !path || typeof feature !== "string" || !feature) continue;
    out.push({ path, feature });
  }
  return out;
}

/**
 * The feature and add-on a plan withholding belongs to, from the manifest's
 * `unavailableFeatures`, or undefined when the manifest does not list it.
 */
export function unavailableFeatureOf(
  withholding: PlanWithholding,
  unavailable: readonly UnavailableFeature[],
): UnavailableFeature | undefined {
  return unavailable.find((feature) => feature.feature === withholding.feature);
}

/**
 * What one response left out: the paths withheld because the reader may not read their kind,
 * and the paths withheld because the organization's plan does not include their feature.
 * Paths carry no list indices (`shipment.moves.tender`).
 */
export type Withholdings = {
  byAccess: readonly string[];
  byPlan: readonly PlanWithholding[];
};

export const NO_WITHHOLDINGS: Withholdings = Object.freeze({
  byAccess: Object.freeze([]) as readonly string[],
  byPlan: Object.freeze([]) as readonly PlanWithholding[],
});

const recorded = new WeakMap<object, Withholdings>();

/**
 * Remembers what the response that produced `value` withheld. The transports call it with the
 * object they return, so a caller that holds the data can ask `withholdingsOf(data)` without
 * the transports' return type changing. A response that withheld nothing records nothing.
 */
export function recordWithholdings(value: unknown, withholdings: Withholdings): void {
  if (typeof value !== "object" || value === null) return;
  if (withholdings.byAccess.length === 0 && withholdings.byPlan.length === 0) return;
  recorded.set(value, withholdings);
}

/**
 * What the response that produced `value` withheld: the object `requestGraphQL` resolved to,
 * a REST body, or a `requestGraphQLResult`'s `data`. Empty for anything else, and for a copy
 * of the data (a `select`, a spread), which is not the object the response produced.
 */
export function withholdingsOf(value: unknown): Withholdings {
  if (typeof value !== "object" || value === null) return NO_WITHHOLDINGS;
  return recorded.get(value) ?? NO_WITHHOLDINGS;
}

/** The withholdings a GraphQL response's `extensions` names. */
export function withholdingsOfExtensions(extensions: unknown): Withholdings {
  const byAccess = withheldByAccessOf(extensions);
  const byPlan = withheldByPlanOf(extensions);
  return byAccess.length === 0 && byPlan.length === 0 ? NO_WITHHOLDINGS : { byAccess, byPlan };
}

/** The withholdings a REST response's headers name. */
export function withholdingsOfHeaders(headers: Headers): Withholdings {
  const byAccess = parseWithheldByAccessHeader(headers.get(WITHHELD_BY_ACCESS_HEADER));
  const byPlan = parseWithheldByPlanHeader(headers.get(WITHHELD_BY_PLAN_HEADER));
  return byAccess.length === 0 && byPlan.length === 0 ? NO_WITHHOLDINGS : { byAccess, byPlan };
}

/**
 * The plan withholding at a path, written without list indices as the server names it
 * (`invoice.factoringCompany`), or undefined when the plan withheld nothing there.
 */
export function planWithholdingAt(
  withholdings: Withholdings,
  path: string,
): PlanWithholding | undefined {
  return withholdings.byPlan.find((entry) => entry.path === path);
}

/** Whether the reader's access withheld the value at a path. */
export function isWithheldByAccess(withholdings: Withholdings, path: string): boolean {
  return withholdings.byAccess.includes(path);
}
