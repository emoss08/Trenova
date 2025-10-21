/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import {
  FavoriteSchema,
  ToggleFavoriteSchema,
} from "@/lib/schemas/favorite-schema";
import { QueryOptions } from "@/types/common";
import { LimitOffsetResponse } from "@/types/server";

type CheckFavoriteResponse = {
  isFavorite: boolean;
  favorite: FavoriteSchema | null;
};

type ToggleFavoriteResponse = {
  action: "added" | "removed";
  favorite?: FavoriteSchema | null;
};

export class FavoriteAPI {
  async list(req?: QueryOptions) {
    const response = await http.get<LimitOffsetResponse<FavoriteSchema>>(
      "/favorites/",
      {
        params: req
          ? {
              limit: req.limit?.toString(),
              offset: req.offset?.toString(),
              query: req.query,
              filters: req.filters,
              sort: req.sort,
            }
          : undefined,
      },
    );
    return response.data;
  }

  async delete(favoriteId: FavoriteSchema["id"]) {
    await http.delete(`/favorites/${favoriteId}/`);
  }

  async toggle(request: ToggleFavoriteSchema) {
    const response = await http.post<ToggleFavoriteResponse>(
      "/favorites/toggle/",
      request,
    );
    return response.data;
  }

  async checkFavorite(pageUrl: string) {
    const response = await http.post<CheckFavoriteResponse>(
      "/favorites/check/",
      { pageUrl },
    );
    return response.data;
  }
}
