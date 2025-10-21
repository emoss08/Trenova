import { http } from "@/lib/http-client";
import { DistanceOverrideSchema } from "@/lib/schemas/distance-override-schema";

export class DistanceOverrideAPI {
  async delete(id: DistanceOverrideSchema["id"]) {
    await http.delete(`/distance-overrides/${id}/`);
  }
}
