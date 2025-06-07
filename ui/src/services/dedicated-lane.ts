import { http } from "@/lib/http-client";
import { DedicatedLaneSchema } from "@/lib/schemas/dedicated-lane-schema";

export type GetDedicatedLaneByShipmentRequest = {
  customerId: string;
  serviceTypeId: string;
  shipmentTypeId: string;
  originLocationId?: string | null;
  destinationLocationId?: string | null;
  trailerTypeId?: string | null;
  tractorTypeId?: string | null;
};

export class DedicatedLaneAPI {
  async getByShipment(req: GetDedicatedLaneByShipmentRequest) {
    const response = await http.post<DedicatedLaneSchema | null>(
      "/dedicated-lanes/find-by-shipment",
      req,
    );

    return response.data;
  }
}
