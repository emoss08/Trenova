import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateShipmentTypeStatusResponseSchema,
  shipmentTypeSchema,
  type BulkUpdateShipmentTypeStatusRequest,
  type BulkUpdateShipmentTypeStatusResponse,
  type ShipmentType,
} from "@/types/shipment-type";

export class ShipmentTypeService {
  public async bulkUpdateStatus(request: BulkUpdateShipmentTypeStatusRequest) {
    const response = await api.post<BulkUpdateShipmentTypeStatusResponse>(
      "/shipment-types/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateShipmentTypeStatusResponseSchema, response, "Bulk Update Shipment Type Status");
  }

  public async patch(id: ShipmentType["id"], data: Partial<ShipmentType>) {
    const response = await api.patch<ShipmentType>(
      `/shipment-types/${id}/`,
      data,
    );

    return safeParse(shipmentTypeSchema, response, "Shipment Type");
  }
}
