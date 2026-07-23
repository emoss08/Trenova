import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  distanceOverrideSchema,
  type DistanceOverride,
} from "@/types/distance-override";

export class DistanceOverrideService {
  public async patch(id: DistanceOverride["id"], data: Partial<DistanceOverride>) {
    const response = await api.patch<DistanceOverride>(
      `/distance-overrides/${id}/`,
      data,
    );

    return safeParse(distanceOverrideSchema, response, "Distance Override");
  }

  public async delete(id: DistanceOverride["id"]) {
    await api.delete(`/distance-overrides/${id}/`);
  }
}
