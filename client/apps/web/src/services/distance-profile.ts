import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { distanceProfileSchema, type DistanceProfile } from "@/types/distance-profile";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";

export class DistanceProfileService {
  public async selectOptions(query = "", limit = 50, offset = 0) {
    const response = await api.get<GenericLimitOffsetResponse<DistanceProfile>>(
      `/distance-profiles/select-options/?query=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`,
    );
    return response;
  }

  public async getOption(id: DistanceProfile["id"]) {
    const response = await api.get<DistanceProfile>(`/distance-profiles/select-options/${id}`);
    return safeParse(distanceProfileSchema, response, "Distance Profile");
  }

  public async patch(id: DistanceProfile["id"], data: Partial<DistanceProfile>) {
    const response = await api.patch<DistanceProfile>(`/distance-profiles/${id}/`, data);
    return safeParse(distanceProfileSchema, response, "Distance Profile");
  }

  public async delete(id: DistanceProfile["id"]) {
    await api.delete(`/distance-profiles/${id}/`);
  }

  public async setDefault(id: DistanceProfile["id"]) {
    const response = await api.post<DistanceProfile>(`/distance-profiles/${id}/set-default/`);
    return safeParse(distanceProfileSchema, response, "Distance Profile");
  }
}
