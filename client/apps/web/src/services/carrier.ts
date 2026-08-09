import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  bulkUpdateCarrierStatusResponseSchema,
  carrierSchema,
  type BulkUpdateCarrierStatusRequest,
  type BulkUpdateCarrierStatusResponse,
  type Carrier,
} from "@trenova/shared/types/carrier";

export class CarrierService {
  public async bulkUpdateStatus(request: BulkUpdateCarrierStatusRequest) {
    const response = await api.post<BulkUpdateCarrierStatusResponse>(
      "/carriers/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateCarrierStatusResponseSchema, response, "Bulk Update Carrier Status");
  }

  public async patch(carrierId: Carrier["id"], data: Partial<Carrier>) {
    const response = await api.patch<Carrier>(`/carriers/${carrierId}/`, data);

    return safeParse(carrierSchema, response, "Carrier");
  }
}
