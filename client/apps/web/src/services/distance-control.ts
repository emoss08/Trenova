import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { distanceControlSchema, type DistanceControl } from "@/types/distance-control";

export class DistanceControlService {
  public async get() {
    const response = await api.get<DistanceControl>("/distance-controls/");
    return safeParse(distanceControlSchema, response, "Distance Control");
  }

  public async patch(data: Partial<DistanceControl>) {
    const response = await api.patch<DistanceControl>("/distance-controls/", data);
    return safeParse(distanceControlSchema, response, "Distance Control");
  }
}
