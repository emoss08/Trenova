import { pageFavoriteService } from "@/services/page-favorite";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const pageFavorite = createQueryKeys("pageFavorite", {
  all: () => ({
    queryKey: ["all"],
    queryFn: async () => {
      return pageFavoriteService.listPageFavorites();
    },
  }),
  check: (pageUrl: string) => ({
    queryKey: ["check", pageUrl],
    queryFn: async () => {
      return pageFavoriteService.checkPageFavorite(pageUrl);
    },
  }),
});
