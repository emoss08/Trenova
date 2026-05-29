import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { distanceProfileSchema, type DistanceProfile } from "@/types/distance-profile";

export class DistanceProfileService {
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
