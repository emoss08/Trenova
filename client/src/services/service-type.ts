import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateServiceTypeStatusResponseSchema,
  serviceTypeSchema,
  type BulkUpdateServiceTypeStatusRequest,
  type BulkUpdateServiceTypeStatusResponse,
  type ServiceType,
} from "@/types/service-type";

export class ServiceTypeService {
  public async bulkUpdateStatus(request: BulkUpdateServiceTypeStatusRequest) {
    const response = await api.post<BulkUpdateServiceTypeStatusResponse>(
      "/service-types/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateServiceTypeStatusResponseSchema, response, "BulkUpdateServiceTypeStatus");
  }

  public async patch(id: ServiceType["id"], data: Partial<ServiceType>) {
    const response = await api.patch<ServiceType>(
      `/service-types/${id}/`,
      data,
    );

    return safeParse(serviceTypeSchema, response, "ServiceType");
  }
}
