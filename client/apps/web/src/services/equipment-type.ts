import {
  BulkUpdateEquipmentTypeStatusDocument,
  CreateEquipmentTypeDocument,
  PatchEquipmentTypeDocument,
  UpdateEquipmentTypeDocument,
  type BulkUpdateEquipmentTypeStatusMutation,
  type CreateEquipmentTypeMutation,
  type EquipmentTypeInput,
  type PatchEquipmentTypeMutation,
  type UpdateEquipmentTypeMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateEquipmentTypeStatusResponseSchema,
  equipmentTypeSchema,
  type BulkUpdateEquipmentTypeStatusRequest,
  type EquipmentType,
} from "@/types/equipment-type";

function toEquipmentTypeInput(data: EquipmentType): EquipmentTypeInput {
  return {
    status: data.status,
    code: data.code,
    description: data.description ?? null,
    class: data.class,
    color: data.color ?? null,
    interiorLength: data.interiorLength ?? null,
    version: data.version,
  };
}

export class EquipmentTypeService {
  public async create(data: EquipmentType) {
    const response = (await requestGraphQL({
      document: CreateEquipmentTypeDocument,
      operationName: "CreateEquipmentType",
      variables: { input: toEquipmentTypeInput(data) },
    })) as CreateEquipmentTypeMutation;

    return safeParse(equipmentTypeSchema, response.createEquipmentType, "EquipmentType");
  }

  public async update(id: NonNullable<EquipmentType["id"]>, data: EquipmentType) {
    const response = (await requestGraphQL({
      document: UpdateEquipmentTypeDocument,
      operationName: "UpdateEquipmentType",
      variables: { id, input: toEquipmentTypeInput(data) },
    })) as UpdateEquipmentTypeMutation;

    return safeParse(equipmentTypeSchema, response.updateEquipmentType, "EquipmentType");
  }

  public async bulkUpdateStatus(request: BulkUpdateEquipmentTypeStatusRequest) {
    const response = (await requestGraphQL({
      document: BulkUpdateEquipmentTypeStatusDocument,
      operationName: "BulkUpdateEquipmentTypeStatus",
      variables: { input: request },
    })) as BulkUpdateEquipmentTypeStatusMutation;

    return safeParse(
      bulkUpdateEquipmentTypeStatusResponseSchema,
      response.bulkUpdateEquipmentTypeStatus,
      "BulkUpdateEquipmentTypeStatus",
    );
  }

  public async patch(id: NonNullable<EquipmentType["id"]>, data: Partial<EquipmentType>) {
    const response = (await requestGraphQL({
      document: PatchEquipmentTypeDocument,
      operationName: "PatchEquipmentType",
      variables: { id, input: data },
    })) as PatchEquipmentTypeMutation;

    return safeParse(equipmentTypeSchema, response.patchEquipmentType, "EquipmentType");
  }
}
