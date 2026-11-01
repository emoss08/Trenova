import {
  BulkUpdateTrailerStatusDocument,
  CreateTrailerDocument,
  LocateTrailerDocument,
  PatchTrailerDocument,
  UpdateTrailerDocument,
  type TrailerInput,
  type TrailerPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  BulkUpdateTrailerStatusRequest,
  LocateTrailerPayload,
  Trailer,
} from "@/types/trailer";

export function toTrailerInput(data: Trailer): TrailerInput {
  return {
    equipmentTypeId: data.equipmentTypeId,
    equipmentManufacturerId: data.equipmentManufacturerId,
    registrationStateId: data.registrationStateId ?? null,
    fleetCodeId: data.fleetCodeId ?? null,
    status: data.status,
    code: data.code,
    model: data.model || null,
    make: data.make || null,
    year: data.year,
    licensePlateNumber: data.licensePlateNumber || null,
    vin: data.vin || null,
    externalId: data.externalId || null,
    registrationNumber: data.registrationNumber || null,
    maxLoadWeight: data.maxLoadWeight ?? null,
    lastInspectionDate: data.lastInspectionDate ?? null,
    registrationExpiry: data.registrationExpiry ?? null,
    version: data.version,
    customFields: data.customFields ?? null,
  };
}

export class TrailerService {
  public async create(data: Trailer) {
    const response = await requestGraphQL({
      document: CreateTrailerDocument,
      operationName: "CreateTrailer",
      variables: { input: toTrailerInput(data) },
    });

    return response.createTrailer;
  }

  public async update(id: string, data: Trailer) {
    const response = await requestGraphQL({
      document: UpdateTrailerDocument,
      operationName: "UpdateTrailer",
      variables: { id, input: toTrailerInput(data) },
    });

    return response.updateTrailer;
  }

  public async patch(id: string, input: TrailerPatchInput) {
    const response = await requestGraphQL({
      document: PatchTrailerDocument,
      operationName: "PatchTrailer",
      variables: { id, input },
    });

    return response.patchTrailer;
  }

  public async bulkUpdateStatus(request: BulkUpdateTrailerStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateTrailerStatusDocument,
      operationName: "BulkUpdateTrailerStatus",
      variables: { input: request },
    });

    return response.bulkUpdateTrailerStatus;
  }

  public async locate(trailerId: string, data: LocateTrailerPayload) {
    const response = await requestGraphQL({
      document: LocateTrailerDocument,
      operationName: "LocateTrailer",
      variables: { input: { trailerId, newLocationId: data.newLocationId } },
    });

    return response.locateTrailer;
  }
}
