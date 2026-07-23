import { listShipmentEventsGraphQL } from "@/lib/graphql/shipment";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  shipmentEventListSchema,
  type ShipmentEventList,
  type ShipmentEventType,
} from "@/types/shipment-event";

export type ListShipmentEventsParams = {
  shipmentId?: string;
  types?: ShipmentEventType[];
  limit?: number;
  before?: number;
};

export class ShipmentEventService {
  public async list(params: ListShipmentEventsParams = {}): Promise<ShipmentEventList> {
    const response = await listShipmentEventsGraphQL(params);
    return safeParse(shipmentEventListSchema, response, "Shipment Event");
  }
}
