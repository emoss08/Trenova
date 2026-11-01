import {
  IsMyPageFavoriteDocument,
  MyPageFavoritesDocument,
  ToggleMyPageFavoriteDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  checkFavoriteResponseSchema,
  pageFavoriteSchema,
  toggleFavoriteResponseSchema,
  type ToggleFavoriteRequest,
} from "@/types/page-favorite";
import { z } from "zod";

export class PageFavoriteService {
  public async listPageFavorites(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: MyPageFavoritesDocument,
      operationName: "MyPageFavorites",
      signal: options?.signal,
    });

    return safeParse(z.array(pageFavoriteSchema), response.myPageFavorites, "PageFavorites");
  }

  public async togglePageFavorite(req: ToggleFavoriteRequest) {
    const response = await requestGraphQL({
      document: ToggleMyPageFavoriteDocument,
      operationName: "ToggleMyPageFavorite",
      variables: { input: req },
    });

    return safeParse(
      toggleFavoriteResponseSchema,
      response.toggleMyPageFavorite,
      "ToggleFavoriteResponse",
    );
  }

  public async checkPageFavorite(pageUrl: string, options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: IsMyPageFavoriteDocument,
      operationName: "IsMyPageFavorite",
      variables: { pageUrl },
      signal: options?.signal,
    });

    return safeParse(
      checkFavoriteResponseSchema,
      { favorited: response.isMyPageFavorite },
      "CheckFavoriteResponse",
    );
  }
}
