import {
  getCapacityUnitMatchesGraphQL,
  getShipmentBoardCapabilitiesGraphQL,
  getShipmentBriefingGraphQL,
  getShipmentCapacityGraphQL,
  getShipmentCoverageSuggestionsGraphQL,
  getShipmentQuickFilterCountsGraphQL,
  getShipmentStageSummaryGraphQL,
  getShipmentSuggestionsGraphQL,
  getShipmentWatchlistGraphQL,
} from "@/lib/graphql/shipment-board";
import type { CapacityUnitKind, ShipmentBoardScopeInput } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const shipmentBoard = createQueryKeys("shipmentBoard", {
  capabilities: () => ({
    queryKey: ["capabilities"],
    queryFn: ({ signal }) => getShipmentBoardCapabilitiesGraphQL({ signal }),
  }),
  stageSummary: (input: ShipmentBoardScopeInput) => ({
    queryKey: ["stage-summary", input],
    queryFn: ({ signal }) => getShipmentStageSummaryGraphQL(input, { signal }),
  }),
  quickFilterCounts: (input: ShipmentBoardScopeInput) => ({
    queryKey: ["quick-filter-counts", input],
    queryFn: ({ signal }) => getShipmentQuickFilterCountsGraphQL(input, { signal }),
  }),
  briefing: (timezone: string) => ({
    queryKey: ["briefing", timezone],
    queryFn: ({ signal }) => getShipmentBriefingGraphQL(timezone, { signal }),
  }),
  capacity: (kind: CapacityUnitKind) => ({
    queryKey: ["capacity", kind],
    queryFn: ({ signal }) => getShipmentCapacityGraphQL(kind, { signal }),
  }),
  capacityMatches: (kind: CapacityUnitKind, unitId: string, limit = 2) => ({
    queryKey: ["capacity-matches", kind, unitId, limit],
    queryFn: ({ signal }) => getCapacityUnitMatchesGraphQL({ kind, unitId, limit }, { signal }),
  }),
  coverageSuggestions: (shipmentId: string) => ({
    queryKey: ["coverage-suggestions", shipmentId],
    queryFn: ({ signal }) => getShipmentCoverageSuggestionsGraphQL(shipmentId, { signal }),
  }),
  suggestions: (timezone: string) => ({
    queryKey: ["suggestions", timezone],
    queryFn: ({ signal }) => getShipmentSuggestionsGraphQL(timezone, { signal }),
  }),
  watchlist: (timezone: string) => ({
    queryKey: ["watchlist", timezone],
    queryFn: ({ signal }) => getShipmentWatchlistGraphQL(timezone, { signal }),
  }),
});
