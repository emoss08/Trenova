import { CARRIER_INTEL_SECTIONS, CARRIER_INTEL_SEVERITIES } from "@/lib/carrier-intelligence";
import type {
  CarrierIntelEventFilterInput,
  CarrierIntelEventStatus,
  CarrierIntelSection,
  CarrierIntelSeverity,
} from "@trenova/graphql/generated/graphql";
import { parseAsArrayOf, parseAsString, parseAsStringLiteral } from "nuqs";

export const EVENT_SCOPES = [
  "attention",
  "Open",
  "Acknowledged",
  "Resolved",
  "Dismissed",
  "all",
] as const;

export type EventScope = (typeof EVENT_SCOPES)[number];

export const eventInboxSearchParams = {
  scope: parseAsStringLiteral(EVENT_SCOPES).withDefault("attention"),
  severity: parseAsArrayOf(parseAsStringLiteral(CARRIER_INTEL_SEVERITIES)).withDefault([]),
  category: parseAsStringLiteral(CARRIER_INTEL_SECTIONS),
  carrier: parseAsString,
};

export type EventInboxFilterState = {
  scope: EventScope;
  severity: readonly CarrierIntelSeverity[];
  category: CarrierIntelSection | null;
  carrier: string | null;
};

export function buildEventFilter(state: EventInboxFilterState): CarrierIntelEventFilterInput {
  const filter: CarrierIntelEventFilterInput = {};
  if (state.scope === "attention") {
    filter.openOnly = true;
  } else if (state.scope !== "all") {
    filter.statuses = [state.scope satisfies CarrierIntelEventStatus];
  }
  if (state.severity.length > 0) {
    filter.severities = [...state.severity];
  }
  if (state.category) {
    filter.categories = [state.category];
  }
  if (state.carrier) {
    filter.carrierId = state.carrier;
  }
  return filter;
}

export function hasNarrowingEventFilters(state: EventInboxFilterState): boolean {
  return (
    state.scope !== "attention" ||
    state.severity.length > 0 ||
    state.category !== null ||
    state.carrier !== null
  );
}
