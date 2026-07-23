import { apiService } from "@/services/api";
import type { AutocompleteLocationRequest } from "@/types/google-maps";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const googleMaps = createQueryKeys("googleMaps", {
  getAPIKey: () => ({
    queryKey: ["googleMapsAPIKey"],
    queryFn: async () => apiService.googleMapsService.getAPIKey(),
  }),
  autocomplete: (req: AutocompleteLocationRequest) => ({
    queryKey: ["autocomplete", req],
    queryFn: async () => apiService.googleMapsService.autocomplete(req),
  }),
});
