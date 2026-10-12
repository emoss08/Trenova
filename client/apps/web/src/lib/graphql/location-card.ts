import {
  DeskLocationCardDocument,
  DeskLocationDwellDocument,
  DeskLocationFailuresDocument,
  DeskLocationPoliciesDocument,
  type DeskLocationCardQuery,
  type DeskLocationDwellQuery,
  type DeskLocationFailuresQuery,
  type DeskLocationPoliciesQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskLocation = NonNullable<DeskLocationCardQuery["location"]>;
export type LocationDwell = DeskLocationDwellQuery["detentionFacilityStats"][number];
export type LocationPolicy =
  DeskLocationPoliciesQuery["detentionPolicies"]["edges"][number]["node"];
export type LocationFailure = DeskLocationFailuresQuery["serviceFailures"]["edges"][number]["node"];

/** A window of time the card reads a location's history over, in Unix seconds. */
export type TimeWindow = { from: number; to: number };

/** One page of the service failures at a location. `total` is read with the first page only. */
export type LocationFailuresPage = {
  rows: LocationFailure[];
  total: number | null;
  endCursor: string | null;
  hasNextPage: boolean;
};

/** The policies the card reads for a location, and how many there are in all. */
export type LocationPolicies = { rows: LocationPolicy[]; total: number };

/**
 * One location as the Desk's card reads it. A location the server does not
 * return is an error rather than an empty card, so the card says it could not
 * be read.
 */
export async function fetchDeskLocation(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskLocation> {
  const data = await requestGraphQL({
    document: DeskLocationCardDocument,
    operationName: "DeskLocationCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.location) {
    throw new Error("Location not found");
  }
  return data.location;
}

/**
 * How long trucks waited at the location over the window, or null when no
 * closed stop there was scored in it.
 */
export async function fetchLocationDwell(
  locationId: string,
  window: TimeWindow,
  options?: { signal?: AbortSignal },
): Promise<LocationDwell | null> {
  const data = await requestGraphQL({
    document: DeskLocationDwellDocument,
    operationName: "DeskLocationDwell",
    variables: { locationId, from: window.from, to: window.to },
    signal: options?.signal,
  });
  return data.detentionFacilityStats.find((stat) => stat.locationId === locationId) ?? null;
}

/** The detention policies written for the location itself, highest priority first. */
export async function fetchLocationPolicies(
  locationId: string,
  first: number,
  options?: { signal?: AbortSignal },
): Promise<LocationPolicies> {
  const data = await requestGraphQL({
    document: DeskLocationPoliciesDocument,
    operationName: "DeskLocationPolicies",
    variables: {
      input: {
        first,
        fieldFilters: [{ field: "locationId", operator: "eq", value: locationId }],
        sort: [{ field: "priority", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });
  const rows = data.detentionPolicies.edges.map((edge) => edge.node);
  return { rows, total: data.detentionPolicies.totalCount ?? rows.length };
}

/** A page of the service failures at stops made at the location, newest first. */
export async function fetchLocationFailures(
  locationId: string,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<LocationFailuresPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskLocationFailuresDocument,
    operationName: "DeskLocationFailures",
    variables: {
      locationId,
      input: {
        first: page.first,
        after: page.after,
        sort: [{ field: "detectedAt", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const rows = data.serviceFailures.edges.map((edge) => edge.node);
  return {
    rows,
    total: firstPage ? (data.serviceFailures.totalCount ?? rows.length) : null,
    endCursor: data.serviceFailures.pageInfo.endCursor ?? null,
    hasNextPage: data.serviceFailures.pageInfo.hasNextPage,
  };
}
