import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  LocationGeofenceFieldsFragmentDoc,
  LocationGeofencesDocument,
  type LocationGeofenceFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type LocationGeofence = LocationGeofenceFieldsFragment;

/** The first `first` locations with what the map needs to draw their geofences. */
export async function fetchLocationGeofences(
  first: number,
  options?: { signal?: AbortSignal },
): Promise<LocationGeofence[]> {
  const data = await requestGraphQL({
    document: LocationGeofencesDocument,
    operationName: "LocationGeofences",
    variables: { first },
    signal: options?.signal,
  });

  return data.locations.edges.map((edge) =>
    getFragmentData(LocationGeofenceFieldsFragmentDoc, edge.node),
  );
}
