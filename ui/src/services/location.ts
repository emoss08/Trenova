import { http } from "@/lib/http-client";
import { LocationSchema } from "@/lib/schemas/location-schema";

export class LocationAPI {
  async getById(id: LocationSchema["id"]) {
    const { data } = await http.get<LocationSchema>(`/locations/${id}/`);
    return data;
  }
}
