import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateEquipmentTypeStatusResponseSchema,
  equipmentTypeSchema,
  type BulkUpdateEquipmentTypeStatusRequest,
  type BulkUpdateEquipmentTypeStatusResponse,
  type EquipmentType,
} from "@/types/equipment-type";

export class EquipmentTypeService {
  public async bulkUpdateStatus(request: BulkUpdateEquipmentTypeStatusRequest) {
    const response = await api.post<BulkUpdateEquipmentTypeStatusResponse>(
      "/equipment-types/bulk-update-status",
      request,
    );

    return safeParse(bulkUpdateEquipmentTypeStatusResponseSchema, response, "BulkUpdateEquipmentTypeStatus");
  }

  public async patch(id: EquipmentType["id"], data: Partial<EquipmentType>) {
    const response = await api.patch<EquipmentType>(
      `/equipment-types/${id}`,
      data,
    );

    return safeParse(equipmentTypeSchema, response, "EquipmentType");
  }
}
