import {
  BulkUpdateServiceTypeStatusDocument,
  CreateServiceTypeDocument,
  PatchServiceTypeDocument,
  UpdateServiceTypeDocument,
  type ServiceTypeInput,
  type ServiceTypePatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateServiceTypeStatusRequest, ServiceType } from "@/types/service-type";

export function toServiceTypeInput(data: ServiceType): ServiceTypeInput {
  return {
    status: data.status,
    code: data.code,
    description: data.description || null,
    color: data.color || null,
    version: data.version,
  };
}

export class ServiceTypeService {
  public async create(data: ServiceType) {
    const response = await requestGraphQL({
      document: CreateServiceTypeDocument,
      operationName: "CreateServiceType",
      variables: { input: toServiceTypeInput(data) },
    });

    return response.createServiceType;
  }

  public async update(id: string, data: ServiceType) {
    const response = await requestGraphQL({
      document: UpdateServiceTypeDocument,
      operationName: "UpdateServiceType",
      variables: { id, input: toServiceTypeInput(data) },
    });

    return response.updateServiceType;
  }

  public async bulkUpdateStatus(request: BulkUpdateServiceTypeStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateServiceTypeStatusDocument,
      operationName: "BulkUpdateServiceTypeStatus",
      variables: { input: request },
    });

    return response.bulkUpdateServiceTypeStatus;
  }

  public async patch(id: string, input: ServiceTypePatchInput) {
    const response = await requestGraphQL({
      document: PatchServiceTypeDocument,
      operationName: "PatchServiceType",
      variables: { id, input },
    });

    return response.patchServiceType;
  }
}
