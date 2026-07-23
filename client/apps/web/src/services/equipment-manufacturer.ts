import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateEquipmentManufacturerStatusResponseSchema,
  equipmentManufacturerSchema,
  type BulkUpdateEquipmentManufacturerStatusRequest,
  type BulkUpdateEquipmentManufacturerStatusResponse,
  type EquipmentManufacturer,
} from "@/types/equipment-manufacturer";

export class EquipmentManufacturerService {
  public async bulkUpdateStatus(
    request: BulkUpdateEquipmentManufacturerStatusRequest,
  ) {
    const response =
      await api.post<BulkUpdateEquipmentManufacturerStatusResponse>(
        "/equipment-manufacturers/bulk-update-status",
        request,
      );

    return safeParse(bulkUpdateEquipmentManufacturerStatusResponseSchema, response, "BulkUpdateEquipmentManufacturerStatus");
  }

  public async patch(
    id: EquipmentManufacturer["id"],
    data: Partial<EquipmentManufacturer>,
  ) {
    const response = await api.patch<EquipmentManufacturer>(
      `/equipment-manufacturers/${id}`,
      data,
    );

    return safeParse(equipmentManufacturerSchema, response, "EquipmentManufacturer");
  }
}
