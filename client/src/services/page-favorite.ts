import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  checkFavoriteResponseSchema,
  toggleFavoriteResponseSchema,
  type CheckFavoriteResponse,
  type PageFavorite,
  type ToggleFavoriteRequest,
  type ToggleFavoriteResponse,
} from "@/types/page-favorite";

export class PageFavoriteService {
  public async listPageFavorites() {
    return api.get<PageFavorite[]>("/page-favorites/");
  }

  public async togglePageFavorite(req: ToggleFavoriteRequest) {
    const response = await api.post<ToggleFavoriteResponse>(
      "/page-favorites/toggle",
      req,
    );

    return safeParse(toggleFavoriteResponseSchema, response, "ToggleFavoriteResponse");
  }

  public async checkPageFavorite(pageUrl: string) {
    const params = new URLSearchParams({ pageUrl });
    const response = await api.get<CheckFavoriteResponse>(
      `/page-favorites/check?${params}`,
    );

    return safeParse(checkFavoriteResponseSchema, response, "CheckFavoriteResponse");
  }
}
