import { http } from "@/lib/http-client";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { useQuery } from "@tanstack/react-query";

export function useLocationData(locationId: string) {
  return useQuery({
    queryKey: ["location", locationId],
    queryFn: async () => {
      const response = await http.get<LocationSchema>(
        `/locations/${locationId}`,
        {
          params: {
            includeCategory: "true",
            includeState: "true",
          },
        },
      );
      return response.data;
    },
    enabled: !!locationId && locationId !== "",
    staleTime: 30000,
    gcTime: 5 * 60 * 1000,
  });
}
