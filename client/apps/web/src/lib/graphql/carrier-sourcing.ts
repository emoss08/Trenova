import {
  CarrierIntelLookupDocument,
  CarrierSourcingAutocompleteDocument,
  CarrierSourcingSearchDocument,
  ImportSourcedCarrierDocument,
  type CarrierIntelLookupInput,
  type CarrierIntelLookupQuery,
  type CarrierSourcingAutocompleteQuery,
  type CarrierSourcingSearchInput,
  type CarrierSourcingSearchQuery,
  type ImportSourcedCarrierInput,
  type ImportSourcedCarrierMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";

export type CarrierSourcingPage = UnmaskFragments<
  CarrierSourcingSearchQuery["carrierSourcingSearch"]
>;
export type CarrierSourcingResult = CarrierSourcingPage["items"][number];
export type CarrierSourcingSuggestion =
  CarrierSourcingAutocompleteQuery["carrierSourcingAutocomplete"][number];
export type CarrierIntelProspectLookup = UnmaskFragments<
  CarrierIntelLookupQuery["carrierIntelLookup"]
>;
export type ImportedSourcedCarrier = ImportSourcedCarrierMutation["importSourcedCarrier"];

export const CARRIER_SOURCING_SEARCH_KEY = "carrier-sourcing-search";
export const CARRIER_SOURCING_AUTOCOMPLETE_KEY = "carrier-sourcing-autocomplete";
export const CARRIER_INTEL_LOOKUP_KEY = "carrier-intel-lookup";

type RequestOptions = { signal?: AbortSignal };

export async function searchCarrierSourcing(
  input: CarrierSourcingSearchInput,
  options?: RequestOptions,
): Promise<CarrierSourcingPage> {
  const data = await requestGraphQL({
    document: CarrierSourcingSearchDocument,
    operationName: "CarrierSourcingSearch",
    variables: { input },
    signal: options?.signal,
  });
  return data.carrierSourcingSearch as unknown as CarrierSourcingPage;
}

export async function fetchCarrierSourcingAutocomplete(
  query: string,
  limit: number,
  options?: RequestOptions,
): Promise<CarrierSourcingSuggestion[]> {
  const data = await requestGraphQL({
    document: CarrierSourcingAutocompleteDocument,
    operationName: "CarrierSourcingAutocomplete",
    variables: { query, limit },
    signal: options?.signal,
  });
  return data.carrierSourcingAutocomplete;
}

export async function lookupCarrierIntelProspect(
  input: CarrierIntelLookupInput,
  options?: RequestOptions,
): Promise<CarrierIntelProspectLookup> {
  const data = await requestGraphQL({
    document: CarrierIntelLookupDocument,
    operationName: "CarrierIntelLookup",
    variables: { input },
    signal: options?.signal,
  });
  return data.carrierIntelLookup as unknown as CarrierIntelProspectLookup;
}

export async function importSourcedCarrier(
  input: ImportSourcedCarrierInput,
): Promise<ImportedSourcedCarrier> {
  const data = await requestGraphQL({
    document: ImportSourcedCarrierDocument,
    operationName: "ImportSourcedCarrier",
    variables: { input },
  });
  return data.importSourcedCarrier;
}
