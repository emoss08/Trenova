import {
  CapacityUnitMatchesDocument,
  DecideShipmentSuggestionDocument,
  NotifyShipmentDelayDocument,
  ShipmentBoardCapabilitiesDocument,
  ShipmentBoardGroupsDocument,
  ShipmentBriefingDocument,
  ShipmentCapacityDocument,
  ShipmentCoverageSuggestionsDocument,
  ShipmentQuickFilterCountsDocument,
  ShipmentStageSummaryDocument,
  ShipmentSuggestionsDocument,
  ShipmentWatchlistDocument,
  TenderShipmentsDocument,
  UndoShipmentSuggestionDecisionDocument,
  type CapacityUnitKind,
  type CapacityUnitMatchesQuery,
  type DecideShipmentSuggestionInput,
  type NotifyShipmentDelayInput,
  type ShipmentBoardCapabilitiesQuery,
  type ShipmentBoardGrouping,
  type ShipmentBoardGroupsQuery,
  type ShipmentBoardScopeInput,
  type ShipmentBriefingQuery,
  type ShipmentCapacityQuery,
  type ShipmentCoverageSuggestionsQuery,
  type ShipmentQuickFilterCountsQuery,
  type ShipmentStageSummaryQuery,
  type ShipmentSuggestionsQuery,
  type ShipmentWatchlistQuery,
  type TenderShipmentsInput,
  type TenderShipmentsMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export type ShipmentBoardCapabilities = ShipmentBoardCapabilitiesQuery["shipmentBoardCapabilities"];
export type ShipmentStageSummary = ShipmentStageSummaryQuery["shipmentStageSummary"][number];
export type ShipmentBoardGroup = ShipmentBoardGroupsQuery["shipmentBoardGroups"][number];
export type ShipmentQuickFilterCount =
  ShipmentQuickFilterCountsQuery["shipmentQuickFilterCounts"][number];
export type ShipmentBriefing = ShipmentBriefingQuery["shipmentBriefing"];
export type ShipmentCapacity = ShipmentCapacityQuery["shipmentCapacity"];
export type CapacityUnit = ShipmentCapacity["units"][number];
export type CapacityMatch = CapacityUnitMatchesQuery["capacityUnitMatches"][number];
export type ShipmentCoverageSuggestions =
  ShipmentCoverageSuggestionsQuery["shipmentCoverageSuggestions"];
export type DriverCoverageSuggestion = ShipmentCoverageSuggestions["drivers"][number];
export type CarrierCoverageSuggestion = ShipmentCoverageSuggestions["carriers"][number];
export type ShipmentSuggestionQueue = ShipmentSuggestionsQuery["shipmentSuggestions"];
export type ShipmentSuggestion = ShipmentSuggestionQueue["items"][number];
export type ShipmentWatchlist = ShipmentWatchlistQuery["shipmentWatchlist"];
export type TenderShipmentsResult = TenderShipmentsMutation["tenderShipments"];

export async function getShipmentBoardCapabilitiesGraphQL(options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentBoardCapabilitiesDocument,
    operationName: "ShipmentBoardCapabilities",
    signal: options?.signal,
  });
  return data.shipmentBoardCapabilities;
}

export async function getShipmentStageSummaryGraphQL(
  input: ShipmentBoardScopeInput,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentStageSummaryDocument,
    operationName: "ShipmentStageSummary",
    variables: { input },
    signal: options?.signal,
  });
  return data.shipmentStageSummary;
}

export async function getShipmentBoardGroupsGraphQL(
  input: ShipmentBoardScopeInput,
  groupBy: ShipmentBoardGrouping,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentBoardGroupsDocument,
    operationName: "ShipmentBoardGroups",
    variables: { input, groupBy },
    signal: options?.signal,
  });
  return data.shipmentBoardGroups;
}

export async function getShipmentQuickFilterCountsGraphQL(
  input: ShipmentBoardScopeInput,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentQuickFilterCountsDocument,
    operationName: "ShipmentQuickFilterCounts",
    variables: { input },
    signal: options?.signal,
  });
  return data.shipmentQuickFilterCounts;
}

export async function getShipmentBriefingGraphQL(timezone: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentBriefingDocument,
    operationName: "ShipmentBriefing",
    variables: { timezone },
    signal: options?.signal,
  });
  return data.shipmentBriefing;
}

export async function getShipmentCapacityGraphQL(kind: CapacityUnitKind, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentCapacityDocument,
    operationName: "ShipmentCapacity",
    variables: { kind },
    signal: options?.signal,
  });
  return data.shipmentCapacity;
}

export async function getCapacityUnitMatchesGraphQL(
  params: { kind: CapacityUnitKind; unitId: string; limit?: number },
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: CapacityUnitMatchesDocument,
    operationName: "CapacityUnitMatches",
    variables: params,
    signal: options?.signal,
  });
  return data.capacityUnitMatches;
}

export async function getShipmentCoverageSuggestionsGraphQL(
  shipmentId: string,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentCoverageSuggestionsDocument,
    operationName: "ShipmentCoverageSuggestions",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentCoverageSuggestions;
}

export async function getShipmentSuggestionsGraphQL(timezone: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentSuggestionsDocument,
    operationName: "ShipmentSuggestions",
    variables: { timezone },
    signal: options?.signal,
  });
  return data.shipmentSuggestions;
}

export async function getShipmentWatchlistGraphQL(timezone: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentWatchlistDocument,
    operationName: "ShipmentWatchlist",
    variables: { timezone },
    signal: options?.signal,
  });
  return data.shipmentWatchlist;
}

export async function tenderShipmentsGraphQL(input: TenderShipmentsInput) {
  const data = await requestGraphQL({
    document: TenderShipmentsDocument,
    operationName: "TenderShipments",
    variables: { input },
  });
  return data.tenderShipments;
}

export async function decideShipmentSuggestionGraphQL(input: DecideShipmentSuggestionInput) {
  const data = await requestGraphQL({
    document: DecideShipmentSuggestionDocument,
    operationName: "DecideShipmentSuggestion",
    variables: { input },
  });
  return data.decideShipmentSuggestion;
}

export async function undoShipmentSuggestionDecisionGraphQL(key: string) {
  const data = await requestGraphQL({
    document: UndoShipmentSuggestionDecisionDocument,
    operationName: "UndoShipmentSuggestionDecision",
    variables: { key },
  });
  return data.undoShipmentSuggestionDecision;
}

export async function notifyShipmentDelayGraphQL(input: NotifyShipmentDelayInput) {
  const data = await requestGraphQL({
    document: NotifyShipmentDelayDocument,
    operationName: "NotifyShipmentDelay",
    variables: { input },
  });
  return data.notifyShipmentDelay;
}
