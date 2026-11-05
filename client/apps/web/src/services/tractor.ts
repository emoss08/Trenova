import {
  bulkUpdateTractorStatusResponseSchema,
  tractorSchema,
  type BulkUpdateTractorStatusRequest,
  type BulkUpdateTractorStatusResponse,
  type Tractor,
} from "@/types/tractor";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";

export class TractorService {
  public async patch(id: Tractor["id"], data: Partial<Tractor>) {
    const response = await api.patch<Tractor>(`/tractors/${id}/`, data);

    return safeParse(tractorSchema, response, "Tractor");
  }

  public async bulkUpdateStatus(request: BulkUpdateTractorStatusRequest) {
    const response = await api.post<BulkUpdateTractorStatusResponse>(
      "/tractors/bulk-update-status",
      request,
    );

    return safeParse(bulkUpdateTractorStatusResponseSchema, response, "Bulk Update Tractor Status");
  }
}

export const tractorService = new TractorService();
