/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { useQuery, useQueryClient } from "@tanstack/react-query";

// Create a cache for location data to avoid excessive requests
const locationCache = new Map<string, LocationSchema>();

export function useLocationData(locationId: string) {
  const queryClient = useQueryClient();

  const result = useQuery({
    queryKey: ["location", locationId],
    queryFn: async () => {
      // Check local cache first
      if (locationCache.has(locationId)) {
        return locationCache.get(locationId)!;
      }

      // Use the query cache if available
      const cachedData = queryClient.getQueryData<LocationSchema>([
        "location",
        locationId,
      ]);
      if (cachedData) {
        locationCache.set(locationId, cachedData);
        return cachedData;
      }

      // Fetch from server if not cached
      const response = await http.get<LocationSchema>(
        `/locations/${locationId}`,
        {
          params: {
            includeCategory: "true",
            includeState: "true",
          },
        },
      );

      // Store in local cache and update the query client cache
      locationCache.set(locationId, response.data);
      queryClient.setQueryData(["location", locationId], response.data);

      return response.data;
    },
    enabled: !!locationId && locationId !== "",
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes
    // Prevent refetching on window focus to avoid unnecessary requests
    refetchOnWindowFocus: false,
  });

  // When the component mounts or the locationId changes, ensure we have the data from cache
  if (locationId && locationCache.has(locationId) && !result.data) {
    // Force an update of the data from our local cache to React Query cache
    queryClient.setQueryData(
      ["location", locationId],
      locationCache.get(locationId),
    );
  }

  return result;
}
