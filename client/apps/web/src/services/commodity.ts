import {
  BulkUpdateCommodityStatusDocument,
  CreateCommodityDocument,
  PatchCommodityDocument,
  UpdateCommodityDocument,
  type CommodityInput,
  type CommodityPatchInput,
  type FreightClass,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BulkUpdateCommodityStatusRequest, Commodity } from "@trenova/shared/types/commodity";

export function toCommodityInput(data: Commodity): CommodityInput {
  return {
    status: data.status,
    name: data.name,
    description: data.description,
    hazardousMaterialId: data.hazardousMaterialId ?? null,
    minTemperature: data.minTemperature ?? null,
    maxTemperature: data.maxTemperature ?? null,
    weightPerUnit: data.weightPerUnit ?? null,
    linearFeetPerUnit: data.linearFeetPerUnit ?? null,
    lengthPerUnit: data.lengthPerUnit ?? null,
    widthPerUnit: data.widthPerUnit ?? null,
    heightPerUnit: data.heightPerUnit ?? null,
    maxQuantityPerShipment: data.maxQuantityPerShipment ?? null,
    freightClass: (data.freightClass as FreightClass | null | undefined) ?? null,
    loadingInstructions: data.loadingInstructions || null,
    stackable: data.stackable,
    fragile: data.fragile,
    version: data.version,
  };
}

export class CommodityService {
  public async create(data: Commodity) {
    const response = await requestGraphQL({
      document: CreateCommodityDocument,
      operationName: "CreateCommodity",
      variables: { input: toCommodityInput(data) },
    });

    return response.createCommodity;
  }

  public async update(id: string, data: Commodity) {
    const response = await requestGraphQL({
      document: UpdateCommodityDocument,
      operationName: "UpdateCommodity",
      variables: { id, input: toCommodityInput(data) },
    });

    return response.updateCommodity;
  }

  public async bulkUpdateStatus(request: BulkUpdateCommodityStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateCommodityStatusDocument,
      operationName: "BulkUpdateCommodityStatus",
      variables: { input: request },
    });

    return response.bulkUpdateCommodityStatus;
  }

  public async patch(id: string, input: CommodityPatchInput) {
    const response = await requestGraphQL({
      document: PatchCommodityDocument,
      operationName: "PatchCommodity",
      variables: { id, input },
    });

    return response.patchCommodity;
  }
}

export const commodityService = new CommodityService();
