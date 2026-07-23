import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateHazardousMaterialStatusResponseSchema,
  hazardousMaterialSchema,
  type BulkUpdateHazardousMaterialStatusRequest,
  type BulkUpdateHazardousMaterialStatusResponse,
  type HazardousMaterial,
} from "@/types/hazardous-material";

export class HazardousMaterialService {
  public async bulkUpdateStatus(request: BulkUpdateHazardousMaterialStatusRequest) {
    const response = await api.post<BulkUpdateHazardousMaterialStatusResponse>(
      "/hazardous-materials/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateHazardousMaterialStatusResponseSchema, response, "BulkUpdateHazardousMaterialStatus");
  }

  public async patch(id: HazardousMaterial["id"], data: Partial<HazardousMaterial>) {
    const response = await api.patch<HazardousMaterial>(
      `/hazardous-materials/${id}/`,
      data,
    );

    return safeParse(hazardousMaterialSchema, response, "HazardousMaterial");
  }
}
