import {
  fetchDeskLocation,
  fetchLocationDwell,
  fetchLocationPolicies,
  type TimeWindow,
} from "@/lib/graphql/location-card";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

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
});
