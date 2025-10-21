import { http } from "@/lib/http-client";
import type { ShipmentControlSchema } from "@/lib/schemas/shipmentcontrol-schema";

export class ShipmentControlAPI {
  async get() {
    const response = await http.get<ShipmentControlSchema>(
      "/shipment-controls/",
    );
    return response.data;
  }

  async update(data: ShipmentControlSchema) {
    const response = await http.put<ShipmentControlSchema>(
      `/shipment-controls/`,
      data,
    );
    return response.data;
  }
}
