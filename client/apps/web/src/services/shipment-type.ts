import {
  BulkUpdateShipmentTypeStatusDocument,
  CreateShipmentTypeDocument,
  PatchShipmentTypeDocument,
  UpdateShipmentTypeDocument,
  type ShipmentTypeInput,
  type ShipmentTypePatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateShipmentTypeStatusRequest, ShipmentType } from "@/types/shipment-type";

export function toShipmentTypeInput(data: ShipmentType): ShipmentTypeInput {
  return {
    status: data.status,
    code: data.code,
    description: data.description || null,
    color: data.color || null,
    version: data.version,
  };
}

export class ShipmentTypeService {
  public async create(data: ShipmentType) {
    const response = await requestGraphQL({
      document: CreateShipmentTypeDocument,
      operationName: "CreateShipmentType",
      variables: { input: toShipmentTypeInput(data) },
    });

    return response.createShipmentType;
  }

  public async update(id: string, data: ShipmentType) {
    const response = await requestGraphQL({
      document: UpdateShipmentTypeDocument,
      operationName: "UpdateShipmentType",
      variables: { id, input: toShipmentTypeInput(data) },
    });

    return response.updateShipmentType;
  }

  public async bulkUpdateStatus(request: BulkUpdateShipmentTypeStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateShipmentTypeStatusDocument,
      operationName: "BulkUpdateShipmentTypeStatus",
      variables: { input: request },
    });

    return response.bulkUpdateShipmentTypeStatus;
  }

  public async patch(id: string, input: ShipmentTypePatchInput) {
    const response = await requestGraphQL({
      document: PatchShipmentTypeDocument,
      operationName: "PatchShipmentType",
      variables: { id, input },
    });

    return response.patchShipmentType;
  }
}
