import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateFleetCodeStatusResponseSchema,
  fleetCodeSchema,
  type BulkUpdateFleetCodeStatusRequest,
  type BulkUpdateFleetCodeStatusResponse,
  type FleetCode,
} from "@/types/fleet-code";

export class FleetCodeService {
  public async bulkUpdateStatus(request: BulkUpdateFleetCodeStatusRequest) {
    const response = await api.post<BulkUpdateFleetCodeStatusResponse>(
      "/fleet-codes/bulk-update-status",
      request,
    );

    return safeParse(bulkUpdateFleetCodeStatusResponseSchema, response, "BulkUpdateFleetCodeStatus");
  }

  public async patch(id: FleetCode["id"], data: Partial<FleetCode>) {
    const response = await api.patch<FleetCode>(`/fleet-codes/${id}`, data);

    return safeParse(fleetCodeSchema, response, "FleetCode");
  }
}
