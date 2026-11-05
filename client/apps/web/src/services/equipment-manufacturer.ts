import {
  BulkUpdateEquipmentManufacturerStatusDocument,
  CreateEquipmentManufacturerDocument,
  PatchEquipmentManufacturerDocument,
  UpdateEquipmentManufacturerDocument,
  type EquipmentManufacturerInput,
  type EquipmentManufacturerPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  BulkUpdateEquipmentManufacturerStatusRequest,
  EquipmentManufacturer,
} from "@/types/equipment-manufacturer";

export function toEquipmentManufacturerInput(
  data: EquipmentManufacturer,
): EquipmentManufacturerInput {
  return {
    status: data.status,
    name: data.name,
    description: data.description || null,
    version: data.version,
  };
}

export class EquipmentManufacturerService {
  public async create(data: EquipmentManufacturer) {
    const response = await requestGraphQL({
      document: CreateEquipmentManufacturerDocument,
      operationName: "CreateEquipmentManufacturer",
      variables: { input: toEquipmentManufacturerInput(data) },
    });

    return response.createEquipmentManufacturer;
  }

  public async update(id: string, data: EquipmentManufacturer) {
    const response = await requestGraphQL({
      document: UpdateEquipmentManufacturerDocument,
      operationName: "UpdateEquipmentManufacturer",
      variables: { id, input: toEquipmentManufacturerInput(data) },
    });

    return response.updateEquipmentManufacturer;
  }

  public async bulkUpdateStatus(request: BulkUpdateEquipmentManufacturerStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateEquipmentManufacturerStatusDocument,
      operationName: "BulkUpdateEquipmentManufacturerStatus",
      variables: { input: request },
    });

    return response.bulkUpdateEquipmentManufacturerStatus;
  }

  public async patch(id: string, input: EquipmentManufacturerPatchInput) {
    const response = await requestGraphQL({
      document: PatchEquipmentManufacturerDocument,
      operationName: "PatchEquipmentManufacturer",
      variables: { id, input },
    });

    return response.patchEquipmentManufacturer;
  }
}

export const equipmentManufacturerService = new EquipmentManufacturerService();
