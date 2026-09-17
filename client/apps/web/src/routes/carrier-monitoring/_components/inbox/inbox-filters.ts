import { CARRIER_INTEL_SECTIONS, CARRIER_INTEL_SEVERITIES } from "@/lib/carrier-intelligence";
import type { CarrierIntelEventInboxVariables } from "@/lib/graphql/carrier-monitoring-table";
import type {
  CarrierIntelEventFilterInput,
  CarrierIntelEventSource,
  CarrierIntelSection,
  CarrierIntelSeverity,
  FieldFilterInput,
} from "@trenova/graphql/generated/graphql";
import { parseAsArrayOf, parseAsString, parseAsStringLiteral } from "nuqs";

export const INBOX_SCOPES = ["attention", "acknowledged", "resolved", "all"] as const;
export type InboxScope = (typeof INBOX_SCOPES)[number];

export const INBOX_GROUPINGS = ["day", "carrier"] as const;
export type InboxGrouping = (typeof INBOX_GROUPINGS)[number];

export const CARRIER_INTEL_EVENT_SOURCES = [
  "NativeChangeFeed",
  "SnapshotDiff",
  "RuleEvaluation",
  "EquipmentVerification",
  "Enrollment",
  "ProviderError",
  "Override",
] as const satisfies readonly CarrierIntelEventSource[];

export const INBOX_FILTER_IDS = ["severity", "category", "source", "carrier"] as const;
export type InboxFilterId = (typeof INBOX_FILTER_IDS)[number];

export const inboxSearchParams = {
  scope: parseAsStringLiteral(INBOX_SCOPES).withDefault("attention"),
  q: parseAsString.withDefault(""),
  severity: parseAsArrayOf(parseAsStringLiteral(CARRIER_INTEL_SEVERITIES)).withDefault([]),
  category: parseAsArrayOf(parseAsStringLiteral(CARRIER_INTEL_SECTIONS)).withDefault([]),
  source: parseAsArrayOf(parseAsStringLiteral(CARRIER_INTEL_EVENT_SOURCES)).withDefault([]),
  carrier: parseAsString,
  event: parseAsString,
  group: parseAsStringLiteral(INBOX_GROUPINGS).withDefault("day"),
};

export const CLEARED_INBOX_FILTERS = {
  q: null,
  severity: null,
  category: null,
  source: null,
  carrier: null,
} as const;

export const CLEARED_INBOX_STATE = {
  ...CLEARED_INBOX_FILTERS,
  scope: null,
  event: null,
  group: null,
} as const;

export type InboxFilterState = {
  scope: InboxScope;
  q: string;
  severity: readonly CarrierIntelSeverity[];
  category: readonly CarrierIntelSection[];
  source: readonly CarrierIntelEventSource[];
  carrier: string | null;
};

function scopeFilter(scope: InboxScope): CarrierIntelEventFilterInput {
  switch (scope) {
    case "attention":
      return { statuses: ["Open"] };
    case "acknowledged":
      return { statuses: ["Acknowledged"] };
    case "resolved":
      return { statuses: ["Resolved", "Dismissed"] };
    case "all":
      return {};
  }
}

export function buildInboxQueryVariables(state: InboxFilterState): CarrierIntelEventInboxVariables {
  const filter = scopeFilter(state.scope);
  if (state.severity.length > 0) {
    filter.severities = [...state.severity];
  }
  if (state.category.length > 0) {
    filter.categories = [...state.category];
  }
  if (state.carrier) {
    filter.carrierId = state.carrier;
  }

  const fieldFilters: FieldFilterInput[] = [];
  if (state.source.length > 0) {
    fieldFilters.push({ field: "source", operator: "in", value: [...state.source] });
  }

  const query = state.q.trim();
  return { filter, query: query === "" ? null : query, fieldFilters };
}

export function hasInboxFilters(state: InboxFilterState): boolean {
  return (
    state.q.trim() !== "" ||
    state.severity.length > 0 ||
    state.category.length > 0 ||
    state.source.length > 0 ||
    state.carrier !== null
  );
}
