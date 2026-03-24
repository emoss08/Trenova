import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateLocationStatusResponseSchema,
  locationSchema,
  type BulkUpdateLocationStatusRequest,
  type BulkUpdateLocationStatusResponse,
  type Location,
} from "@/types/location";

export class LocationService {
  public async bulkUpdateStatus(request: BulkUpdateLocationStatusRequest) {
    const response = await api.post<BulkUpdateLocationStatusResponse>(
      "/locations/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateLocationStatusResponseSchema, response, "BulkUpdateLocationStatus");
  }

  public async patch(id: Location["id"], data: Partial<Location>) {
    const response = await api.patch<Location>(`/locations/${id}/`, data);

    return safeParse(locationSchema, response, "Location");
  }

  public async getOption(id: Location["id"]) {
    const response = await api.get<Location>(
      `/locations/select-options/${id}`,
    );

    return safeParse(locationSchema, response, "Location");
  }
}
