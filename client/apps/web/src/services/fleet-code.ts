import {
  BulkUpdateFleetCodeStatusDocument,
  CreateFleetCodeDocument,
  PatchFleetCodeDocument,
  UpdateFleetCodeDocument,
  type FleetCodeInput,
  type FleetCodePatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateFleetCodeStatusRequest, FleetCode } from "@trenova/shared/types/fleet-code";

export function toFleetCodeInput(data: FleetCode): FleetCodeInput {
  return {
    status: data.status,
    code: data.code,
    description: data.description || null,
    managerId: data.managerId,
    revenueGoal: data.revenueGoal ?? null,
    deadheadGoal: data.deadheadGoal ?? null,
    mileageGoal: data.mileageGoal ?? null,
    color: data.color || null,
    version: data.version,
  };
}

export class FleetCodeService {
  public async create(data: FleetCode) {
    const response = await requestGraphQL({
      document: CreateFleetCodeDocument,
      operationName: "CreateFleetCode",
      variables: { input: toFleetCodeInput(data) },
    });

    return response.createFleetCode;
  }

  public async update(id: string, data: FleetCode) {
    const response = await requestGraphQL({
      document: UpdateFleetCodeDocument,
      operationName: "UpdateFleetCode",
      variables: { id, input: toFleetCodeInput(data) },
    });

    return response.updateFleetCode;
  }

  public async bulkUpdateStatus(request: BulkUpdateFleetCodeStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateFleetCodeStatusDocument,
      operationName: "BulkUpdateFleetCodeStatus",
      variables: { input: request },
    });

    return response.bulkUpdateFleetCodeStatus;
  }

  public async patch(id: string, input: FleetCodePatchInput) {
    const response = await requestGraphQL({
      document: PatchFleetCodeDocument,
      operationName: "PatchFleetCode",
      variables: { id, input },
    });

    return response.patchFleetCode;
  }
}
