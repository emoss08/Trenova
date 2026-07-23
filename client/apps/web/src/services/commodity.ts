import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateCommodityStatusResponseSchema,
  commoditySchema,
  type BulkUpdateCommodityStatusRequest,
  type BulkUpdateCommodityStatusResponse,
  type Commodity,
} from "@/types/commodity";

export class CommodityService {
  public async bulkUpdateStatus(request: BulkUpdateCommodityStatusRequest) {
    const response = await api.post<BulkUpdateCommodityStatusResponse>(
      "/commodities/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateCommodityStatusResponseSchema, response, "Bulk Update Commodity Status");
  }

  public async patch(id: Commodity["id"], data: Partial<Commodity>) {
    const response = await api.patch<Commodity>(
      `/commodities/${id}/`,
      data,
    );

    return safeParse(commoditySchema, response, "Commodity");
  }
}
