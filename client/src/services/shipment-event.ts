import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
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
    const search = new URLSearchParams();
    if (params.shipmentId) {
      search.set("shipmentId", params.shipmentId);
    }
    if (params.types && params.types.length > 0) {
      search.set("types", params.types.join(","));
    }
    if (typeof params.limit === "number") {
      search.set("limit", String(params.limit));
    }
    if (typeof params.before === "number") {
      search.set("before", String(params.before));
    }

    const query = search.toString();
    const response = await api.get<ShipmentEventList>(
      `/shipment-events/${query ? `?${query}` : ""}`,
    );

    return safeParse(shipmentEventListSchema, response, "Shipment Event");
  }
}
