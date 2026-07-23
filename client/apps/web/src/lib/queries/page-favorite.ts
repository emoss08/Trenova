import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const pageFavoite = createQueryKeys("pageFavorite", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async () => {
      return apiService.pageFavoriteService.listPageFavorites();
    },
  }),
  check: (pageUrl: string) => ({
    queryKey: ["check", pageUrl],
    queryFn: async () => {
      return apiService.pageFavoriteService.checkPageFavorite(pageUrl);
    },
  }),
});
