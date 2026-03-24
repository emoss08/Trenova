import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  shipmentControlSchema,
  type ShipmentControl,
} from "@/types/shipment-control";

export class ShipmentControlService {
  public async get() {
    const response = await api.get<ShipmentControl>("/shipment-controls/");

    return safeParse(shipmentControlSchema, response, "Shipment Control");
  }

  public async update(data: ShipmentControl) {
    const response = await api.put<ShipmentControl>(
      "/shipment-controls/",
      data,
    );

    return safeParse(shipmentControlSchema, response, "Shipment Control");
  }
}
