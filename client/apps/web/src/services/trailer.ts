import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateTrailerStatusResponseSchema,
  trailerSchema,
  type BulkUpdateTrailerStatusRequest,
  type BulkUpdateTrailerStatusResponse,
  type LocateTrailerPayload,
  type Trailer,
} from "@/types/trailer";

export class TrailerService {
  public async patch(id: Trailer["id"], data: Partial<Trailer>) {
    const response = await api.patch<Trailer>(`/trailers/${id}/`, data);

    return safeParse(trailerSchema, response, "Trailer");
  }

  public async bulkUpdateStatus(request: BulkUpdateTrailerStatusRequest) {
    const response = await api.post<BulkUpdateTrailerStatusResponse>(
      "/trailers/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateTrailerStatusResponseSchema, response, "Bulk Update Trailer Status");
  }

  public async locate(trailerId: string, data: LocateTrailerPayload) {
    return api.post(`/trailers/${trailerId}/locate/`, data);
  }

  public async getOption(id: Trailer["id"]) {
    const response = await api.get<Trailer>(`/trailers/select-options/${id}`);

    return safeParse(trailerSchema, response, "Trailer");
  }
}
