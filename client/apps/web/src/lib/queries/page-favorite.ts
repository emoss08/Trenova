import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const pageFavorite = createQueryKeys("pageFavorite", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      return apiService.pageFavoriteService.listPageFavorites({ signal });
    },
  }),
  check: (pageUrl: string) => ({
    queryKey: ["check", pageUrl],
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      return apiService.pageFavoriteService.checkPageFavorite(pageUrl, { signal });
    },
  }),
});
