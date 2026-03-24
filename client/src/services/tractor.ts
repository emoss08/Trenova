import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateTractorStatusResponseSchema,
  tractorSchema,
  type BulkUpdateTractorStatusRequest,
  type BulkUpdateTractorStatusResponse,
  type Tractor,
} from "@/types/tractor";

export class TractorService {
  public async patch(id: Tractor["id"], data: Partial<Tractor>) {
    const response = await api.patch<Tractor>(`/tractors/${id}`, data);

    return safeParse(tractorSchema, response, "Tractor");
  }

  public async bulkUpdateStatus(request: BulkUpdateTractorStatusRequest) {
    const response = await api.post<BulkUpdateTractorStatusResponse>(
      "/tractors/bulk-update-status",
      request,
    );

    return safeParse(bulkUpdateTractorStatusResponseSchema, response, "Bulk Update Tractor Status");
  }
  public async getOption(id: Tractor["id"]) {
    const response = await api.get<Worker>(`/tractors/select-options/${id}`);

    return safeParse(tractorSchema, response, "Tractor");
  }
}
