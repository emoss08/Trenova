import {
  fetchDeskLocation,
  fetchLocationDwell,
  fetchLocationFailures,
  fetchLocationPolicies,
  type TimeWindow,
} from "@/lib/graphql/location-card";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { infiniteQueryOptions } from "@tanstack/react-query";

export const location = createQueryKeys("location", {
  geofences: (limit = 100) => ({
    queryKey: ["geofences", limit],
    queryFn: async () => apiService.locationService.listGeofenced(limit),
  }),
  card: (locationId: string) => ({
    queryKey: ["card", locationId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskLocation(locationId, { signal }),
  }),
  dwell: (locationId: string, window: TimeWindow) => ({
    queryKey: ["dwell", locationId, window],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchLocationDwell(locationId, window, { signal }),
  }),
  policies: (locationId: string, first: number) => ({
    queryKey: ["policies", locationId, first],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchLocationPolicies(locationId, first, { signal }),
  }),
  failures: (locationId: string) => ({
    queryKey: ["failures", locationId],
  }),
});

/** The service failures at a location, newest first, a page at a time. */
export function locationFailuresQuery(locationId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: [...location.failures(locationId).queryKey, pageSize],
    queryFn: ({ pageParam, signal }) =>
      fetchLocationFailures(locationId, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}
