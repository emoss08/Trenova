import {
  BulkUpdateTractorStatusDocument,
  CreateTractorDocument,
  PatchTractorDocument,
  UpdateTractorDocument,
  type TractorInput,
  type TractorPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateTractorStatusRequest, Tractor } from "@/types/tractor";

export function toTractorInput(data: Tractor): TractorInput {
  return {
    primaryWorkerId: data.primaryWorkerId,
    equipmentTypeId: data.equipmentTypeId,
    equipmentManufacturerId: data.equipmentManufacturerId,
    stateId: data.stateId ?? null,
    fleetCodeId: data.fleetCodeId ?? null,
    secondaryWorkerId: data.secondaryWorkerId ?? null,
    status: data.status,
    code: data.code,
    model: data.model || null,
    make: data.make || null,
    year: data.year ?? null,
    licensePlateNumber: data.licensePlateNumber || null,
    registrationNumber: data.registrationNumber || null,
    registrationExpiry: data.registrationExpiry ?? null,
    vin: data.vin || null,
    externalId: data.externalId || null,
    fuelType: data.fuelType,
    iftaQualified: data.iftaQualified,
    version: data.version,
    customFields: data.customFields ?? null,
  };
}

export class TractorService {
  public async create(data: Tractor) {
    const response = await requestGraphQL({
      document: CreateTractorDocument,
      operationName: "CreateTractor",
      variables: { input: toTractorInput(data) },
    });

    return response.createTractor;
  }

  public async update(id: string, data: Tractor) {
    const response = await requestGraphQL({
      document: UpdateTractorDocument,
      operationName: "UpdateTractor",
      variables: { id, input: toTractorInput(data) },
    });

    return response.updateTractor;
  }

  public async patch(id: string, input: TractorPatchInput) {
    const response = await requestGraphQL({
      document: PatchTractorDocument,
      operationName: "PatchTractor",
      variables: { id, input },
    });

    return response.patchTractor;
  }

  public async bulkUpdateStatus(request: BulkUpdateTractorStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateTractorStatusDocument,
      operationName: "BulkUpdateTractorStatus",
      variables: { input: request },
    });

    return response.bulkUpdateTractorStatus;
  }
}
