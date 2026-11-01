import {
  BulkUpdateHazardousMaterialStatusDocument,
  CreateHazardousMaterialDocument,
  PatchHazardousMaterialDocument,
  UpdateHazardousMaterialDocument,
  type HazardousMaterialInput,
  type HazardousMaterialPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  BulkUpdateHazardousMaterialStatusRequest,
  HazardousMaterial,
} from "@/types/hazardous-material";

export function toHazardousMaterialInput(data: HazardousMaterial): HazardousMaterialInput {
  return {
    status: data.status,
    code: data.code || null,
    name: data.name,
    description: data.description || null,
    class: data.class,
    unNumber: data.unNumber || null,
    packingGroup: data.packingGroup ?? null,
    subsidiaryHazardClass: data.subsidiaryHazardClass || null,
    ergGuideNumber: data.ergGuideNumber || null,
    labelCodes: data.labelCodes || null,
    specialProvisions: data.specialProvisions || null,
    properShippingName: data.properShippingName || null,
    handlingInstructions: data.handlingInstructions || null,
    emergencyContact: data.emergencyContact || null,
    emergencyContactPhoneNumber: data.emergencyContactPhoneNumber || null,
    quantityThreshold: data.quantityThreshold || null,
    placardRequired: data.placardRequired,
    isReportableQuantity: data.isReportableQuantity,
    marinePollutant: data.marinePollutant,
    inhalationHazard: data.inhalationHazard,
    version: data.version,
  };
}

export class HazardousMaterialService {
  public async create(data: HazardousMaterial) {
    const response = await requestGraphQL({
      document: CreateHazardousMaterialDocument,
      operationName: "CreateHazardousMaterial",
      variables: { input: toHazardousMaterialInput(data) },
    });

    return response.createHazardousMaterial;
  }

  public async update(id: string, data: HazardousMaterial) {
    const response = await requestGraphQL({
      document: UpdateHazardousMaterialDocument,
      operationName: "UpdateHazardousMaterial",
      variables: { id, input: toHazardousMaterialInput(data) },
    });

    return response.updateHazardousMaterial;
  }

  public async bulkUpdateStatus(request: BulkUpdateHazardousMaterialStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateHazardousMaterialStatusDocument,
      operationName: "BulkUpdateHazardousMaterialStatus",
      variables: { input: request },
    });

    return response.bulkUpdateHazardousMaterialStatus;
  }

  public async patch(id: string, input: HazardousMaterialPatchInput) {
    const response = await requestGraphQL({
      document: PatchHazardousMaterialDocument,
      operationName: "PatchHazardousMaterial",
      variables: { id, input },
    });

    return response.patchHazardousMaterial;
  }
}
