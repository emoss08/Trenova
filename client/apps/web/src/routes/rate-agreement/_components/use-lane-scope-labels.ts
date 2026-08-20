import { fetchGraphQLSelectOptions, type SelectOption } from "@/lib/graphql/select-options";
import { useQueries } from "@tanstack/react-query";
import type { SelectOptionResource } from "@trenova/graphql/generated/graphql";
import { laneKeyScopeIds, laneKeyStringDisplay } from "@trenova/shared/lib/rate";
import type { LaneScopeLabelResolver } from "@trenova/shared/lib/rate";
import type { RateAgreementRule, RateScopeType } from "@trenova/shared/types/rate";
import { useCallback, useMemo } from "react";

/**
 * Which select-option resource resolves each id-backed scope. CityState stores
 * a state id next to its city text, so it reads from the same place a State
 * scope does. Literal scopes — postal codes, countries — are their own display
 * and never appear here.
 */
const SCOPE_RESOURCE: Partial<Record<RateScopeType, SelectOptionResource>> = {
  State: "US_STATE",
  CityState: "US_STATE",
  Location: "LOCATION",
  Zone: "RATE_ZONE",
};

const RESOURCES = ["US_STATE", "LOCATION", "RATE_ZONE"] as const satisfies SelectOptionResource[];

// States, zones and locations change on the timescale of contracts, not
// keystrokes; refetching them per lane edit would be pure churn.
const LABELS_STALE_TIME = 5 * 60 * 1000;

function scopeLabel(resource: SelectOptionResource, option: SelectOption): string {
  if (resource === "US_STATE") {
    // The lane key is one compact line, which is exactly what the postal
    // abbreviation exists for — "TX", not "Texas".
    const abbreviation = option.meta?.abbreviation;
    if (typeof abbreviation === "string" && abbreviation) {
      return abbreviation;
    }
  }

  return option.label;
}

function addScopeId(
  collected: Map<SelectOptionResource, Set<string>>,
  type: RateScopeType,
  value: string | null | undefined,
): void {
  const resource = SCOPE_RESOURCE[type];
  if (!resource || !value) {
    return;
  }

  let ids = collected.get(resource);
  if (!ids) {
    ids = new Set();
    collected.set(resource, ids);
  }
  ids.add(value);
}

/**
 * The collected ids in RESOURCES order. Sorted, so the query keys are stable
 * however the source data is ordered — the cache is keyed by what is asked,
 * not where it sits.
 */
function sortedByResource(collected: Map<SelectOptionResource, Set<string>>): string[][] {
  return RESOURCES.map((resource) => [...(collected.get(resource) ?? [])].sort());
}

/**
 * Fetches the names behind the collected ids and answers lookups by resource.
 * Ids are batched per resource and fetched once, so an agreement with forty
 * Texas lanes asks about Texas one time.
 */
function useScopeIdLabels(idsByResource: string[][]): LaneScopeLabelResolver {
  const results = useQueries({
    queries: RESOURCES.map((resource, index) => {
      const ids = idsByResource[index];

      return {
        queryKey: ["select-option-labels", resource, ids],
        queryFn: () => fetchGraphQLSelectOptions({ resource, ids, initialLimit: ids.length }),
        enabled: ids.length > 0,
        staleTime: LABELS_STALE_TIME,
      };
    }),
  });

  // RESOURCES is a fixed-length tuple, so each query has a fixed slot. The
  // data references are structurally shared by the query cache, which makes
  // them the stable dependencies the results array itself is not.
  const stateData = results[0].data;
  const locationData = results[1].data;
  const zoneData = results[2].data;

  const labelByScopeValue = useMemo(() => {
    const labels = new Map<string, string>();
    const dataByIndex = [stateData, locationData, zoneData];

    dataByIndex.forEach((data, index) => {
      const resource = RESOURCES[index];
      for (const option of data?.results ?? []) {
        labels.set(`${resource}:${option.id}`, scopeLabel(resource, option));
      }
    });

    return labels;
  }, [stateData, locationData, zoneData]);

  return useCallback(
    (type: RateScopeType, value: string) => {
      const resource = SCOPE_RESOURCE[type];
      if (!resource || !value) {
        return undefined;
      }

      return labelByScopeValue.get(`${resource}:${value}`);
    },
    [labelByScopeValue],
  );
}

/**
 * Resolves the record ids a lane's scopes store — state, zone and location ids
 * — to those records' names, so the lane header shows a person the actual
 * places rather than identifiers.
 */
export function useLaneScopeLabels(rules: RateAgreementRule[]): LaneScopeLabelResolver {
  // Recomputed on every render on purpose: query keys are hashed structurally,
  // so equal id lists never refetch, and memoizing against `rules` — a watched
  // array with a fresh identity per keystroke — would buy nothing.
  const collected = new Map<SelectOptionResource, Set<string>>();
  for (const rule of rules) {
    if (!rule) {
      continue;
    }
    addScopeId(collected, rule.originScopeType, rule.originScopeValue);
    addScopeId(collected, rule.destinationScopeType, rule.destinationScopeValue);
  }

  return useScopeIdLabels(sortedByResource(collected));
}

/**
 * Resolves the ids inside stored lane keys — the strings import diffs and
 * simulation rows arrive with, where there is no rule object to read from —
 * and returns each key ready to show: names in place of ids, everything
 * literal left exactly as stored.
 */
export function useLaneKeyLabels(keys: Array<string | null | undefined>): (key: string) => string {
  const collected = new Map<SelectOptionResource, Set<string>>();
  for (const key of keys) {
    if (!key) {
      continue;
    }
    for (const ref of laneKeyScopeIds(key)) {
      addScopeId(collected, ref.type, ref.id);
    }
  }

  const resolveScopeValue = useScopeIdLabels(sortedByResource(collected));

  return useCallback(
    (key: string) => laneKeyStringDisplay(key, resolveScopeValue),
    [resolveScopeValue],
  );
}
