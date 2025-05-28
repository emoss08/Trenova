import { http } from "@/lib/http-client";
import type {
  CheckFavoriteResponse,
  CreateFavoriteRequest,
  Favorite,
  ToggleFavoriteRequest,
  ToggleFavoriteResponse,
  UpdateFavoriteRequest,
} from "@/types/favorite";

export class FavoriteAPI {
  async list() {
    const response = await http.get<{ results: Favorite[] }>("/favorites/");
    return response.data.results;
  }

  async get(favoriteId: string) {
    const response = await http.get<Favorite>(`/favorites/${favoriteId}/`);
    return response.data;
  }

  async create(request: CreateFavoriteRequest) {
    const response = await http.post<Favorite>("/favorites/", request);
    return response.data;
  }

  async update(favoriteId: string, request: UpdateFavoriteRequest) {
    const response = await http.put<Favorite>(
      `/favorites/${favoriteId}/`,
      request,
    );
    return response.data;
  }

  async delete(favoriteId: string) {
    await http.delete(`/favorites/${favoriteId}/`);
  }

  async toggle(request: ToggleFavoriteRequest) {
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

  async isFavorite(pageUrl: string): Promise<boolean> {
    try {
      const result = await this.checkFavorite(pageUrl);
      return result.isFavorite;
    } catch {
      return false;
    }
  }
}
